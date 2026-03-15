package model

import (
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	StatusCreated    PaymentStatus = "created"
	StatusProcessing PaymentStatus = "processing"
	StatusSucceeded  PaymentStatus = "succeeded"
	StatusFailed     PaymentStatus = "failed"
)

type Currency string

const (
	USD Currency = "USD"
	EUR Currency = "EUR"
	RUB Currency = "RUB"
)

type PaymentType string

const (
	TypeCharge PaymentType = "charge"
	TypeRefund PaymentType = "refund"
	TypePayout PaymentType = "payout"
)

type Payment struct {
	ID        uuid.UUID     `db:"id" json:"id"`
	UserID    uuid.UUID     `db:"user_id" json:"userId"`
	Type      PaymentType   `db:"type" json:"type"`
	Amount    int64         `db:"amount" json:"amount"`
	Currency  Currency      `db:"currency" json:"currency"`
	Status    PaymentStatus `db:"status" json:"status"`
	CreatedAt time.Time     `db:"created_at" json:"createdAt"`
}
