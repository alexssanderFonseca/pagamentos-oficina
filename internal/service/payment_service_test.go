package service

import (
	"context"
	"errors"
	"testing"

	"github.com/alexssanderFonseca/pagamento/internal/domain"
)

// Mock do Repository
type MockRepo struct {
	SaveFunc                   func(ctx context.Context, payment domain.Payment) error
	GetByExternalReferenceFunc func(ctx context.Context, ref string) (*domain.Payment, error)
	UpdateStatusFunc           func(ctx context.Context, id string, status domain.PaymentStatus) error
}

func (m *MockRepo) Save(ctx context.Context, payment domain.Payment) error {
	return m.SaveFunc(ctx, payment)
}
func (m *MockRepo) GetByID(ctx context.Context, id string) (*domain.Payment, error) { return nil, nil }
func (m *MockRepo) GetByExternalReference(ctx context.Context, ref string) (*domain.Payment, error) {
	return m.GetByExternalReferenceFunc(ctx, ref)
}
func (m *MockRepo) UpdateStatus(ctx context.Context, id string, status domain.PaymentStatus) error {
	return m.UpdateStatusFunc(ctx, id, status)
}

// Mock do MP Client
type MockMPClient struct {
	CreateQRCodeFunc      func(ctx context.Context, req domain.CreatePaymentRequest) (string, error)
	GetPaymentDetailsFunc func(ctx context.Context, id string) (*domain.MPPaymentResponse, error)
}

func (m *MockMPClient) CreateQRCodeOrder(ctx context.Context, req domain.CreatePaymentRequest) (string, error) {
	return m.CreateQRCodeFunc(ctx, req)
}
func (m *MockMPClient) GetPaymentDetails(ctx context.Context, paymentID string) (*domain.MPPaymentResponse, error) {
	return m.GetPaymentDetailsFunc(ctx, paymentID)
}

// Mock do SNS Publisher
type MockPublisher struct {
	PublishFunc func(ctx context.Context, event domain.PaymentProcessedEvent) error
}

func (m *MockPublisher) PublishPaymentProcessed(ctx context.Context, event domain.PaymentProcessedEvent) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, event)
	}
	return nil
}

func TestCreatePayment_Success(t *testing.T) {
	repo := &MockRepo{
		SaveFunc: func(ctx context.Context, payment domain.Payment) error { return nil },
	}
	mp := &MockMPClient{
		CreateQRCodeFunc: func(ctx context.Context, req domain.CreatePaymentRequest) (string, error) {
			return "qr_data_mock", nil
		},
	}
	publisher := &MockPublisher{}

	svc := NewPaymentService(repo, mp, publisher)

	req := domain.CreatePaymentRequest{
		ExternalReference: "ORDER-1",
		Amount:            10.50,
		Description:       "Test",
	}

	payment, err := svc.CreatePayment(context.Background(), req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if payment.QRCode != "qr_data_mock" {
		t.Errorf("expected qr_data_mock, got %s", payment.QRCode)
	}

	if payment.Status != domain.StatusPending {
		t.Errorf("expected status pending, got %s", payment.Status)
	}
}

func TestCreatePayment_MP_Error(t *testing.T) {
	repo := &MockRepo{}
	mp := &MockMPClient{
		CreateQRCodeFunc: func(ctx context.Context, req domain.CreatePaymentRequest) (string, error) {
			return "", errors.New("mp api error")
		},
	}
	publisher := &MockPublisher{}

	svc := NewPaymentService(repo, mp, publisher)

	req := domain.CreatePaymentRequest{
		ExternalReference: "ORDER-1",
		Amount:            10.50,
	}

	_, err := svc.CreatePayment(context.Background(), req)

	if err == nil {
		t.Fatal("expected error from MP, got nil")
	}
}
