package ports

import (
	"context"

	"payment-service/internal/kafka/events"
	"payment-service/internal/model"

	"github.com/google/uuid"
)

type PaymentService interface {
	CreatePayment(ctx context.Context, cmd events.PaymentCreatePayload) error
}

type PaymentRepository interface {
	Create(ctx context.Context, p *model.Payment) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.PaymentStatus) error
	Get(ctx context.Context, userID uuid.UUID) (*[]model.Payment, error)
	GetById(ctx context.Context, id uuid.UUID) (*model.Payment, error)
}

