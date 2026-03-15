package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"payment-service/internal/config"
	httpPaymentHandler "payment-service/internal/http"
	"payment-service/internal/kafka"
	"payment-service/internal/repository"
	"payment-service/internal/service"
	"syscall"

	"github.com/gofiber/fiber/v3"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func main() {
	cfg := config.Load()

	if cfg.LogFormat == "json" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	}

	db, err := sqlx.Connect("pgx", cfg.DBUrl)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}

	producer := kafka.NewProducer(cfg.KafkaBrokers)
	repository := repository.NewPaymentRepo(db)
	svc := service.NewPaymentService(repository, producer)
	handler := httpPaymentHandler.NewHttpPaymentHandler(svc)
	consumer, err := kafka.NewConsumer(cfg.KafkaBrokers, svc)

	if err != nil {
		slog.Error("failed to create kafka consumer", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go consumer.Start(ctx)

	app := fiber.New()

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

	if err := app.Shutdown(); err != nil {
		slog.Error("http shutdown error", "err", err)
	}
}
