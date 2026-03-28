// Пакет usecase определяет интерфейсы бизнес-логики приложения.
package usecase

import (
	"context"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/google/uuid"
)

// UserUseCase — бизнес-логика управления пользователями.
type UserUseCase interface {
	Register(ctx context.Context, telegramID int64, username string) (*domain.User, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	// GetBySubToken возвращает пользователя по токену подписки (GET /sub/{token}).
	GetBySubToken(ctx context.Context, token string) (*domain.User, error)
	List(ctx context.Context, query string, page, pageSize int) ([]*domain.User, int64, error)
}

// TelegramStartOutcome — результат обработки /start в пользовательском боте.
type TelegramStartOutcome struct {
	IsNewUser        bool
	TrialGranted     bool
	TrialSkippedByIP bool // триал не выдан из‑за лимита по сети (аккаунт создан)
}

// TelegramClientMeta — опциональные данные клиента (IP/UA), если доступны (например за reverse-proxy).
type TelegramClientMeta struct {
	IP        string
	UserAgent string
}

// TelegramBotUserUseCase — онбординг пользователя из Telegram (/start, рефералы, триал).
type TelegramBotUserUseCase interface {
	// OnStart создаёт пользователя при необходимости, выдаёт приветственный триал, начисляет бонус пригласившему.
	OnStart(ctx context.Context, telegramID int64, username string, referrerUserID *uuid.UUID, client TelegramClientMeta) (*domain.User, *TelegramStartOutcome, error)
}

// AccessProbeUseCase — мягкий учёт обращений к URL подписки.
type AccessProbeUseCase interface {
	RecordSubscriptionFetch(ctx context.Context, userID uuid.UUID, ip, userAgent string) error
}

// PaymentSuccessNotifier — уведомление пользователя в Telegram после оплаты.
type PaymentSuccessNotifier interface {
	NotifySubscriptionPaid(ctx context.Context, userID uuid.UUID) error
}

// SubscriptionUseCase — бизнес-логика управления подписками.
type SubscriptionUseCase interface {
	GetUserSubscription(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error)
	ActivateSubscription(ctx context.Context, userID uuid.UUID, tier domain.SubscriptionTier, days int) (*domain.Subscription, error)
	ExtendSubscription(ctx context.Context, userID uuid.UUID, days int) (*domain.Subscription, error)
	DeactivateSubscription(ctx context.Context, userID uuid.UUID) error

	// ExpireOld деактивирует истёкшие подписки → даунгрейд до Free.
	ExpireOld(ctx context.Context) error

	// GetActiveByUserID возвращает активную подписку пользователя (nil, nil если нет активной).
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error)

	// GetExpiringIn3Days возвращает подписки, истекающие примерно через 3 дня.
	GetExpiringIn3Days(ctx context.Context) ([]*domain.Subscription, error)

	// AddBonusDays добавляет дни к текущей подписке (от max(now, expires_at)); если подписки нет — создаёт Basic на days.
	AddBonusDays(ctx context.Context, userID uuid.UUID, days int) error
}

// NodeUseCase — бизнес-логика работы с нодами.
type NodeUseCase interface {
	GetAllNodes(ctx context.Context) ([]*domain.Node, error)
	GetNodesForTier(ctx context.Context, tier domain.SubscriptionTier) ([]*domain.Node, error)
	UpdateNode(ctx context.Context, node *domain.Node) error
}

// ConfigUseCase — генерация VPN-конфигурации для пользователя.
type ConfigUseCase interface {
	// GenerateSubscription возвращает base64-закодированный список VLESS-конфигов.
	GenerateSubscription(ctx context.Context, userUUID uuid.UUID) (string, error)
}

// StatsUseCase — статистика для менеджерской панели.
type StatsUseCase interface {
	GetTotalUsers(ctx context.Context) (int64, error)
	GetActiveSubscriptions(ctx context.Context) (int64, error)
	GetDashboardStats(ctx context.Context) (*DashboardStats, error)
	GetFinanceStats(ctx context.Context, period string) (*FinanceStats, error)
}

// PaymentUseCase — бизнес-логика платежей (ЮKassa + активация подписок).
type PaymentUseCase interface {
	// CreatePayment создаёт платёж и возвращает сущность платежа + ссылку для оплаты.
	CreatePayment(ctx context.Context, userID uuid.UUID, tier domain.SubscriptionTier, days int) (*domain.Payment, string, error)

	// HandleWebhook обрабатывает событие от платёжного провайдера.
	HandleWebhook(ctx context.Context, event WebhookEvent) error

	// GetPendingPayments возвращает платежи pending старше 5 минут.
	GetPendingPayments(ctx context.Context) ([]*domain.Payment, error)

	// CheckAndUpdatePayment обновляет статус платежа по данным провайдера.
	CheckAndUpdatePayment(ctx context.Context, paymentID uuid.UUID) error

	// GetByID возвращает платеж по UUID.
	GetByID(ctx context.Context, paymentID uuid.UUID) (*domain.Payment, error)

	// CancelStale отменяет платеж, если он всё ещё pending.
	CancelStale(ctx context.Context, paymentID uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Payment, error)
	ListByFilter(ctx context.Context, status string, period string, page, pageSize int) ([]*domain.Payment, int64, error)
}

// WebhookEvent — событие вебхука платежного провайдера (минимально, под ЮKassa).
type WebhookEvent struct {
	Type   string `json:"type"`
	Event  string `json:"event"`
	Object struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		// Metadata может быть вложенным объектом; для usecase важны строки.
		Metadata map[string]string `json:"metadata"`
	} `json:"object"`
}

// PaymentGateway — абстракция платёжного провайдера.
type PaymentGateway interface {
	CreatePayment(ctx context.Context, req PaymentGatewayCreateRequest) (*PaymentGatewayPayment, error)
	GetPayment(ctx context.Context, providerID string) (*PaymentGatewayPayment, error)
}

type PaymentGatewayCreateRequest struct {
	AmountValue string            // "299.00"
	Currency    string            // "RUB"
	ReturnURL   string            // куда вернётся пользователь после оплаты
	Description string
	Metadata    map[string]string // user_id, tier, days
	Capture     bool
}

type PaymentGatewayPayment struct {
	ID              string
	Status          string // pending/succeeded/canceled
	ConfirmationURL string
}

// RemnawaveClient — абстракция панели управления Xray-нодами.
type RemnawaveClient interface {
	CreateUser(ctx context.Context, userUUID, username string, tier domain.SubscriptionTier) error
	DeleteUser(ctx context.Context, userUUID string) error
	UpdateUserExpiry(ctx context.Context, userUUID string, expiresAt *time.Time) error
}

type RoutingUseCase interface {
	GetLists(ctx context.Context) (*RoutingLists, error)
	UpdateFromAntifilter(ctx context.Context) error
	AddDomain(ctx context.Context, domain string, action domain.RouteAction) error
	RemoveDomain(ctx context.Context, domain string) error
}

type RoutingLists struct {
	Version  string   `json:"version"`
	ProxyEU  []string `json:"proxy_eu"`
	ProxyUSA []string `json:"proxy_usa"`
	// Direct — домены которые всегда идут напрямую, без VPN.
	Direct []string `json:"direct"`
	Manual []string `json:"manual,omitempty"`
	// DirectStrictMode — direct правила выше любого CDN-fallback.
	DirectStrictMode bool `json:"direct_strict_mode"`
	Meta             RoutingMeta `json:"meta"`
}

type RoutingMeta struct {
	CDNFallbackExcludesDirect bool   `json:"cdn_fallback_excludes_direct"`
	DirectDomainsNote         string `json:"direct_domains_note"`
}

type DashboardStats struct {
	TotalUsers          int
	FreeUsers           int
	BasicUsers          int
	PremiumUsers        int
	ActiveSubscriptions int
	RevenueToday        float64
	RevenueMonth        float64
	PaymentsToday       int
	Nodes               []NodeStatus
	RecentPayments      []PaymentSummary
	NewUsersPerDay      []DailyCount
}

type NodeStatus struct {
	Name      string
	Region    string
	IsOnline  bool
	LatencyMs int
	Online    int
}

type PaymentSummary struct {
	YookassaID string
	UserID     string
	Tier       string
	AmountRub  int
	Status     string
	CreatedAt  time.Time
}

type DailyCount struct {
	Date  string
	Count int
}

type FinanceStats struct {
	Period       string
	Revenue      float64
	Payments     int
	Recent       []PaymentSummary
	AverageCheck float64
}
