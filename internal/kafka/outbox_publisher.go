package kafka

import (
	"context"
	"log/slog"
	"payment-service/internal/repository"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

type OutboxPublisher struct {
	repo     *repository.OutboxRepo
	producer *Producer
	interval time.Duration
}

func NewOutboxPublisher(repo *repository.OutboxRepo, producer *Producer, interval time.Duration) *OutboxPublisher {
	return &OutboxPublisher{
		repo:     repo,
		producer: producer,
		interval: interval,
	}
}

func (p *OutboxPublisher) Start(ctx context.Context) {
	slog.Info("outbox publisher started", "interval", p.interval)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("outbox publisher stopped")
			return
		case <-ticker.C:
			p.publishBatch(ctx)
		}
	}
}

func (p *OutboxPublisher) publishBatch(ctx context.Context) {
	entries, err := p.repo.FetchUnpublished(ctx, 100)
	if err != nil {
		slog.Error("outbox: failed to fetch entries", "err", err)
		return
	}

	for _, entry := range entries {
		value := []byte(`{"eventType":"` + entry.EventType + `","payload":` + string(entry.Payload) + `}`)

		record := &kgo.Record{
			Topic: entry.Topic,
			Value: value,
		}

		InjectTraceContext(ctx, record)

		msgCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := p.producer.client.ProduceSync(msgCtx, record).FirstErr()
		cancel()

		if err != nil {
			slog.Error("outbox: failed to publish", "err", err, "outbox_id", entry.ID)
			return
		}

		if err := p.repo.MarkPublished(ctx, entry.ID); err != nil {
			slog.Error("outbox: failed to mark published", "err", err, "outbox_id", entry.ID)
			return
		}

		slog.Debug("outbox: published", "outbox_id", entry.ID, "event_type", entry.EventType)
	}
}
