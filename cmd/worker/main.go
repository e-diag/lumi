// cmd/worker — фоновые задачи (платежи, подписки, health нод/доменов, antifilter).
// Запускайте отдельно от cmd/api, чтобы HTTP не делился CPU с воркерами.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/freeway-vpn/backend/internal/bootstrap"
	"github.com/freeway-vpn/backend/internal/infrastructure/config"
	"github.com/freeway-vpn/backend/internal/worker"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg, err := config.Load(config.ConfigPath())
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	app, err := bootstrap.NewWorker(cfg)
	if err != nil {
		slog.Error("bootstrap failed", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	start := func(name string, fn func(context.Context)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			slog.Info("worker goroutine started", "worker", name)
			fn(ctx)
		}()
	}

	start("payment", worker.NewPaymentWorker(app.PaymentUC).Start)
	start("subscription", worker.NewSubscriptionWorker(app.SubUC).Start)
	start("node_health", worker.NewNodeHealthWorker(app.NodeUC).Start)
	start("domain_health", worker.NewDomainHealthWorker(app.NodeRepo).Start)
	start("routing_update", worker.NewRoutingUpdateWorker(app.RoutingUC).Start)

	slog.Info("background workers running")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	slog.Info("shutting down workers...")
	cancel()
	wg.Wait()
	slog.Info("workers stopped")
}
