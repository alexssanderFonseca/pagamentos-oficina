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
		// Se não houver secret configurado, deixamos passar (útil para desenvolvimento inicial)
		// Mas em produção, isso deve ser obrigatório.
		return true
	}

	signatureHeader := c.GetHeader("x-signature")
	if signatureHeader == "" {
		return false
	}

	// O formato esperado do header é: ts=TIMESTAMP,v1=HASH
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

	// O manifesto para assinar é: id:ID_DA_NOTIFICACAO;ts:TIMESTAMP;
	// O ID vem do campo data.id da notificação
	manifest := fmt.Sprintf("id:%s;ts:%s;", notification.Data.ID, ts)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(manifest))
	expectedHash := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(hash), []byte(expectedHash))
}
