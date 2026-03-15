package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"payment-service/internal/kafka/events"
	"payment-service/internal/ports"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Consumer struct {
	client *kgo.Client
	svc    ports.PaymentService
}

func NewConsumer(brokers []string, svc ports.PaymentService) (*Consumer, error) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumeTopics("PaymentService"),
		kgo.ConsumerGroup("payment.create"),
	)

	if err != nil {
		return nil, err
	}

	return &Consumer{client: cl, svc: svc}, nil
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
			msgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := c.processMessage(msgCtx, rec); err != nil {
				slog.Error("failed to process message", "err", err, "topic", rec.Topic, "offset", rec.Offset)
			}
		})
	}
}

func (c *Consumer) processMessage(ctx context.Context, rec *kgo.Record) error {
	var base struct {
		EventType string          `json:"eventType"`
		Payload   json.RawMessage `json:"payload"`
	}

	if err := json.Unmarshal(rec.Value, &base); err != nil {
		return fmt.Errorf("envelope unmarshal error: %w", err)
	}

	switch base.EventType {
	case events.PaymentCreateEventType:
		var payload events.PaymentCreatePayload

		if err := json.Unmarshal(base.Payload, &payload); err != nil {
			return fmt.Errorf("Unmarshal payload error: %w", err)
		}

		return c.svc.CreatePayment(ctx, payload)
	default:
		return fmt.Errorf("unknown event type: %s", base.EventType)
	}
}
