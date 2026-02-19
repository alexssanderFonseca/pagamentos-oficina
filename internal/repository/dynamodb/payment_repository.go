package dynamodb

import (
	"context"
	"os"
	"time"

	"github.com/alexssanderFonseca/pagamento/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type PaymentRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewPaymentRepository(client *dynamodb.Client) *PaymentRepository {
	tableName := os.Getenv("DYNAMODB_TABLE_NAME")
	if tableName == "" {
		tableName = "Payments"
	}
	return &PaymentRepository{
		client:    client,
		tableName: tableName,
	}
}

func (r *PaymentRepository) Save(ctx context.Context, payment domain.Payment) error {
	item, err := attributevalue.MarshalMap(payment)
	if err != nil {
		return err
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	return err
}

func (r *PaymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, nil
	}

	var payment domain.Payment
	err = attributevalue.UnmarshalMap(result.Item, &payment)
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

func (r *PaymentRepository) GetByExternalReference(ctx context.Context, ref string) (*domain.Payment, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("ExternalReferenceIndex"),
		KeyConditionExpression: aws.String("external_reference = :ref"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":ref": &types.AttributeValueMemberS{Value: ref},
		},
	}

	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	var payment domain.Payment
	err = attributevalue.UnmarshalMap(result.Items[0], &payment)
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

func (r *PaymentRepository) UpdateStatus(ctx context.Context, id string, status domain.PaymentStatus) error {
	_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression:         aws.String("SET #status = :status, updated_at = :updated_at"),
		ExpressionAttributeNames: map[string]string{"#status": "status"},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":     &types.AttributeValueMemberS{Value: string(status)},
			":updated_at": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		},
	})
	return err
}
