package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/freeway-vpn/backend/internal/usecase"
)

type PaymentWorker struct {
	paymentUC usecase.PaymentUseCase
	interval  time.Duration
}

func NewPaymentWorker(paymentUC usecase.PaymentUseCase) *PaymentWorker {
	return &PaymentWorker{
		paymentUC: paymentUC,
		interval:  30 * time.Second,
	}
}

func (w *PaymentWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.run(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (w *PaymentWorker) run(ctx context.Context) {
	payments, err := w.paymentUC.GetPendingPayments(ctx)
	if err != nil {
		slog.Error("payment worker: get pending", "error", err)
		return
	}

	now := time.Now()
	if len(payments) > 0 {
		slog.Info("payment worker: pending batch", "count", len(payments))
	}
	for _, p := range payments {
		if p == nil {
			continue
		}

		// >24h — отменяем.
		if now.Sub(p.CreatedAt) > 24*time.Hour {
			if err := w.paymentUC.CancelStale(ctx, p.ID); err != nil {
				slog.Error("payment worker: cancel stale failed", "error", err, "payment_id", p.ID)
			}
			continue
		}

		if err := w.paymentUC.CheckAndUpdatePayment(ctx, p.ID); err != nil {
			slog.Error("payment worker: check failed", "error", err, "payment_id", p.ID)
		}
	}
}
