package http

import (
	"fmt"
	"log/slog"
	"payment-service/internal/tracing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func RequestID() fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		c.Set("X-Request-ID", id)
		c.Locals("request_id", id)
		return c.Next()
	}
}

func Logger() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		slog.Info("http request",
			"request_id", c.Locals("request_id"),
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"duration", time.Since(start).String(),
		)

		return err
	}
}

func Tracing() fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx, span := tracing.Tracer().Start(c.Context(),
			fmt.Sprintf("%s %s", c.Method(), c.Path()),
			trace.WithAttributes(
				attribute.String("http.method", c.Method()),
				attribute.String("http.path", c.Path()),
				attribute.String("request_id", fmt.Sprintf("%v", c.Locals("request_id"))),
			),
		)
		defer span.End()

		c.SetContext(ctx)

		err := c.Next()

		span.SetAttributes(attribute.Int("http.status_code", c.Response().StatusCode()))

		return err
	}
}

func Recover() fiber.Handler {
	return func(c fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered",
					"request_id", c.Locals("request_id"),
					"error", r,
					"path", c.Path(),
				)
				err = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "internal server error",
				})
			}
		}()
		return c.Next()
	}
}
