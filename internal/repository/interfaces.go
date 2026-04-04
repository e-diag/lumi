// Пакет repository определяет интерфейсы доступа к данным.
package repository

import (
	"context"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/google/uuid"
)

// UserRepository — операции с пользователями.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error)
	GetBySubToken(ctx context.Context, token string) (*domain.User, error)
	Count(ctx context.Context) (int64, error)
	List(ctx context.Context, query string, limit, offset int) ([]*domain.User, int64, error)
	CountCreatedBetween(ctx context.Context, from, to time.Time) (int64, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SubscriptionRepository — операции с подписками.
type SubscriptionRepository interface {
	Create(ctx context.Context, sub *domain.Subscription) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error)
	ListExpiredBefore(ctx context.Context, t time.Time) ([]*domain.Subscription, error)
	ListExpiringBetween(ctx context.Context, from, to time.Time) ([]*domain.Subscription, error)
	CountActive(ctx context.Context, now time.Time) (int64, error)
	CountActiveByTier(ctx context.Context, tier domain.SubscriptionTier, now time.Time) (int64, error)
	// CountExpired — подписки с expires_at < now (включая уже даунгрейднутые записи).
	CountExpired(ctx context.Context, now time.Time) (int64, error)
	Update(ctx context.Context, sub *domain.Subscription) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// NodeRepository — операции с нодами.
type NodeRepository interface {
	GetAll(ctx context.Context) ([]*domain.Node, error)
	GetByRegion(ctx context.Context, region domain.NodeRegion) ([]*domain.Node, error)
	// GetByRegionWithTopology возвращает активные ноды региона с Preload Inbounds и Domains (для подписки).
	GetByRegionWithTopology(ctx context.Context, region domain.NodeRegion) ([]*domain.Node, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Node, error)
	Create(ctx context.Context, node *domain.Node) error
	Update(ctx context.Context, node *domain.Node) error
	// ListActiveNodeDomains — домены с is_active (включая временно заблокированные — воркер снимает блок).
	ListActiveNodeDomains(ctx context.Context) ([]*domain.NodeDomain, error)
	UpdateNodeDomain(ctx context.Context, row *domain.NodeDomain) error
}

// PaymentRepository — операции с платежами.
type PaymentRepository interface {
	Create(ctx context.Context, payment *domain.Payment) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error)
	GetByYookassaID(ctx context.Context, yookassaID string) (*domain.Payment, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Payment, error)
	ListPendingOlderThan(ctx context.Context, t time.Time) ([]*domain.Payment, error)
	ListByFilter(ctx context.Context, status string, from, to *time.Time, limit, offset int) ([]*domain.Payment, int64, error)
	SumSucceededBetween(ctx context.Context, from, to time.Time) (int64, error)
	CountSucceededBetween(ctx context.Context, from, to time.Time) (int64, error)
	Update(ctx context.Context, payment *domain.Payment) error

	// ClaimSucceededByYookassaID атомарно переводит платёж pending→succeeded по внешнему ID.
	// Второй аргумент: true, если переход выполнен этим вызовом (нужна активация подписки).
	ClaimSucceededByYookassaID(ctx context.Context, yookassaID string) (*domain.Payment, bool, error)
	// ClaimCanceledByYookassaID — pending→canceled.
	ClaimCanceledByYookassaID(ctx context.Context, yookassaID string) (*domain.Payment, bool, error)
	// ClaimSucceededByID — pending→succeeded по внутреннему UUID (опрос провайдера воркером).
	ClaimSucceededByID(ctx context.Context, id uuid.UUID) (*domain.Payment, bool, error)
	// ClaimCanceledByID — pending→canceled по внутреннему UUID.
	ClaimCanceledByID(ctx context.Context, id uuid.UUID) (*domain.Payment, bool, error)

	// ConsumePaymentActivation резервирует право на активацию подписки по платежу (один раз на payment_id).
	ConsumePaymentActivation(ctx context.Context, paymentID uuid.UUID) (consumed bool, err error)
	// ReleasePaymentActivation снимает резерв при ошибке активации (повтор webhook/воркера).
	ReleasePaymentActivation(ctx context.Context, paymentID uuid.UUID) error
}

// RoutingRepository — операции с routing-правилами.
type RoutingRepository interface {
	GetAll(ctx context.Context) ([]*domain.RoutingRule, error)
	GetActive(ctx context.Context) ([]*domain.RoutingRule, error)
	SaveDomains(ctx context.Context, source string, action domain.RouteAction, domains []string) error
	GetRoutingList(ctx context.Context) (*domain.RoutingList, error)
	GetVersion(ctx context.Context) (string, error)
	AddManualDomain(ctx context.Context, domain string, action domain.RouteAction) error
	DeleteManualDomain(ctx context.Context, domain string) error
	Create(ctx context.Context, rule *domain.RoutingRule) error
	Update(ctx context.Context, rule *domain.RoutingRule) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// BotAntiAbuseRepository — аудит триалов и реферальных бонусов из Telegram-бота.
type BotAntiAbuseRepository interface {
	CountTrialGrantsByIP(ctx context.Context, ip string) (int64, error)
	CountTrialGrantsGloballySince(ctx context.Context, since time.Time) (int64, error)
	InsertTrialSignup(ctx context.Context, row *domain.BotTrialSignup) error
	GetFirstTrialSignupIPForUser(ctx context.Context, userID uuid.UUID) (string, error)
	CountReferralGrantsSince(ctx context.Context, inviterID uuid.UUID, since time.Time) (int64, error)
	InsertReferralGrant(ctx context.Context, row *domain.ReferralBonusLog) error
}

// PlanRepository — тарифы из БД.
type PlanRepository interface {
	Create(ctx context.Context, p *domain.Plan) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error)
	GetByCode(ctx context.Context, code string) (*domain.Plan, error)
	ListActive(ctx context.Context) ([]*domain.Plan, error)
	ListAll(ctx context.Context) ([]*domain.Plan, error)
	Update(ctx context.Context, p *domain.Plan) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ProductSettingsRepository — глобальные настройки продукта (одна строка).
type ProductSettingsRepository interface {
	Get(ctx context.Context) (*domain.ProductSettings, error)
	Upsert(ctx context.Context, s *domain.ProductSettings) error
}

// VPNServerRepository — учёт серверов для админки.
type VPNServerRepository interface {
	Create(ctx context.Context, s *domain.VPNServer) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.VPNServer, error)
	ListAll(ctx context.Context) ([]*domain.VPNServer, error)
	Count(ctx context.Context) (int64, error)
	Update(ctx context.Context, s *domain.VPNServer) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// AccessProbeRepository — мягкий учёт обращений к подписке (последние N записей на пользователя).
type AccessProbeRepository interface {
	Append(ctx context.Context, userID uuid.UUID, ip, userAgent string, keep int) error
}
