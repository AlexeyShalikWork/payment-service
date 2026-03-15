package service

import (
	"context"
	"errors"
	"payment-service/internal/kafka/events"
	"payment-service/internal/model"
	"testing"

	"github.com/google/uuid"
)

type mockRepo struct {
	payments  map[uuid.UUID]*model.Payment
	createErr error
}

func newMockRepo() *mockRepo {
	return &mockRepo{payments: make(map[uuid.UUID]*model.Payment)}
}

func (m *mockRepo) Create(_ context.Context, p *model.Payment) error {
	if m.createErr != nil {
		return m.createErr
	}
	if _, exists := m.payments[p.ID]; !exists {
		m.payments[p.ID] = p
	}
	return nil
}

func (m *mockRepo) UpdateStatus(_ context.Context, id uuid.UUID, status model.PaymentStatus) error {
	if p, ok := m.payments[id]; ok {
		p.Status = status
		return nil
	}
	return errors.New("not found")
}

func (m *mockRepo) Get(_ context.Context, userID uuid.UUID) (*[]model.Payment, error) {
	var result []model.Payment
	for _, p := range m.payments {
		if p.UserID == userID {
			result = append(result, *p)
		}
	}
	return &result, nil
}

func (m *mockRepo) GetById(_ context.Context, id uuid.UUID) (*model.Payment, error) {
	if p, ok := m.payments[id]; ok {
		return p, nil
	}
	return nil, errors.New("not found")
}

func validPayload() events.PaymentCreatePayload {
	return events.PaymentCreatePayload{
		PaymentID: uuid.New(),
		UserID:    uuid.New(),
		Type:      model.TypeCharge,
		Amount:    1000,
		Currency:  model.USD,
	}
}

func TestCreatePayment_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewPaymentService(repo)

	payload := validPayload()
	err := svc.CreatePayment(context.Background(), payload)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(repo.payments) != 1 {
		t.Fatalf("expected 1 payment in repo, got %d", len(repo.payments))
	}

	created := repo.payments[payload.PaymentID]
	if created.Status != model.StatusCreated {
		t.Errorf("expected status %s, got %s", model.StatusCreated, created.Status)
	}
	if created.Type != model.TypeCharge {
		t.Errorf("expected type %s, got %s", model.TypeCharge, created.Type)
	}
}

func TestCreatePayment_InvalidAmount(t *testing.T) {
	svc := NewPaymentService(newMockRepo())

	payload := validPayload()
	payload.Amount = -100

	err := svc.CreatePayment(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error for negative amount")
	}
}

func TestCreatePayment_ZeroAmount(t *testing.T) {
	svc := NewPaymentService(newMockRepo())

	payload := validPayload()
	payload.Amount = 0

	err := svc.CreatePayment(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error for zero amount")
	}
}

func TestCreatePayment_InvalidCurrency(t *testing.T) {
	svc := NewPaymentService(newMockRepo())

	payload := validPayload()
	payload.Currency = "BANANA"

	err := svc.CreatePayment(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error for invalid currency")
	}
}

func TestCreatePayment_InvalidType(t *testing.T) {
	svc := NewPaymentService(newMockRepo())

	payload := validPayload()
	payload.Type = "unknown"

	err := svc.CreatePayment(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestCreatePayment_Idempotent(t *testing.T) {
	repo := newMockRepo()
	svc := NewPaymentService(repo)

	payload := validPayload()

	if err := svc.CreatePayment(context.Background(), payload); err != nil {
		t.Fatalf("first call: %v", err)
	}

	if err := svc.CreatePayment(context.Background(), payload); err != nil {
		t.Fatalf("second call: %v", err)
	}

	if len(repo.payments) != 1 {
		t.Errorf("expected 1 payment in repo, got %d", len(repo.payments))
	}
}

func TestCreatePayment_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.createErr = errors.New("db connection lost")
	svc := NewPaymentService(repo)

	err := svc.CreatePayment(context.Background(), validPayload())
	if err == nil {
		t.Fatal("expected error when repo fails")
	}
}

func TestGetPayment_Success(t *testing.T) {
	repo := newMockRepo()
	id := uuid.New()
	repo.payments[id] = &model.Payment{
		ID:       id,
		Amount:   500,
		Currency: model.EUR,
		Status:   model.StatusCreated,
	}

	svc := NewPaymentService(repo)

	payment, err := svc.GetPayment(context.Background(), id)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if payment.ID != id {
		t.Errorf("expected id %s, got %s", id, payment.ID)
	}
}

func TestGetPayment_NotFound(t *testing.T) {
	svc := NewPaymentService(newMockRepo())

	_, err := svc.GetPayment(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent payment")
	}
}
