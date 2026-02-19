package service

import (
	"context"
	"time"

	"github.com/alexssanderFonseca/pagamento/internal/domain"
	"github.com/alexssanderFonseca/pagamento/internal/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type PaymentService struct {
	repo           domain.PaymentRepository
	mpClient       domain.MercadoPagoClient
	eventPublisher domain.PaymentEventPublisher
}

func NewPaymentService(repo domain.PaymentRepository, mpClient domain.MercadoPagoClient, eventPublisher domain.PaymentEventPublisher) *PaymentService {
	return &PaymentService{
		repo:           repo,
		mpClient:       mpClient,
		eventPublisher: eventPublisher,
	}
}

func (s *PaymentService) CreatePayment(ctx context.Context, req domain.CreatePaymentRequest) (*domain.Payment, error) {
	logger.Info("creating payment order",
		zap.String("external_reference", req.ExternalReference),
		zap.Float64("amount", req.Amount),
	)

	qrCode, err := s.mpClient.CreateQRCodeOrder(ctx, req)
	if err != nil {
		logger.Error("failed to create qr code order in mercadopago",
			zap.Error(err),
			zap.String("external_reference", req.ExternalReference),
		)
		return nil, err
	}

	payment := domain.Payment{
		ID:                uuid.New().String(),
		ExternalReference: req.ExternalReference,
		Amount:            req.Amount,
		Status:            domain.StatusPending,
		QRCode:            qrCode,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err = s.repo.Save(ctx, payment)
	if err != nil {
		logger.Error("failed to save payment in dynamodb",
			zap.Error(err),
			zap.String("payment_id", payment.ID),
		)
		return nil, err
	}

	logger.Info("payment created successfully",
		zap.String("payment_id", payment.ID),
		zap.String("status", string(payment.Status)),
	)

	return &payment, nil
}

func (s *PaymentService) ProcessWebhook(ctx context.Context, notification domain.MPWebhookNotification) error {
	logger.Info("received webhook notification",
		zap.String("type", notification.Type),
		zap.String("action", notification.Action),
	)

	if notification.Type == "payment" {
		paymentID := notification.Data.ID
		mpPayment, err := s.mpClient.GetPaymentDetails(ctx, paymentID)
		if err != nil {
			logger.Error("failed to get payment details from mercadopago",
				zap.Error(err),
				zap.String("mp_payment_id", paymentID),
			)
			return err
		}

		logger.Info("mercadopago payment details fetched",
			zap.String("mp_payment_id", paymentID),
			zap.String("mp_status", mpPayment.Status),
			zap.String("external_reference", mpPayment.ExternalReference),
		)

		payment, err := s.repo.GetByExternalReference(ctx, mpPayment.ExternalReference)
		if err != nil {
			logger.Error("failed to fetch local payment by external reference",
				zap.Error(err),
				zap.String("external_reference", mpPayment.ExternalReference),
			)
			return err
		}

		if payment == nil {
			logger.Warn("payment not found for received webhook",
				zap.String("external_reference", mpPayment.ExternalReference),
			)
			return nil
		}

		var newStatus domain.PaymentStatus
		switch mpPayment.Status {
		case "approved":
			newStatus = domain.StatusApproved
		case "rejected", "cancelled":
			newStatus = domain.StatusRejected
		default:
			newStatus = domain.StatusPending
		}

		err = s.repo.UpdateStatus(ctx, payment.ID, newStatus)
		if err != nil {
			logger.Error("failed to update payment status",
				zap.Error(err),
				zap.String("payment_id", payment.ID),
				zap.String("new_status", string(newStatus)),
			)
			return err
		}

		logger.Info("payment status updated via webhook",
			zap.String("payment_id", payment.ID),
			zap.String("new_status", string(newStatus)),
		)

		// Publicar evento no SNS
		if s.eventPublisher != nil {
			err = s.eventPublisher.PublishPaymentProcessed(ctx, domain.PaymentProcessedEvent{
				PaymentID:         payment.ID,
				ExternalReference: payment.ExternalReference,
				Status:            newStatus,
				ProcessedAt:       time.Now(),
			})
			if err != nil {
				logger.Error("failed to publish payment processed event to SNS",
					zap.Error(err),
					zap.String("payment_id", payment.ID),
				)
				// Não retornamos erro aqui para não causar re-tentativas do webhook MP por falha no SNS
			} else {
				logger.Info("payment processed event published to SNS",
					zap.String("payment_id", payment.ID),
				)
			}
		}
	}
	return nil
}
