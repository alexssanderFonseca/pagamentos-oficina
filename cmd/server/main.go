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
