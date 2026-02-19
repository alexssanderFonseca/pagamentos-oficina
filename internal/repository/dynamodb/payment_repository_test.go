package dynamodb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alexssanderFonseca/pagamento/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func setupTestDB(t *testing.T) (*dynamodb.Client, string) {
	ctx := context.Background()
	tableName := "PaymentsTest"

	// Configuração para o LocalStack
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: "http://localhost:4566"}, nil
			})),
	)
	if err != nil {
		t.Skip("LocalStack não disponível, pulando teste de integração")
	}

	client := dynamodb.NewFromConfig(cfg)

	// Criar tabela para o teste
	_, err = client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("external_reference"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("id"), KeyType: types.KeyTypeHash},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("ExternalReferenceIndex"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("external_reference"), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(5),
					WriteCapacityUnits: aws.Int64(5),
				},
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	})

	if err != nil {
		// Se a tabela já existir, ignoramos o erro (ou deletamos e criamos de novo)
		t.Logf("Aviso ao criar tabela: %v", err)
	}

	return client, tableName
}

func TestPaymentRepository_Integration(t *testing.T) {
	// Pula se for rodado apenas testes unitários
	if testing.Short() {
		t.Skip("Pulando teste de integração")
	}

	client, tableName := setupTestDB(t)
	os.Setenv("DYNAMODB_TABLE_NAME", tableName)
	repo := NewPaymentRepository(client)

	ctx := context.Background()
	payment := domain.Payment{
		ID:                "test-id-1",
		ExternalReference: "REF-INTEGRATION-1",
		Amount:            100.50,
		Status:            domain.StatusPending,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// 1. Teste Save
	t.Run("Save Payment", func(t *testing.T) {
		err := repo.Save(ctx, payment)
		if err != nil {
			t.Fatalf("falha ao salvar: %v", err)
		}
	})

	// 2. Teste GetByID
	t.Run("Get Payment By ID", func(t *testing.T) {
		p, err := repo.GetByID(ctx, payment.ID)
		if err != nil || p == nil {
			t.Fatalf("falha ao buscar por ID: %v", err)
		}
		if p.ExternalReference != payment.ExternalReference {
			t.Errorf("esperava ref %s, obteve %s", payment.ExternalReference, p.ExternalReference)
		}
	})

	// 3. Teste GetByExternalReference (GSI)
	t.Run("Get Payment By External Reference (GSI)", func(t *testing.T) {
		p, err := repo.GetByExternalReference(ctx, payment.ExternalReference)
		if err != nil || p == nil {
			t.Fatalf("falha ao buscar por GSI: %v", err)
		}
		if p.ID != payment.ID {
			t.Errorf("esperava ID %s, obteve %s", payment.ID, p.ID)
		}
	})

	// 4. Teste UpdateStatus
	t.Run("Update Status", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, payment.ID, domain.StatusApproved)
		if err != nil {
			t.Fatalf("falha ao atualizar status: %v", err)
		}

		p, _ := repo.GetByID(ctx, payment.ID)
		if p.Status != domain.StatusApproved {
			t.Errorf("esperava status approved, obteve %s", p.Status)
		}
	})
}
