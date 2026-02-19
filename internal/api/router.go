package api

import (
	_ "github.com/alexssanderFonseca/pagamento/docs"
	"github.com/alexssanderFonseca/pagamento/internal/api/handler"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func SetupRouter(paymentHandler *handler.PaymentHandler) *gin.Engine {
	r := gin.Default()

	// OpenTelemetry Middleware
	r.Use(otelgin.Middleware("pagamento"))

	// Swagger route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "up"})
	})

	v1 := r.Group("/v1")
	{
		payments := v1.Group("/pagamentos")
		{
			payments.POST("", paymentHandler.CreatePayment)
		}

		// Rota para Webhooks do Mercado Pago
		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/mercadopago", paymentHandler.HandleWebhook)
		}
	}

	return r
}
