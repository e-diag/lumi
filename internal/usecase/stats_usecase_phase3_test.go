package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/usecase"
	"github.com/google/uuid"
)

type statsUserRepo struct{}

func (r *statsUserRepo) Create(context.Context, *domain.User) error                                   { return nil }
func (r *statsUserRepo) GetByID(context.Context, uuid.UUID) (*domain.User, error)                     { return nil, domain.ErrUserNotFound }
func (r *statsUserRepo) GetByTelegramID(context.Context, int64) (*domain.User, error)                 { return nil, domain.ErrUserNotFound }
func (r *statsUserRepo) GetBySubToken(context.Context, string) (*domain.User, error)                  { return nil, domain.ErrUserNotFound }
func (r *statsUserRepo) Update(context.Context, *domain.User) error                                    { return nil }
func (r *statsUserRepo) Delete(context.Context, uuid.UUID) error                                       { return nil }
func (r *statsUserRepo) Count(context.Context) (int64, error)                                          { return 100, nil }
func (r *statsUserRepo) List(context.Context, string, int, int) ([]*domain.User, int64, error)        { return nil, 0, nil }
func (r *statsUserRepo) CountCreatedBetween(context.Context, time.Time, time.Time) (int64, error)     { return 3, nil }

type statsSubRepo struct{}

func (r *statsSubRepo) Create(context.Context, *domain.Subscription) error                                       { return nil }
func (r *statsSubRepo) GetByUserID(context.Context, uuid.UUID) (*domain.Subscription, error)                    { return nil, domain.ErrSubscriptionNotFound }
func (r *statsSubRepo) ListExpiredBefore(context.Context, time.Time) ([]*domain.Subscription, error)            { return nil, nil }
func (r *statsSubRepo) ListExpiringBetween(context.Context, time.Time, time.Time) ([]*domain.Subscription, error) { return nil, nil }
func (r *statsSubRepo) Update(context.Context, *domain.Subscription) error                                       { return nil }
func (r *statsSubRepo) Delete(context.Context, uuid.UUID) error                                                  { return nil }
func (r *statsSubRepo) CountActive(context.Context, time.Time) (int64, error)                                   { return 25, nil }
func (r *statsSubRepo) CountActiveByTier(_ context.Context, tier domain.SubscriptionTier, _ time.Time) (int64, error) {
	if tier == domain.TierBasic {
		return 10, nil
	}
	if tier == domain.TierPremium {
		return 15, nil
	}
	return 0, nil
}

type statsPaymentRepo struct{}

func (r *statsPaymentRepo) Create(context.Context, *domain.Payment) error                                  { return nil }
func (r *statsPaymentRepo) GetByID(context.Context, uuid.UUID) (*domain.Payment, error)                    { return nil, domain.ErrPaymentNotFound }
func (r *statsPaymentRepo) GetByYookassaID(context.Context, string) (*domain.Payment, error)               { return nil, domain.ErrPaymentNotFound }
func (r *statsPaymentRepo) GetByUserID(context.Context, uuid.UUID) ([]*domain.Payment, error)              { return nil, nil }
func (r *statsPaymentRepo) ListPendingOlderThan(context.Context, time.Time) ([]*domain.Payment, error)     { return nil, nil }
func (r *statsPaymentRepo) Update(context.Context, *domain.Payment) error                                   { return nil }
func (r *statsPaymentRepo) ListByFilter(context.Context, string, *time.Time, *time.Time, int, int) ([]*domain.Payment, int64, error) {
	return []*domain.Payment{{YookassaID: "p1", UserID: uuid.New(), Tier: domain.TierBasic, AmountRub: 149, Status: domain.PaymentSucceeded, CreatedAt: time.Now()}}, 1, nil
}
func (r *statsPaymentRepo) SumSucceededBetween(context.Context, time.Time, time.Time) (int64, error) { return 1000, nil }
func (r *statsPaymentRepo) CountSucceededBetween(context.Context, time.Time, time.Time) (int64, error) {
	return 5, nil
}

type statsNodeRepo struct{}

func (r *statsNodeRepo) GetAll(context.Context) ([]*domain.Node, error) {
	return []*domain.Node{{ID: uuid.New(), Name: "EU-NL", Region: domain.RegionEU, Active: true, LatencyMs: 40}}, nil
}
func (r *statsNodeRepo) GetByRegion(context.Context, domain.NodeRegion) ([]*domain.Node, error) { return nil, nil }
func (r *statsNodeRepo) GetByID(context.Context, uuid.UUID) (*domain.Node, error)                { return nil, domain.ErrNodeNotFound }
func (r *statsNodeRepo) Create(context.Context, *domain.Node) error                               { return nil }
func (r *statsNodeRepo) Update(context.Context, *domain.Node) error                               { return nil }

func TestStatsUseCase_GetDashboardStats(t *testing.T) {
	uc := usecase.NewStatsUseCase(&statsUserRepo{}, &statsSubRepo{}, &statsPaymentRepo{}, &statsNodeRepo{})
	stats, err := uc.GetDashboardStats(context.Background())
	if err != nil {
		t.Fatalf("GetDashboardStats error: %v", err)
	}
	if stats.TotalUsers != 100 {
		t.Fatalf("unexpected total users: %d", stats.TotalUsers)
	}
	if len(stats.Nodes) == 0 {
		t.Fatal("expected nodes in dashboard stats")
	}
}

