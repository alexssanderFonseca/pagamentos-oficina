package api

import (
	"github.com/alexssanderFonseca/pagamento/internal/api/handler"
	"github.com/gin-gonic/gin"
)

func SetupRouter(paymentHandler *handler.PaymentHandler) *gin.Engine {
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "up"})
	})

	v1 := r.Group("/v1")
	{
		payments := v1.Group("/pagamentos")
		{
			payments.POST("", paymentHandler.CreatePayment)
			// Aqui você pode adicionar outras rotas como:
			// payments.GET("/:id", paymentHandler.GetPayment)
		}

		// Rota para Webhooks do Mercado Pago
		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/mercadopago", paymentHandler.HandleWebhook)
		}
	}

	return r
}
