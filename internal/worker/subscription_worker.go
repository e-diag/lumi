package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/freeway-vpn/backend/internal/usecase"
)

type SubscriptionWorker struct {
	subUC    usecase.SubscriptionUseCase
	interval time.Duration
}

func NewSubscriptionWorker(subUC usecase.SubscriptionUseCase) *SubscriptionWorker {
	return &SubscriptionWorker{
		subUC:    subUC,
		interval: time.Minute,
	}
}

func (w *SubscriptionWorker) Start(ctx context.Context) {
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

func (w *SubscriptionWorker) run(ctx context.Context) {
	if err := w.subUC.ExpireOld(ctx); err != nil {
		slog.Error("subscription worker: expire old failed", "error", err)
	}
	subs, err := w.subUC.GetExpiringIn3Days(ctx)
	if err != nil {
		slog.Error("subscription worker: get expiring failed", "error", err)
		return
	}
	for _, s := range subs {
		if s == nil {
			continue
		}
		slog.Info("subscription expiring soon", "user_id", s.UserID, "tier", s.Tier, "expires_at", s.ExpiresAt)
	}
}
