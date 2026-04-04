package yookassa

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	apiBaseURL = "https://api.yookassa.ru/v3"
)

// Client — минимальный HTTP-клиент ЮKassa (v3).
type Client struct {
	shopID     string
	secretKey  string
	httpClient *http.Client
}

// NewClient создаёт клиент ЮKassa.
func NewClient(shopID, secretKey string) *Client {
	return &Client{
		shopID:    shopID,
		secretKey: secretKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type CreatePaymentRequest struct {
	Amount       Amount            `json:"amount"`
	Confirmation Confirmation      `json:"confirmation"`
	Description  string            `json:"description"`
	Metadata     map[string]string `json:"metadata"` // user_id, tier, days
	Capture      bool              `json:"capture"`
}

type Amount struct {
	Value    string `json:"value"`    // "299.00"
	Currency string `json:"currency"` // "RUB"
}

type Confirmation struct {
	Type      string `json:"type"` // "redirect"
	ReturnURL string `json:"return_url"`
}

type PaymentResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"` // pending/succeeded/canceled
	Confirmation struct {
		ConfirmationURL string `json:"confirmation_url"`
	} `json:"confirmation"`
}

// CreatePayment создаёт платёж в ЮKassa.
// POST https://api.yookassa.ru/v3/payments
// Auth: Basic base64(shopID:secretKey)
// Idempotence-Key: uuid (важно для защиты от двойной оплаты)
func (c *Client) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*PaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("yookassa: marshal create payment request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBaseURL+"/payments", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("yookassa: create request: %w", err)
	}
	c.setHeaders(httpReq)
	httpReq.Header.Set("Idempotence-Key", uuid.NewString())

	var resp PaymentResponse
	if err := c.doJSON(httpReq, &resp); err != nil {
		return nil, fmt.Errorf("yookassa: create payment: %w", err)
	}
	return &resp, nil
}

// GetPayment возвращает статус платежа.
// GET https://api.yookassa.ru/v3/payments/{id}
func (c *Client) GetPayment(ctx context.Context, id string) (*PaymentResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBaseURL+"/payments/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("yookassa: create request: %w", err)
	}
	c.setHeaders(httpReq)

	var resp PaymentResponse
	if err := c.doJSON(httpReq, &resp); err != nil {
		return nil, fmt.Errorf("yookassa: get payment: %w", err)
	}
	return &resp, nil
}

func (c *Client) setHeaders(r *http.Request) {
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Basic "+basicAuth(c.shopID, c.secretKey))
}

func basicAuth(user, pass string) string {
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
}

func (c *Client) doJSON(r *http.Request, out any) error {
	res, err := c.httpClient.Do(r)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		// ЮKassa возвращает JSON с полями type/id/description/parameter, но формат может меняться.
		return fmt.Errorf("http %d: %s", res.StatusCode, string(bytes.TrimSpace(b)))
	}

	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}
