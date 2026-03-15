package events

import (
	"payment-service/internal/model"

	"github.com/google/uuid"
)

const PaymentCreateEventType = "payment.create"

type PaymentCreatePayload struct {
	PaymentID uuid.UUID         `json:"id"`
	UserID    uuid.UUID         `json:"userId"`
	Type      model.PaymentType `json:"type"`
	Amount    int64             `json:"amount"`
	Currency  model.Currency    `json:"currency"`
}
