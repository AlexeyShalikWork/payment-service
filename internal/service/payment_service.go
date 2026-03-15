package service

import (
	"context"
	"fmt"
	"payment-service/internal/kafka"
	"payment-service/internal/kafka/events"
	"payment-service/internal/model"
	"payment-service/internal/repository"

	"github.com/google/uuid"
)

type PaymentService struct {
	repo  *repository.PaymentRepo
	kafka *kafka.Producer
}

func NewPaymentService(r *repository.PaymentRepo, k *kafka.Producer) *PaymentService {
	return &PaymentService{repo: r, kafka: k}
}

func (s *PaymentService) CreatePayment(ctx context.Context, payload events.PaymentCreatePayload) error {
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

	existing, err := s.repo.GetById(ctx, payload.PaymentID)

	if err == nil && existing != nil {
		return nil
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
		return err
	}

	if err := s.kafka.PublishPaymentCreated(ctx, payment); err != nil {
		return fmt.Errorf("publish payment created: %w", err)
	}

	return nil
}

func (s *PaymentService) GetPayments(ctx context.Context, user_id uuid.UUID) (*[]model.Payment, error) {
	payment, err := s.repo.Get(ctx, user_id)

	if err != nil {
		return nil, err

	}

	return payment, nil
}

func (s *PaymentService) GetPayment(ctx context.Context, id uuid.UUID) (*model.Payment, error) {
	payment, err := s.repo.GetById(ctx, id)

	if err != nil {
		return nil, err

	}

	return payment, nil
}
