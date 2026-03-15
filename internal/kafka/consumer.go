package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"payment-service/internal/kafka/events"
	"payment-service/internal/ports"
	"payment-service/internal/tracing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const maxRetries = 3

type Consumer struct {
	client   *kgo.Client
	svc      ports.PaymentService
	producer *Producer
}

func NewConsumer(brokers []string, svc ports.PaymentService, producer *Producer) (*Consumer, error) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumeTopics("PaymentService"),
		kgo.ConsumerGroup("payment.create"),
	)

	if err != nil {
		return nil, err
	}

	return &Consumer{client: cl, svc: svc, producer: producer}, nil
}

func (c *Consumer) Start(ctx context.Context) {
	defer c.client.Close()

	slog.Info("kafka consumer started")

	for {
		fetches := c.client.PollFetches(ctx)

		if ctx.Err() != nil {
			return
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			slog.Error("kafka poll errors", "errors", errs)
			continue
		}

		fetches.EachRecord(func(rec *kgo.Record) {
			c.handleWithRetry(ctx, rec)
		})
	}
}

func (c *Consumer) handleWithRetry(ctx context.Context, rec *kgo.Record) {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		msgCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		lastErr = c.processMessage(msgCtx, rec)
		cancel()

		if lastErr == nil {
			return
		}

		slog.Warn("message processing failed, retrying",
			"attempt", attempt,
			"max_retries", maxRetries,
			"err", lastErr,
			"topic", rec.Topic,
			"offset", rec.Offset,
		)

		select {
		case <-time.After(time.Duration(attempt) * time.Second):
		case <-ctx.Done():
			return
		}
	}

	slog.Error("message failed after all retries, sending to DLQ",
		"err", lastErr,
		"topic", rec.Topic,
		"offset", rec.Offset,
	)

	if err := c.producer.PublishToDLQ(ctx, rec.Value, lastErr.Error()); err != nil {
		slog.Error("failed to send to DLQ", "err", err)
	}
}

func (c *Consumer) processMessage(ctx context.Context, rec *kgo.Record) error {
	ctx = ExtractTraceContext(ctx, rec)

	ctx, span := tracing.Tracer().Start(ctx, "kafka.consume",
		trace.WithAttributes(
			attribute.String("kafka.topic", rec.Topic),
			attribute.Int64("kafka.offset", rec.Offset),
		),
	)
	defer span.End()

	var base struct {
		EventType string          `json:"eventType"`
		Payload   json.RawMessage `json:"payload"`
	}

	if err := json.Unmarshal(rec.Value, &base); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("envelope unmarshal error: %w", err)
	}

	span.SetAttributes(attribute.String("event.type", base.EventType))

	switch base.EventType {
	case events.PaymentCreateEventType:
		var payload events.PaymentCreatePayload

		if err := json.Unmarshal(base.Payload, &payload); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("Unmarshal payload error: %w", err)
		}

		return c.svc.CreatePayment(ctx, payload)
	default:
		err := fmt.Errorf("unknown event type: %s", base.EventType)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
}
