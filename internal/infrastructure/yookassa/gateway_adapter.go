package yookassa

import (
	"context"
	"fmt"

	"github.com/freeway-vpn/backend/internal/usecase"
)

// GatewayAdapter адаптирует инфраструктурный Client под интерфейс usecase.PaymentGateway.
type GatewayAdapter struct {
	c *Client
}

func NewGatewayAdapter(c *Client) *GatewayAdapter {
	return &GatewayAdapter{c: c}
}

func (a *GatewayAdapter) CreatePayment(ctx context.Context, req usecase.PaymentGatewayCreateRequest) (*usecase.PaymentGatewayPayment, error) {
	resp, err := a.c.CreatePayment(ctx, CreatePaymentRequest{
		Amount: Amount{
			Value:    req.AmountValue,
			Currency: req.Currency,
		},
		Confirmation: Confirmation{
			Type:      "redirect",
			ReturnURL: req.ReturnURL,
		},
		Description: req.Description,
		Metadata:    req.Metadata,
		Capture:     req.Capture,
	})
	if err != nil {
		return nil, err
	}
	return &usecase.PaymentGatewayPayment{
		ID:              resp.ID,
		Status:          resp.Status,
		ConfirmationURL: resp.Confirmation.ConfirmationURL,
	}, nil
}

func (a *GatewayAdapter) GetPayment(ctx context.Context, providerID string) (*usecase.PaymentGatewayPayment, error) {
	resp, err := a.c.GetPayment(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("yookassa: empty response")
	}
	return &usecase.PaymentGatewayPayment{
		ID:              resp.ID,
		Status:          resp.Status,
		ConfirmationURL: resp.Confirmation.ConfirmationURL,
	}, nil
}

