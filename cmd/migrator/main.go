// cmd/migrator — применение схемы БД (GORM AutoMigrate, индексы, идемпотентный seed через database.Connect).
// Запускайте перед обновлением стека в проде: тот же образ, что и api, после docker compose pull.
package main

import (
	"log/slog"
	"os"

	"github.com/freeway-vpn/backend/internal/infrastructure/config"
	"github.com/freeway-vpn/backend/internal/infrastructure/database"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg, err := config.Load(config.ConfigPath())
	if err != nil {
		slog.Error("migrator: load config", "error", err)
		os.Exit(1)
	}
	if err := cfg.ValidateMigrator(); err != nil {
		slog.Error("migrator: invalid config", "error", err)
		os.Exit(1)
	}

	if _, err := database.Connect(cfg.Database.DSN); err != nil {
		slog.Error("migrator: database connect/migrate failed", "error", err)
		os.Exit(1)
	}
	slog.Info("migrator: schema and bootstrap steps completed successfully")
}
