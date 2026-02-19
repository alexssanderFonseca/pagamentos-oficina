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
	if m.GetByExternalReferenceFunc != nil {
		return m.GetByExternalReferenceFunc(ctx, ref)
	}
	return nil, nil
}
func (m *MockRepo) UpdateStatus(ctx context.Context, id string, status domain.PaymentStatus) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, status)
	}
	return nil
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
	if m.GetPaymentDetailsFunc != nil {
		return m.GetPaymentDetailsFunc(ctx, paymentID)
	}
	return nil, nil
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

func TestCreatePayment_Repo_Error(t *testing.T) {
	repo := &MockRepo{
		SaveFunc: func(ctx context.Context, payment domain.Payment) error { return errors.New("db error") },
	}
	mp := &MockMPClient{
		CreateQRCodeFunc: func(ctx context.Context, req domain.CreatePaymentRequest) (string, error) {
			return "qr_data", nil
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
		t.Fatal("expected error from Repo, got nil")
	}
}

func TestProcessWebhook_Approved(t *testing.T) {
	repo := &MockRepo{
		GetByExternalReferenceFunc: func(ctx context.Context, ref string) (*domain.Payment, error) {
			return &domain.Payment{ID: "local-1", ExternalReference: "ext-1"}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, id string, status domain.PaymentStatus) error {
			if status != domain.StatusApproved {
				t.Errorf("expected approved status, got %s", status)
			}
			return nil
		},
	}
	mp := &MockMPClient{
		GetPaymentDetailsFunc: func(ctx context.Context, id string) (*domain.MPPaymentResponse, error) {
			return &domain.MPPaymentResponse{Status: "approved", ExternalReference: "ext-1"}, nil
		},
	}
	publisher := &MockPublisher{}

	svc := NewPaymentService(repo, mp, publisher)

	notification := domain.MPWebhookNotification{
		Type: "payment",
		Data: struct {
			ID string `json:"id"`
		}{ID: "mp-123"},
	}

	err := svc.ProcessWebhook(context.Background(), notification)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestProcessWebhook_Rejected(t *testing.T) {
	repo := &MockRepo{
		GetByExternalReferenceFunc: func(ctx context.Context, ref string) (*domain.Payment, error) {
			return &domain.Payment{ID: "local-1", ExternalReference: "ext-1"}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, id string, status domain.PaymentStatus) error {
			if status != domain.StatusRejected {
				t.Errorf("expected rejected status, got %s", status)
			}
			return nil
		},
	}
	mp := &MockMPClient{
		GetPaymentDetailsFunc: func(ctx context.Context, id string) (*domain.MPPaymentResponse, error) {
			return &domain.MPPaymentResponse{Status: "cancelled", ExternalReference: "ext-1"}, nil
		},
	}
	publisher := &MockPublisher{}

	svc := NewPaymentService(repo, mp, publisher)

	notification := domain.MPWebhookNotification{
		Type: "payment",
		Data: struct {
			ID string `json:"id"`
		}{ID: "mp-123"},
	}

	err := svc.ProcessWebhook(context.Background(), notification)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestProcessWebhook_NotFoundLocal(t *testing.T) {
	repo := &MockRepo{
		GetByExternalReferenceFunc: func(ctx context.Context, ref string) (*domain.Payment, error) {
			return nil, nil // Not found locally
		},
	}
	mp := &MockMPClient{
		GetPaymentDetailsFunc: func(ctx context.Context, id string) (*domain.MPPaymentResponse, error) {
			return &domain.MPPaymentResponse{Status: "approved", ExternalReference: "ext-unknown"}, nil
		},
	}

	svc := NewPaymentService(repo, mp, nil)

	notification := domain.MPWebhookNotification{
		Type: "payment",
		Data: struct {
			ID string `json:"id"`
		}{ID: "mp-123"},
	}

	err := svc.ProcessWebhook(context.Background(), notification)
	if err != nil {
		t.Fatalf("expected no error (just log warn), got %v", err)
	}
}

func TestProcessWebhook_MPError(t *testing.T) {
	mp := &MockMPClient{
		GetPaymentDetailsFunc: func(ctx context.Context, id string) (*domain.MPPaymentResponse, error) {
			return nil, errors.New("api error")
		},
	}
	svc := NewPaymentService(nil, mp, nil)
	err := svc.ProcessWebhook(context.Background(), domain.MPWebhookNotification{Type: "payment"})
	if err == nil {
		t.Fatal("expected error from MP Client")
	}
}

func TestProcessWebhook_SNSError(t *testing.T) {
	repo := &MockRepo{
		GetByExternalReferenceFunc: func(ctx context.Context, ref string) (*domain.Payment, error) {
			return &domain.Payment{ID: "local-1", ExternalReference: "ext-1"}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, id string, status domain.PaymentStatus) error {
			return nil
		},
	}
	mp := &MockMPClient{
		GetPaymentDetailsFunc: func(ctx context.Context, id string) (*domain.MPPaymentResponse, error) {
			return &domain.MPPaymentResponse{Status: "approved", ExternalReference: "ext-1"}, nil
		},
	}
	publisher := &MockPublisher{
		PublishFunc: func(ctx context.Context, event domain.PaymentProcessedEvent) error {
			return errors.New("sns error")
		},
	}

	svc := NewPaymentService(repo, mp, publisher)

	notification := domain.MPWebhookNotification{
		Type: "payment",
		Data: struct {
			ID string `json:"id"`
		}{ID: "mp-123"},
	}

	err := svc.ProcessWebhook(context.Background(), notification)
	if err != nil {
		t.Fatalf("expected no error from webhook even if SNS fails, got %v", err)
	}
}

func TestProcessWebhook_UnknownType(t *testing.T) {
	svc := NewPaymentService(nil, nil, nil)
	err := svc.ProcessWebhook(context.Background(), domain.MPWebhookNotification{Type: "unknown"})
	if err != nil {
		t.Fatal("should ignore unknown notification types")
	}
}

func TestProcessWebhook_RepoGetError(t *testing.T) {
	repo := &MockRepo{
		GetByExternalReferenceFunc: func(ctx context.Context, ref string) (*domain.Payment, error) {
			return nil, errors.New("db error")
		},
	}
	mp := &MockMPClient{
		GetPaymentDetailsFunc: func(ctx context.Context, id string) (*domain.MPPaymentResponse, error) {
			return &domain.MPPaymentResponse{Status: "approved", ExternalReference: "ext-1"}, nil
		},
	}
	svc := NewPaymentService(repo, mp, nil)
	err := svc.ProcessWebhook(context.Background(), domain.MPWebhookNotification{
		Type: "payment",
		Data: struct {
			ID string `json:"id"`
		}{ID: "mp-123"},
	})
	if err == nil {
		t.Fatal("expected error from repo Get")
	}
}

func TestProcessWebhook_RepoUpdateError(t *testing.T) {
	repo := &MockRepo{
		GetByExternalReferenceFunc: func(ctx context.Context, ref string) (*domain.Payment, error) {
			return &domain.Payment{ID: "local-1"}, nil
		},
		UpdateStatusFunc: func(ctx context.Context, id string, status domain.PaymentStatus) error {
			return errors.New("update error")
		},
	}
	mp := &MockMPClient{
		GetPaymentDetailsFunc: func(ctx context.Context, id string) (*domain.MPPaymentResponse, error) {
			return &domain.MPPaymentResponse{Status: "approved", ExternalReference: "ext-1"}, nil
		},
	}
	svc := NewPaymentService(repo, mp, nil)
	err := svc.ProcessWebhook(context.Background(), domain.MPWebhookNotification{
		Type: "payment",
		Data: struct {
			ID string `json:"id"`
		}{ID: "mp-123"},
	})
	if err == nil {
		t.Fatal("expected error from repo Update")
	}
}

