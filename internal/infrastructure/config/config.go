// Пакет config отвечает за загрузку конфигурации из YAML-файла
// с подстановкой значений переменных окружения.
package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigPath путь к файлу конфигурации: переменная CONFIG_PATH или config.yaml.
func ConfigPath() string {
	p := os.Getenv("CONFIG_PATH")
	if p == "" {
		return "config.yaml"
	}
	return p
}

// Config — корневая структура конфигурации приложения.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	Bot      BotConfig      `yaml:"bot"`
	Web      WebConfig      `yaml:"web"`
	// XUI — панель 3x-ui (провижининг клиентов и источник подписки для клиентов Happ / v2RayTun).
	XUI      XUIConfig      `yaml:"xui"`
	Yookassa YookassaConfig `yaml:"yookassa"`
}

// ServerConfig — параметры HTTP-сервера.
type ServerConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
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
	// Username — имя бота без @ (для реферальных ссылок t.me/username).
	Username string `yaml:"username"`
	// AppURLIOS / AppURLAndroid — ссылки на страницы загрузки клиента (опционально).
	AppURLIOS     string `yaml:"app_url_ios"`
	AppURLAndroid string `yaml:"app_url_android"`
	// PaymentDefaultDays — период оплаты по умолчанию в боте (например 30).
	PaymentDefaultDays int `yaml:"payment_default_days"`
	// MaxTrialsPerIP — максимум выданных триалов на один IP (0 = без лимита).
	MaxTrialsPerIP int `yaml:"max_trials_per_ip"`
	// ReferralBonusMaxPerMonth — потолок реферальных +3 дня у пригласившего в месяц (0 = без лимита).
	ReferralBonusMaxPerMonth int `yaml:"referral_bonus_max_per_month"`
	// SupportURL — ссылка на поддержку (Telegram-группа, @username или веб).
	SupportURL string `yaml:"support_url"`
	// TrialGlobalCapPer24h — максимум выданных триалов за последние 24 ч по всему сервису (0 = без лимита).
	TrialGlobalCapPer24h int `yaml:"trial_global_cap_per_24h"`
}

// WebConfig — настройки веб-панели администратора.
type WebConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	AdminToken string `yaml:"admin_token"`
}

// XUIConfig — доступ к веб-панели 3x-ui и публичному subscription-серверу.
// BaseURL — полный корень панели, включая путь из настроек «URI Path» (например https://host:2053/panel).
type XUIConfig struct {
	BaseURL   string `yaml:"base_url"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	InboundID int    `yaml:"inbound_id"`
	// PublicSubscriptionBaseURL — базовый URL sub-сервера (часто отдельный порт в 3x-ui), без завершающего слэша.
	PublicSubscriptionBaseURL string `yaml:"public_subscription_base_url"`
	// SubscriptionPath — сегмент пути перед subId (в настройках подписки панели), по умолчанию «sub».
	SubscriptionPath string `yaml:"subscription_path"`
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

	applyDefaults(&cfg)
	return &cfg, nil
}

func applyDefaults(c *Config) {
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Web.Host == "" {
		c.Web.Host = "0.0.0.0"
	}
	if c.Web.Port == 0 {
		c.Web.Port = 3000
	}
	if c.Bot.PaymentDefaultDays <= 0 {
		c.Bot.PaymentDefaultDays = 30
	}
	if v := strings.TrimSpace(os.Getenv("XUI_INBOUND_ID")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.XUI.InboundID = n
		}
	}
	if strings.TrimSpace(c.XUI.SubscriptionPath) == "" {
		c.XUI.SubscriptionPath = "sub"
	}
}

// ValidateAPI проверяет обязательные поля для cmd/api.
func (c *Config) ValidateAPI() error {
	if c.Database.DSN == "" {
		return fmt.Errorf("config: DATABASE_DSN is required")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("config: JWT_SECRET is required")
	}
	if len(c.JWT.Secret) < 16 {
		return fmt.Errorf("config: JWT_SECRET must be at least 16 characters")
	}
	if c.Bot.Token == "" {
		return fmt.Errorf("config: TELEGRAM_BOT_TOKEN is required for API auth")
	}
	if c.Yookassa.ShopID == "" || c.Yookassa.SecretKey == "" {
		return fmt.Errorf("config: YOOKASSA_SHOP_ID and YOOKASSA_SECRET_KEY are required for API")
	}
	return nil
}

// ValidateBot проверяет обязательные поля для cmd/bot.
func (c *Config) ValidateBot() error {
	if c.Database.DSN == "" {
		return fmt.Errorf("config: DATABASE_DSN is required")
	}
	if c.Bot.Token == "" {
		return fmt.Errorf("config: TELEGRAM_BOT_TOKEN is required")
	}
	if c.Yookassa.ShopID == "" || c.Yookassa.SecretKey == "" {
		return fmt.Errorf("config: YOOKASSA_SHOP_ID and YOOKASSA_SECRET_KEY are required for bot payments")
	}
	if c.Server.BaseURL == "" {
		return fmt.Errorf("config: BASE_URL is required for payment return URLs")
	}
	return nil
}

// ValidateWeb проверяет обязательные поля для cmd/web.
func (c *Config) ValidateWeb() error {
	if c.Database.DSN == "" {
		return fmt.Errorf("config: DATABASE_DSN is required")
	}
	if c.Web.AdminToken == "" {
		return fmt.Errorf("config: ADMIN_WEB_TOKEN is required for web panel")
	}
	if len(c.Web.AdminToken) < 12 {
		return fmt.Errorf("config: ADMIN_WEB_TOKEN must be at least 12 characters")
	}
	return nil
}
