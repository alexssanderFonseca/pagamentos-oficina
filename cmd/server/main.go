package main

import (
	"context"
	"os"

	"github.com/alexssanderFonseca/pagamento/internal/api"
	"github.com/alexssanderFonseca/pagamento/internal/api/handler"
	"github.com/alexssanderFonseca/pagamento/internal/integration/mercadopago"
	"github.com/alexssanderFonseca/pagamento/internal/integration/sns"
	"github.com/alexssanderFonseca/pagamento/internal/logger"
	repo "github.com/alexssanderFonseca/pagamento/internal/repository/dynamodb"
	"github.com/alexssanderFonseca/pagamento/internal/service"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"go.uber.org/zap"
)

// @title           Pagamento API
// @version         1.0
// @description     Serviço de processamento de pagamentos para a Oficina.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /v1

// @securityDefinitions.basic  BasicAuth

func main() {
	ctx := context.Background()

	// AWS Config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Fatal("unable to load SDK config", zap.Error(err))
	}

	// DynamoDB Client
	dbClient := dynamodb.NewFromConfig(cfg)
	
	// SNS Client
	snsClient := sns.NewClient(cfg)

	// Dependency Injection
	paymentRepo := repo.NewPaymentRepository(dbClient)
	mpClient := mercadopago.NewClient()
	paymentService := service.NewPaymentService(paymentRepo, mpClient, snsClient)
	paymentHandler := handler.NewPaymentHandler(paymentService)

	// Router initialization
	r := api.SetupRouter(paymentHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info("Server starting", zap.String("port", port))
	if err := r.Run(":" + port); err != nil {
		logger.Fatal("failed to run server", zap.Error(err))
	}
}
