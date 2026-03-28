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
	Update(ctx context.Context, sub *domain.Subscription) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// NodeRepository — операции с нодами.
type NodeRepository interface {
	GetAll(ctx context.Context) ([]*domain.Node, error)
	GetByRegion(ctx context.Context, region domain.NodeRegion) ([]*domain.Node, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Node, error)
	Create(ctx context.Context, node *domain.Node) error
	Update(ctx context.Context, node *domain.Node) error
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
