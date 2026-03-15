package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"payment-service/internal/model"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PaymentRepo struct {
	db *sqlx.DB
}

func NewPaymentRepo(db *sqlx.DB) *PaymentRepo {
	return &PaymentRepo{db: db}
}

func (r *PaymentRepo) Create(ctx context.Context, p *model.Payment) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.NamedExecContext(ctx, `
		INSERT INTO payments (id, user_id, type, amount, currency, status)
		VALUES (:id, :user_id, :type, :amount, :currency, :status)
		ON CONFLICT (id) DO NOTHING
	`, p)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}

	payload, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO outbox (topic, event_type, payload)
		VALUES ($1, $2, $3)
	`, "PaymentService", "payment.created", payload)
	if err != nil {
		return fmt.Errorf("insert outbox: %w", err)
	}

	return tx.Commit()
}

func (r *PaymentRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.PaymentStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE payments SET status=$1 WHERE id=$2`,
		status, id)
	return err
}

func (r *PaymentRepo) Get(ctx context.Context, userID uuid.UUID) (*[]model.Payment, error) {
	var payments []model.Payment

	query := `
		SELECT id, user_id, type, amount, currency, status, created_at
		FROM payments
		WHERE user_id=$1
	`

	err := r.db.SelectContext(ctx, &payments, query, userID)
	if err != nil {
		return nil, err
	}
	return &payments, nil
}

func (r *PaymentRepo) GetById(ctx context.Context, id uuid.UUID) (*model.Payment, error) {
	var payment model.Payment

	query := `
		SELECT id, user_id, type, amount, currency, status, created_at
		FROM payments
		WHERE id=$1
	`
	err := r.db.GetContext(ctx, &payment, query, id)
	if err != nil {
		return nil, err
	}
	return &payment, nil
}
