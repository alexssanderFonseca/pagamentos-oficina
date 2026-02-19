package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexssanderFonseca/pagamento/internal/domain"
	"github.com/gin-gonic/gin"
)

type mockPaymentService struct {
	createPaymentFunc func(ctx context.Context, req domain.CreatePaymentRequest) (*domain.Payment, error)
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

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := domain.CreatePaymentRequest{
		ExternalReference: "ORDER-1",
		Amount:            10.0,
		Description:       "Test",
	}
	jsonBody, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/v1/payments", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreatePayment(c)

	if w.Code != http.StatusCreated {
		t.Errorf("esperava status 201, obteve %d", w.Code)
	}

	var resp domain.Payment
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ID != "test-id" {
		t.Errorf("esperava ID test-id, obteve %s", resp.ID)
	}
}

func TestPaymentHandler_HandleWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockPaymentService{
		processWebhookFunc: func(ctx context.Context, notification domain.MPWebhookNotification) error {
			return nil
		},
	}

	h := NewPaymentHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	notification := domain.MPWebhookNotification{
		Type: "payment",
	}
	notification.Data.ID = "123"
	
	jsonBody, _ := json.Marshal(notification)
	c.Request, _ = http.NewRequest("POST", "/v1/webhooks/mercadopago", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	h.HandleWebhook(c)

	if w.Code != http.StatusOK {
		t.Errorf("esperava status 200, obteve %d", w.Code)
	}
}
