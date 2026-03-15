package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"payment-service/internal/model"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Producer struct {
	client *kgo.Client
}

func NewProducer(brokers []string) *Producer {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
	)
	if err != nil {
		panic(err)
	}

	return &Producer{client: client}
}

func (p *Producer) PublishToDLQ(ctx context.Context, originalRecord []byte, errMsg string) error {
	msgCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	record := &kgo.Record{
		Topic: "PaymentService.dlq",
		Value: originalRecord,
		Headers: []kgo.RecordHeader{
			{Key: "error", Value: []byte(errMsg)},
		},
	}

	if err := p.client.ProduceSync(msgCtx, record).FirstErr(); err != nil {
		slog.Error("failed to publish to DLQ", "err", err)
		return err
	}

	return nil
}

func (p *Producer) PublishPaymentCreated(ctx context.Context, payment *model.Payment) error {
	envelope := map[string]any{
		"eventType": "payment.created",
		"payload":   payment,
	}

	b, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	msgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	record := &kgo.Record{
		Topic: "PaymentService",
		Value: b,
	}

	if err := p.client.ProduceSync(msgCtx, record).FirstErr(); err != nil {
		slog.Error("kafka produce error", "err", err, "payment_id", payment.ID)
		return err
	}

	return nil
}
