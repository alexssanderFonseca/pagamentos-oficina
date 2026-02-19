package domain

import (
	"context"
	"time"
)

type PaymentStatus string

const (
	StatusPending  PaymentStatus = "pending"
	StatusApproved PaymentStatus = "approved"
	StatusRejected PaymentStatus = "rejected"
)

type Payment struct {
	ID                string        `json:"id" dynamodbav:"id"`
	ExternalReference string        `json:"external_reference" dynamodbav:"external_reference"`
	Amount            float64       `json:"amount" dynamodbav:"amount"`
	Status            PaymentStatus `json:"status" dynamodbav:"status"`
	QRCode            string        `json:"qr_code" dynamodbav:"qr_code"`
	CreatedAt         time.Time     `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at" dynamodbav:"updated_at"`
}

type CreatePaymentRequest struct {
	ExternalReference string  `json:"external_reference" binding:"required"`
	Amount            float64 `json:"amount" binding:"required"`
	Description       string  `json:"description" binding:"required"`
}

type MPWebhookNotification struct {
	ID          interface{} `json:"id"`
	LiveMode    bool        `json:"live_mode"`
	Type        string      `json:"type"`
	DateCreated string      `json:"date_created"`
	UserID      string      `json:"user_id"`
	APIVersion  string      `json:"api_version"`
	Action      string      `json:"action"`
		Data     struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	
	// Interfaces para Mocking e Desacoplamento
	type PaymentRepository interface {
		Save(ctx context.Context, payment Payment) error
		GetByID(ctx context.Context, id string) (*Payment, error)
		GetByExternalReference(ctx context.Context, ref string) (*Payment, error)
		UpdateStatus(ctx context.Context, id string, status PaymentStatus) error
	}
	
	type MPPaymentResponse struct {
		ID                int64  `json:"id"`
		Status            string `json:"status"`
		ExternalReference string `json:"external_reference"`
	}
	
	type MercadoPagoClient interface {
		CreateQRCodeOrder(ctx context.Context, req CreatePaymentRequest) (string, error)
		GetPaymentDetails(ctx context.Context, paymentID string) (*MPPaymentResponse, error)
	}

	type PaymentProcessedEvent struct {
		PaymentID         string        `json:"payment_id"`
		ExternalReference string        `json:"external_reference"`
		Status            PaymentStatus `json:"status"`
		ProcessedAt       time.Time     `json:"processed_at"`
	}

	type PaymentEventPublisher interface {
		PublishPaymentProcessed(ctx context.Context, event PaymentProcessedEvent) error
	}
