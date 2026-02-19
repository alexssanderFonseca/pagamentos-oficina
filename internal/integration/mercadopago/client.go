package mercadopago

import (
	"context"
	"fmt"
	"os"

	"github.com/alexssanderFonseca/pagamento/internal/domain"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
)

type Client struct {
	httpClient  *resty.Client
	baseURL     string
	accessToken string
}

func NewClient() *Client {
	return &Client{
		httpClient:  resty.New(),
		baseURL:     "https://api.mercadopago.com",
		accessToken: os.Getenv("MERCADO_PAGO_ACCESS_TOKEN"),
	}
}

type OrderRequest struct {
	Type              string       `json:"type"`
	ExternalReference string       `json:"external_reference"`
	TotalAmount       float64      `json:"total_amount"`
	Description       string       `json:"description"`
	Items             []Item       `json:"items"`
	Config            OrderConfig  `json:"config"`
	Transactions      Transactions `json:"transactions"`
}

type OrderConfig struct {
	QR QRConfig `json:"qr"`
}

type QRConfig struct {
	ExternalPOSID string `json:"external_pos_id"`
	Mode          string `json:"mode"`
}

type Transactions struct {
	Payments []TransactionPayment `json:"payments"`
}

type TransactionPayment struct {
	Amount float64 `json:"amount"`
}

type Item struct {
	Title       string  `json:"title"`
	UnitPrice   float64 `json:"unit_price"`
	Quantity    int     `json:"quantity"`
	UnitMeasure string  `json:"unit_measure"`
}

type OrderResponse struct {
	ID           string       `json:"id"`
	TypeResponse TypeResponse `json:"type_response"`
}

type TypeResponse struct {
	QRCodeData string `json:"qr_data"`
}

func (c *Client) CreateQRCodeOrder(ctx context.Context, req domain.CreatePaymentRequest) (string, error) {
	posID := os.Getenv("MERCADO_PAGO_POS_ID")
	url := fmt.Sprintf("%s/v1/orders", c.baseURL)

	orderReq := OrderRequest{
		Type:              "qr",
		ExternalReference: req.ExternalReference,
		TotalAmount:       req.Amount,
		Description:       req.Description,
		Config: OrderConfig{
			QR: QRConfig{
				ExternalPOSID: posID,
				Mode:          "dynamic",
			},
		},
		Transactions: Transactions{
			Payments: []TransactionPayment{
				{Amount: req.Amount},
			},
		},
		Items: []Item{
			{
				Title:       req.Description,
				UnitPrice:   req.Amount,
				Quantity:    1,
				UnitMeasure: "unit",
			},
		},
	}

	var orderResp OrderResponse
	idempotencyKey := uuid.New().String()

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+c.accessToken).
		SetHeader("X-Idempotency-Key", idempotencyKey).
		SetBody(orderReq).
		SetResult(&orderResp).
		Post(url)

	if err != nil {
		return "", err
	}

	if resp.IsError() {
		return "", fmt.Errorf("mercadopago api error: %s", resp.String())
	}

	return orderResp.TypeResponse.QRCodeData, nil
}

func (c *Client) GetPaymentDetails(ctx context.Context, paymentID string) (*domain.MPPaymentResponse, error) {
	url := fmt.Sprintf("%s/v1/payments/%s", c.baseURL, paymentID)

	var paymentResp domain.MPPaymentResponse
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+c.accessToken).
		SetResult(&paymentResp).
		Get(url)

	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, fmt.Errorf("mercadopago api error: %s", resp.String())
	}

	return &paymentResp, nil
}
