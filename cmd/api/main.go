// cmd/api — точка входа REST API сервера FreeWay VPN.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/freeway-vpn/backend/internal/bootstrap"
	apimw "github.com/freeway-vpn/backend/internal/handler/api/middleware"
	"github.com/freeway-vpn/backend/internal/infrastructure/config"
	"github.com/freeway-vpn/backend/internal/worker"
	"github.com/go-chi/chi/v5"
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

	app, err := bootstrap.NewAPI(cfg)
	if err != nil {
		slog.Error("bootstrap failed", "error", err)
		os.Exit(1)
	}

	r := chi.NewRouter()
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP)
	r.Use(apimw.RequestLogger)
	r.Use(chiMiddleware.Compress(1, "application/json", "text/plain"))

	r.With(apimw.SubscriptionRateLimit).Get("/sub/{token}", app.SubHandler.GetSubscription)

	r.With(apimw.TelegramAuthRateLimit).Post("/api/v1/auth/tg", app.AuthHandler.TelegramAuth)

	r.Post("/api/v1/payments/webhook", app.WebhookHandler.YookassaWebhook)
	r.With(chiMiddleware.Throttle(20)).Get("/api/v1/routing/lists", app.RoutingHandler.GetLists)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(apimw.JWTAuth(cfg.JWT.Secret))

		r.Route("/users", func(r chi.Router) {
			r.Get("/me", app.UserHandler.GetMe)
			r.Get("/me/subscription", app.UserHandler.GetSubscription)
		})

		r.Route("/payments", func(r chi.Router) {
			r.Post("/", app.PaymentHandler.CreatePayment)
			r.Get("/{id}/status", app.PaymentHandler.GetPaymentStatus)
		})
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		sqlDB, err := app.DB.DB()
		if err != nil {
			slog.Error("health ready: sql db", "error", err)
			http.Error(w, `{"status":"not ready","reason":"db"}`, http.StatusServiceUnavailable)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := sqlDB.PingContext(ctx); err != nil {
			slog.Error("health ready: ping", "error", err)
			http.Error(w, `{"status":"not ready","reason":"db ping"}`, http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()
	go worker.NewPaymentWorker(app.PaymentUC).Start(ctx)
	go worker.NewSubscriptionWorker(app.SubUC).Start(ctx)
	go worker.NewNodeHealthWorker(app.NodeUC).Start(ctx)
	go worker.NewDomainHealthWorker(app.NodeRepo).Start(ctx)
	go worker.NewRoutingUpdateWorker(app.RoutingUC).Start(ctx)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("api server started", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	slog.Info("shutting down...")
	cancelWorkers()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped")
}
