// Пакет config отвечает за загрузку конфигурации из YAML-файла
// с подстановкой значений переменных окружения.
package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Config — корневая структура конфигурации приложения.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	Bot      BotConfig      `yaml:"bot"`
	Web      WebConfig      `yaml:"web"`
	Remnawave RemnawaveConfig `yaml:"remnawave"`
	Yookassa YookassaConfig  `yaml:"yookassa"`
}

// ServerConfig — параметры HTTP-сервера.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	BaseURL string `yaml:"base_url"` // публичный URL для генерации ссылок
}

// DatabaseConfig — параметры подключения к PostgreSQL.
type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

// JWTConfig — секрет для подписи JWT-токенов.
type JWTConfig struct {
	Secret string `yaml:"secret"`
}

// BotConfig — токен Telegram-бота.
type BotConfig struct {
	Token    string  `yaml:"token"`
	AdminIDs []int64 `yaml:"admin_ids"`
}

// WebConfig — настройки веб-панели администратора.
type WebConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	AdminToken string `yaml:"admin_token"`
}

// RemnawaveConfig — параметры подключения к Remnawave API.
type RemnawaveConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
}

// YookassaConfig — параметры ЮKassa.
type YookassaConfig struct {
	ShopID    string `yaml:"shop_id"`
	SecretKey string `yaml:"secret_key"`
}

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// Load загружает конфигурацию из YAML-файла и подставляет переменные окружения.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read file: %w", err)
	}

	// Подстановка ${ENV_VAR} → значение из окружения
	expanded := envVarPattern.ReplaceAllStringFunc(string(data), func(match string) string {
		key := envVarPattern.FindStringSubmatch(match)[1]
		return os.Getenv(key)
	})

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("config: parse yaml: %w", err)
	}

	return &cfg, nil
}
