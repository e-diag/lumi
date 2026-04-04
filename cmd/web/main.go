// cmd/web — веб-панель менеджера FreeWay VPN.
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/freeway-vpn/backend/internal/bootstrap"
	"github.com/freeway-vpn/backend/internal/infrastructure/config"
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

	templateDir := os.Getenv("WEB_TEMPLATE_DIR")

	app, err := bootstrap.NewWeb(cfg, templateDir)
	if err != nil {
		slog.Error("bootstrap failed", "error", err)
		os.Exit(1)
	}

	r := chi.NewRouter()
	app.Handler.RegisterRoutes(r)

	addr := fmt.Sprintf("%s:%d", cfg.Web.Host, cfg.Web.Port)
	slog.Info("web panel started", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("web panel stopped", "error", err)
		os.Exit(1)
	}
}
