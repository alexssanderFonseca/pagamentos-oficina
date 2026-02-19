package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/alexssanderFonseca/pagamento/internal/domain"
	"github.com/alexssanderFonseca/pagamento/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PaymentService interface {
	CreatePayment(ctx context.Context, req domain.CreatePaymentRequest) (*domain.Payment, error)
	ProcessWebhook(ctx context.Context, notification domain.MPWebhookNotification) error
}

type PaymentHandler struct {
	service PaymentService
}

func NewPaymentHandler(service PaymentService) *PaymentHandler {
	return &PaymentHandler{
		service: service,
	}
}

// CreatePayment godoc
// @Summary      Criar um novo pagamento
// @Description  Gera um QR Code no Mercado Pago para uma ordem de serviço
// @Tags         pagamentos
// @Accept       json
// @Produce      json
// @Param        request  body      domain.CreatePaymentRequest  true  "Dados do Pagamento"
// @Success      201      {object}  domain.Payment
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /pagamentos [post]
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var req domain.CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment, err := h.service.CreatePayment(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, payment)
}

// HandleWebhook godoc
// @Summary      Receber notificação do Mercado Pago
// @Description  Processa o status do pagamento via webhook assinado
// @Tags         webhooks
// @Accept       json
// @Produce      json
// @Param        X-Signature  header    string  true  "Assinura HMAC-SHA256"
// @Param        notification body      domain.MPWebhookNotification  true  "Notificação MP"
// @Success      200      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /webhooks/mercadopago [post]
func (h *PaymentHandler) HandleWebhook(c *gin.Context) {
	var notification domain.MPWebhookNotification
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validação de Segurança do Webhook
	if !h.validateSignature(c, notification) {
		logger.Warn("invalid webhook signature detected",
			zap.String("id", notification.Data.ID),
			zap.String("signature", c.GetHeader("x-signature")),
		)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	if err := h.service.ProcessWebhook(c.Request.Context(), notification); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "received"})
}

func (h *PaymentHandler) validateSignature(c *gin.Context, notification domain.MPWebhookNotification) bool {
	secret := os.Getenv("MERCADO_PAGO_WEBHOOK_SECRET")
	if secret == "" {
		return true
	}

	signatureHeader := c.GetHeader("x-signature")
	if signatureHeader == "" {
		return false
	}

	parts := strings.Split(signatureHeader, ",")
	var ts, hash string
	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			continue
		}
		if kv[0] == "ts" {
			ts = kv[1]
		} else if kv[0] == "v1" {
			hash = kv[1]
		}
	}

	if ts == "" || hash == "" {
		return false
	}

	manifest := fmt.Sprintf("id:%s;ts:%s;", notification.Data.ID, ts)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(manifest))
	expectedHash := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(hash), []byte(expectedHash))
}
