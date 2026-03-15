package repository

import (
	"context"

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
	query := `
        INSERT INTO payments (id, user_id, type, amount, currency, status)
        VALUES (:id, :user_id, :type, :amount, :currency, :status)
    `
	_, err := r.db.NamedExecContext(ctx, query, p)
	return err
}

func (r *PaymentRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.PaymentStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE payments SET status=$1 WHERE id=$2`,
		status, id)
	return err
}

func (r *PaymentRepo) Get(ctx context.Context, user_id uuid.UUID) (*[]model.Payment, error) {
	var payments []model.Payment

	query := `
        SELECT id, user_id, type, amount, currency, status, created_at
		FROM payments
		WHERE user_id=$1
    `

	err := r.db.SelectContext(ctx, &payments, query, user_id)

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
