package sns

import (
	"context"
	"encoding/json"
	"os"

	"github.com/alexssanderFonseca/pagamento/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type Client struct {
	snsClient *sns.Client
	topicARN  string
}

func NewClient(cfg aws.Config) *Client {
	return &Client{
		snsClient: sns.NewFromConfig(cfg),
		topicARN:  os.Getenv("AWS_SNS_TOPIC_ARN"),
	}
}

func (c *Client) PublishPaymentProcessed(ctx context.Context, event domain.PaymentProcessedEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = c.snsClient.Publish(ctx, &sns.PublishInput{
		Message:  aws.String(string(payload)),
		TopicArn: aws.String(c.topicARN),
		MessageAttributes: map[string]sns.MessageAttributesValue{
			"event_type": {
				DataType:    aws.String("String"),
				StringValue: aws.String("payment_processed"),
			},
		},
	})

	return err
}
