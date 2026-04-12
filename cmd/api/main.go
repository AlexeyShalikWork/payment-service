package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"payment-service/internal/config"
	"time"
	httpPaymentHandler "payment-service/internal/http"
	"payment-service/internal/kafka"
	"payment-service/internal/repository"
	"payment-service/internal/service"
	"payment-service/internal/tracing"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func main() {
	cfg := config.Load()

	if cfg.LogFormat == "json" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	}

	tp, err := tracing.Init("payment-service")
	if err != nil {
		slog.Error("failed to init tracing", "err", err)
		os.Exit(1)
	}
	defer tp.Shutdown(context.Background())

	m, err := migrate.New("file://"+cfg.MigrationsDir, cfg.DBUrl)
	if err != nil {
		slog.Error("failed to init migrations", "err", err)
		os.Exit(1)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		slog.Error("failed to run migrations", "err", err)
		os.Exit(1)
	}

	slog.Info("migrations applied")

	db, err := sqlx.Connect("pgx", cfg.DBUrl)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}

	producer := kafka.NewProducer(cfg.KafkaBrokers)
	paymentRepo := repository.NewPaymentRepo(db)
	outboxRepo := repository.NewOutboxRepo(db)
	svc := service.NewPaymentService(paymentRepo)
	handler := httpPaymentHandler.NewHttpPaymentHandler(svc)
	consumer, err := kafka.NewConsumer(cfg.KafkaBrokers, svc, producer)

	if err != nil {
		slog.Error("failed to create kafka consumer", "err", err)
		os.Exit(1)
	}

	outboxPublisher := kafka.NewOutboxPublisher(outboxRepo, producer, 2*time.Second)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go consumer.Start(ctx)
	go outboxPublisher.Start(ctx)

	app := fiber.New()

	app.Use(httpPaymentHandler.Recover())
	app.Use(httpPaymentHandler.RequestID())
	app.Use(httpPaymentHandler.Tracing())
	app.Use(httpPaymentHandler.Logger())

	// Health check — используется Kubernetes для liveness/readiness probes.
	// Возвращает 200 OK, если процесс жив и способен обрабатывать HTTP.
	app.Get("/payment/health", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	app.Get("/payments/:user_id", handler.GetPayments)
	app.Get("/payment/:id", handler.GetPayment)

	go func() {
		slog.Info("payment service started", "port", cfg.HTTPPort)
		if err := app.Listen(":" + cfg.HTTPPort); err != nil {
			slog.Error("http server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		slog.Error("http shutdown error", "err", err)
	}

	db.Close()
}
