package repository

import (
	"context"
	"payment-service/internal/model"

	"github.com/jmoiron/sqlx"
)

type OutboxRepo struct {
	db *sqlx.DB
}

func NewOutboxRepo(db *sqlx.DB) *OutboxRepo {
	return &OutboxRepo{db: db}
}

func (r *OutboxRepo) FetchUnpublished(ctx context.Context, limit int) ([]model.OutboxEntry, error) {
	var entries []model.OutboxEntry

	err := r.db.SelectContext(ctx, &entries, `
		SELECT id, topic, event_type, payload, published, created_at
		FROM outbox
		WHERE published = false
		ORDER BY id
		LIMIT $1
	`, limit)

	return entries, err
}

func (r *OutboxRepo) MarkPublished(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE outbox SET published = true WHERE id = $1`, id)
	return err
}
