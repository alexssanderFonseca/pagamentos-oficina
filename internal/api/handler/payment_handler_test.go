package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexssanderFonseca/pagamento/internal/domain"
	"github.com/gin-gonic/gin"
)

type mockPaymentService struct {
	createPaymentFunc  func(ctx context.Context, req domain.CreatePaymentRequest) (*domain.Payment, error)
	processWebhookFunc func(ctx context.Context, notification domain.MPWebhookNotification) error
}

func (m *mockPaymentService) CreatePayment(ctx context.Context, req domain.CreatePaymentRequest) (*domain.Payment, error) {
	return m.createPaymentFunc(ctx, req)
}

func (m *mockPaymentService) ProcessWebhook(ctx context.Context, notification domain.MPWebhookNotification) error {
	return m.processWebhookFunc(ctx, notification)
}

func TestPaymentHandler_CreatePayment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockPaymentService{
		createPaymentFunc: func(ctx context.Context, req domain.CreatePaymentRequest) (*domain.Payment, error) {
			return &domain.Payment{
				ID:                "test-id",
				ExternalReference: req.ExternalReference,
				Amount:            req.Amount,
			}, nil
		},
	}

	h := NewPaymentHandler(svc)

	t.Run("Success", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := domain.CreatePaymentRequest{ExternalReference: "ORDER-1", Amount: 10.0, Description: "Test"}
		jsonBody, _ := json.Marshal(body)
		c.Request, _ = http.NewRequest("POST", "/", bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")
		h.CreatePayment(c)
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/", bytes.NewBufferString("{invalid}"))
		h.CreatePayment(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("Service Error", func(t *testing.T) {
		svc.createPaymentFunc = func(ctx context.Context, req domain.CreatePaymentRequest) (*domain.Payment, error) {
			return nil, errors.New("service failed")
		}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := domain.CreatePaymentRequest{ExternalReference: "ORDER-1", Amount: 10.0, Description: "Test"}
		jsonBody, _ := json.Marshal(body)
		c.Request, _ = http.NewRequest("POST", "/", bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")
		h.CreatePayment(c)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

func TestPaymentHandler_HandleWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockPaymentService{
		processWebhookFunc: func(ctx context.Context, notification domain.MPWebhookNotification) error {
			return nil
		},
	}

	h := NewPaymentHandler(svc)

	t.Run("Success", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		notification := domain.MPWebhookNotification{Type: "payment"}
		notification.Data.ID = "123"
		jsonBody, _ := json.Marshal(notification)
		c.Request, _ = http.NewRequest("POST", "/", bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")
		h.HandleWebhook(c)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/", bytes.NewBufferString("{invalid}"))
		h.HandleWebhook(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("Service Error", func(t *testing.T) {
		svc.processWebhookFunc = func(ctx context.Context, notification domain.MPWebhookNotification) error {
			return errors.New("webhook failed")
		}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		notification := domain.MPWebhookNotification{Type: "payment"}
		jsonBody, _ := json.Marshal(notification)
		c.Request, _ = http.NewRequest("POST", "/", bytes.NewBuffer(jsonBody))
		c.Request.Header.Set("Content-Type", "application/json")
		h.HandleWebhook(c)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}
