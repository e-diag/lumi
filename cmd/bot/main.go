// cmd/bot — Telegram-бот FreeWay VPN: пользователи + менеджеры (/manager).
package main

import (
	"context"
	"log/slog"
	"os"

	tgbot "github.com/go-telegram/bot"

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

	app, err := bootstrap.NewBot(cfg)
	if err != nil {
		slog.Error("bootstrap failed", "error", err)
		os.Exit(1)
	}

	b, err := tgbot.New(cfg.Bot.Token)
	if err != nil {
		slog.Error("failed to init telegram bot", "error", err)
		os.Exit(1)
	}
	app.Handler.Register(b)

	slog.Info("telegram bot started")
	b.Start(context.Background())
}
