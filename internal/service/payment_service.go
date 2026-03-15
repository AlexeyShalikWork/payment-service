package service

import (
	"context"
	"fmt"
	"payment-service/internal/kafka/events"
	"payment-service/internal/model"
	"payment-service/internal/ports"
	"payment-service/internal/tracing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type PaymentService struct {
	repo ports.PaymentRepository
}

func NewPaymentService(r ports.PaymentRepository) *PaymentService {
	return &PaymentService{repo: r}
}

func (s *PaymentService) CreatePayment(ctx context.Context, payload events.PaymentCreatePayload) error {
	ctx, span := tracing.Tracer().Start(ctx, "PaymentService.CreatePayment",
		trace.WithAttributes(
			attribute.String("payment.id", payload.PaymentID.String()),
			attribute.String("payment.type", string(payload.Type)),
			attribute.Int64("payment.amount", payload.Amount),
			attribute.String("payment.currency", string(payload.Currency)),
		),
	)
	defer span.End()

	if payload.Amount <= 0 {
		return fmt.Errorf("invalid amount: %d", payload.Amount)
	}

	switch payload.Currency {
	case model.USD, model.EUR, model.RUB:
	default:
		return fmt.Errorf("unsupported currency: %s", payload.Currency)
	}

	switch payload.Type {
	case model.TypeCharge, model.TypeRefund, model.TypePayout:
	default:
		return fmt.Errorf("unsupported payment type: %s", payload.Type)
	}

	payment := &model.Payment{
		ID:       payload.PaymentID,
		UserID:   payload.UserID,
		Type:     payload.Type,
		Amount:   payload.Amount,
		Currency: payload.Currency,
		Status:   model.StatusCreated,
	}

	if err := s.repo.Create(ctx, payment); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

func (s *PaymentService) GetPayments(ctx context.Context, userID uuid.UUID) (*[]model.Payment, error) {
	ctx, span := tracing.Tracer().Start(ctx, "PaymentService.GetPayments",
		trace.WithAttributes(attribute.String("user.id", userID.String())),
	)
	defer span.End()

	payments, err := s.repo.Get(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return payments, nil
}

func (s *PaymentService) GetPayment(ctx context.Context, id uuid.UUID) (*model.Payment, error) {
	ctx, span := tracing.Tracer().Start(ctx, "PaymentService.GetPayment",
		trace.WithAttributes(attribute.String("payment.id", id.String())),
	)
	defer span.End()

	payment, err := s.repo.GetById(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return payment, nil
}
