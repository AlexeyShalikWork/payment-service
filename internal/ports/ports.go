package ports

import (
	"context"

	"payment-service/internal/kafka/events"
)

type PaymentService interface {
	CreatePayment(ctx context.Context, cmd events.PaymentCreatePayload) error
}
