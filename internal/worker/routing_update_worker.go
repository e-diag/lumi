package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/freeway-vpn/backend/internal/usecase"
)

type RoutingUpdateWorker struct {
	routingUC usecase.RoutingUseCase
	interval  time.Duration
}

func NewRoutingUpdateWorker(routingUC usecase.RoutingUseCase) *RoutingUpdateWorker {
	return &RoutingUpdateWorker{
		routingUC: routingUC,
		interval:  24 * time.Hour,
	}
}

func (w *RoutingUpdateWorker) Start(ctx context.Context) {
	lists, err := w.routingUC.GetLists(ctx)
	if err != nil {
		slog.Error("routing worker: failed to get current lists", "error", err)
	} else if lists == nil || len(lists.ProxyEU) == 0 {
		if err := w.routingUC.UpdateFromAntifilter(ctx); err != nil {
			slog.Error("routing worker initial update failed", "error", err)
		}
	}
	for {
		wait := durationToNext3UTC(time.Now().UTC())
		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
			if err := w.routingUC.UpdateFromAntifilter(ctx); err != nil {
				slog.Error("routing worker update failed", "error", err)
			}
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

func durationToNext3UTC(now time.Time) time.Duration {
	next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, time.UTC)
	if !now.Before(next) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}

