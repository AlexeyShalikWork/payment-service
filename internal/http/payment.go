package http

import (
	"payment-service/internal/service"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type Handler struct {
	svc *service.PaymentService
}

func NewHttpPaymentHandler(s *service.PaymentService) *Handler {
	return &Handler{svc: s}
}

func (h *Handler) GetPayments(c fiber.Ctx) error {
	id := c.Params("user_id")
	userID, err := uuid.Parse(id)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid uuid format",
		})
	}

	payments, err := h.svc.GetPayments(c.Context(), userID)
	if err != nil {
		return err
	}

	return c.JSON(payments)
}

func (h *Handler) GetPayment(c fiber.Ctx) error {
	id := c.Params("id")
	paymentID, err := uuid.Parse(id)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid uuid format",
		})
	}

	payment, err := h.svc.GetPayment(c.Context(), paymentID)
	if err != nil {
		return err
	}

	return c.JSON(payment)
}
