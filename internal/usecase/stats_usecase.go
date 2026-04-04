package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/repository"
)

type statsUseCase struct {
	userRepo        repository.UserRepository
	subRepo         repository.SubscriptionRepository
	paymentRepo     repository.PaymentRepository
	nodeRepo        repository.NodeRepository
	vpnServerRepo   repository.VPNServerRepository // может быть nil
}

// NewStatsUseCase создаёт реализацию StatsUseCase.
func NewStatsUseCase(
	userRepo repository.UserRepository,
	subRepo repository.SubscriptionRepository,
	paymentRepo repository.PaymentRepository,
	nodeRepo repository.NodeRepository,
	vpnServerRepo repository.VPNServerRepository,
) StatsUseCase {
	return &statsUseCase{
		userRepo:      userRepo,
		subRepo:       subRepo,
		paymentRepo:   paymentRepo,
		nodeRepo:      nodeRepo,
		vpnServerRepo: vpnServerRepo,
	}
}

// GetTotalUsers возвращает общее количество зарегистрированных пользователей.
func (uc *statsUseCase) GetTotalUsers(ctx context.Context) (int64, error) {
	count, err := uc.userRepo.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("usecase: stats total users: %w", err)
	}
	return count, nil
}

// GetActiveSubscriptions возвращает количество активных (не истекших) подписок.
func (uc *statsUseCase) GetActiveSubscriptions(ctx context.Context) (int64, error) {
	count, err := uc.subRepo.CountActive(ctx, time.Now())
	if err != nil {
		return 0, fmt.Errorf("usecase: stats active subscriptions: %w", err)
	}
	return count, nil
}

func (uc *statsUseCase) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	totalUsers, err := uc.userRepo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("usecase: dashboard total users: %w", err)
	}
	basic, err := uc.subRepo.CountActiveByTier(ctx, domain.TierBasic, now)
	if err != nil {
		return nil, fmt.Errorf("usecase: dashboard basic users: %w", err)
	}
	premium, err := uc.subRepo.CountActiveByTier(ctx, domain.TierPremium, now)
	if err != nil {
		return nil, fmt.Errorf("usecase: dashboard premium users: %w", err)
	}
	active, err := uc.subRepo.CountActive(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("usecase: dashboard active subscriptions: %w", err)
	}
	expiredSubs, err := uc.subRepo.CountExpired(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("usecase: dashboard expired subscriptions: %w", err)
	}

	revenueToday, err := uc.paymentRepo.SumSucceededBetween(ctx, todayStart, now)
	if err != nil {
		return nil, fmt.Errorf("usecase: dashboard revenue today: %w", err)
	}
	revenueMonth, err := uc.paymentRepo.SumSucceededBetween(ctx, monthStart, now)
	if err != nil {
		return nil, fmt.Errorf("usecase: dashboard revenue month: %w", err)
	}
	paymentsToday, err := uc.paymentRepo.CountSucceededBetween(ctx, todayStart, now)
	if err != nil {
		return nil, fmt.Errorf("usecase: dashboard payments today: %w", err)
	}

	var vpnSrvCount int64
	if uc.vpnServerRepo != nil {
		vpnSrvCount, err = uc.vpnServerRepo.Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("usecase: dashboard vpn servers count: %w", err)
		}
	}

	nodes, err := uc.nodeRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("usecase: dashboard nodes: %w", err)
	}
	nodeStatuses := make([]NodeStatus, 0, len(nodes))
	for _, n := range nodes {
		nodeStatuses = append(nodeStatuses, NodeStatus{
			Name:      n.Name,
			Region:    string(n.Region),
			IsOnline:  n.Active,
			LatencyMs: n.LatencyMs,
			Online:    0,
		})
	}

	payments, _, err := uc.paymentRepo.ListByFilter(ctx, "", nil, nil, 10, 0)
	if err != nil {
		return nil, fmt.Errorf("usecase: dashboard recent payments: %w", err)
	}
	recent := make([]PaymentSummary, 0, len(payments))
	for _, p := range payments {
		recent = append(recent, PaymentSummary{
			YookassaID: p.YookassaID,
			UserID:     p.UserID.String(),
			Tier:       string(p.Tier),
			AmountRub:  p.AmountRub,
			Status:     string(p.Status),
			CreatedAt:  p.CreatedAt,
		})
	}

	perDay := make([]DailyCount, 0, 30)
	for i := 29; i >= 0; i-- {
		dayStart := todayStart.AddDate(0, 0, -i)
		dayEnd := dayStart.Add(24 * time.Hour)
		c, err := uc.userRepo.CountCreatedBetween(ctx, dayStart, dayEnd)
		if err != nil {
			return nil, fmt.Errorf("usecase: dashboard new users per day: %w", err)
		}
		perDay = append(perDay, DailyCount{
			Date:  dayStart.Format("2006-01-02"),
			Count: int(c),
		})
	}

	free := int(totalUsers - basic - premium)
	if free < 0 {
		free = 0
	}

	return &DashboardStats{
		TotalUsers:           int(totalUsers),
		FreeUsers:            free,
		BasicUsers:           int(basic),
		PremiumUsers:         int(premium),
		ActiveSubscriptions:  int(active),
		ExpiredSubscriptions: int(expiredSubs),
		RevenueToday:         float64(revenueToday),
		RevenueMonth:         float64(revenueMonth),
		PaymentsToday:        int(paymentsToday),
		VPNServerRecords:     int(vpnSrvCount),
		Nodes:                nodeStatuses,
		RecentPayments:       recent,
		NewUsersPerDay:       perDay,
	}, nil
}

func (uc *statsUseCase) GetFinanceStats(ctx context.Context, period string) (*FinanceStats, error) {
	now := time.Now()
	from := now.AddDate(0, 0, -30)
	switch period {
	case "today":
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":
		from = now.AddDate(0, 0, -7)
	case "month", "":
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default:
		from = now.AddDate(0, 0, -30)
	}

	sum, err := uc.paymentRepo.SumSucceededBetween(ctx, from, now)
	if err != nil {
		return nil, fmt.Errorf("usecase: finance sum: %w", err)
	}
	cnt, err := uc.paymentRepo.CountSucceededBetween(ctx, from, now)
	if err != nil {
		return nil, fmt.Errorf("usecase: finance count: %w", err)
	}
	recentRows, _, err := uc.paymentRepo.ListByFilter(ctx, "", &from, &now, 5, 0)
	if err != nil {
		return nil, fmt.Errorf("usecase: finance recent: %w", err)
	}
	recent := make([]PaymentSummary, 0, len(recentRows))
	for _, p := range recentRows {
		recent = append(recent, PaymentSummary{
			YookassaID: p.YookassaID,
			UserID:     p.UserID.String(),
			Tier:       string(p.Tier),
			AmountRub:  p.AmountRub,
			Status:     string(p.Status),
			CreatedAt:  p.CreatedAt,
		})
	}
	avg := 0.0
	if cnt > 0 {
		avg = float64(sum) / float64(cnt)
	}
	return &FinanceStats{
		Period:       period,
		Revenue:      float64(sum),
		Payments:     int(cnt),
		Recent:       recent,
		AverageCheck: avg,
	}, nil
}
