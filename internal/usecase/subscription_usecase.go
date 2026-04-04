package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/repository"
	"github.com/google/uuid"
)

type subscriptionUseCase struct {
	subRepo  repository.SubscriptionRepository
	userRepo repository.UserRepository
	panel    VPNPanelClient
}

// NewSubscriptionUseCase создаёт реализацию SubscriptionUseCase.
// panel может быть nil — изменения только в БД (без вызовов 3x-ui).
func NewSubscriptionUseCase(subRepo repository.SubscriptionRepository, userRepo repository.UserRepository, panel VPNPanelClient) SubscriptionUseCase {
	return &subscriptionUseCase{
		subRepo:  subRepo,
		userRepo: userRepo,
		panel:    panel,
	}
}

// GetUserSubscription возвращает текущую подписку пользователя.
func (uc *subscriptionUseCase) GetUserSubscription(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	sub, err := uc.subRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("usecase: get subscription: %w", err)
	}
	return sub, nil
}

// ActivateSubscription создаёт или обновляет подписку пользователя.
func (uc *subscriptionUseCase) ActivateSubscription(ctx context.Context, userID uuid.UUID, tier domain.SubscriptionTier, days int) (*domain.Subscription, error) {
	if days <= 0 {
		return nil, fmt.Errorf("usecase: activate subscription: invalid days: %d", days)
	}
	if _, ok := domain.TierLimitsMap[tier]; !ok {
		return nil, fmt.Errorf("usecase: activate subscription: invalid tier: %s", tier)
	}

	existing, err := uc.subRepo.GetByUserID(ctx, userID)
	if err != nil && !errors.Is(err, domain.ErrSubscriptionNotFound) {
		return nil, fmt.Errorf("usecase: activate subscription: %w", err)
	}

	if existing != nil {
		// Если подписка активна — продлеваем, иначе считаем от now.
		base := time.Now()
		if existing.ExpiresAt.After(base) {
			base = existing.ExpiresAt
		}

		updated := &domain.Subscription{
			ID:        existing.ID,
			UserID:    userID,
			Tier:      tier,
			ExpiresAt: base.AddDate(0, 0, days),
		}
		if err := uc.subRepo.Update(ctx, updated); err != nil {
			return nil, fmt.Errorf("usecase: activate subscription update: %w", err)
		}
		if err := uc.applyTierSideEffects(ctx, userID, tier, &updated.ExpiresAt); err != nil {
			return nil, fmt.Errorf("usecase: activate subscription side effects: %w", err)
		}
		return updated, nil
	}

	sub := domain.NewSubscription(userID, tier, days)
	if err := uc.subRepo.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("usecase: activate subscription create: %w", err)
	}
	if err := uc.applyTierSideEffects(ctx, userID, tier, &sub.ExpiresAt); err != nil {
		return nil, fmt.Errorf("usecase: activate subscription side effects: %w", err)
	}
	return sub, nil
}

// ExtendSubscription продлевает текущую подписку на указанное количество дней.
func (uc *subscriptionUseCase) ExtendSubscription(ctx context.Context, userID uuid.UUID, days int) (*domain.Subscription, error) {
	sub, err := uc.subRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("usecase: extend subscription: %w", err)
	}

	extended := &domain.Subscription{
		ID:        sub.ID,
		UserID:    sub.UserID,
		Tier:      sub.Tier,
		ExpiresAt: sub.ExpiresAt.AddDate(0, 0, days),
	}
	if err := uc.subRepo.Update(ctx, extended); err != nil {
		return nil, fmt.Errorf("usecase: extend subscription update: %w", err)
	}
	if err := uc.applyTierSideEffects(ctx, userID, sub.Tier, &extended.ExpiresAt); err != nil {
		return nil, fmt.Errorf("usecase: extend subscription side effects: %w", err)
	}
	return extended, nil
}

func (uc *subscriptionUseCase) DeactivateSubscription(ctx context.Context, userID uuid.UUID) error {
	sub, err := uc.subRepo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			return nil
		}
		return fmt.Errorf("usecase: deactivate subscription: %w", err)
	}
	sub.Tier = domain.TierFree
	sub.ExpiresAt = time.Now()
	if err := uc.subRepo.Update(ctx, sub); err != nil {
		return fmt.Errorf("usecase: deactivate subscription update: %w", err)
	}
	if err := uc.applyTierSideEffects(ctx, userID, domain.TierFree, &sub.ExpiresAt); err != nil {
		return fmt.Errorf("usecase: deactivate subscription side effects: %w", err)
	}
	return nil
}

func (uc *subscriptionUseCase) ExpireOld(ctx context.Context) error {
	now := time.Now()
	subs, err := uc.subRepo.ListExpiredBefore(ctx, now)
	if err != nil {
		return fmt.Errorf("usecase: expire old: %w", err)
	}

	for _, sub := range subs {
		// Даунгрейд до free (и выставляем лимиты/панель).
		sub.Tier = domain.TierFree
		sub.ExpiresAt = now
		if err := uc.subRepo.Update(ctx, sub); err != nil {
			return fmt.Errorf("usecase: expire old: update subscription: %w", err)
		}
		if err := uc.applyTierSideEffects(ctx, sub.UserID, domain.TierFree, &sub.ExpiresAt); err != nil {
			return fmt.Errorf("usecase: expire old: side effects: %w", err)
		}
	}
	return nil
}

func (uc *subscriptionUseCase) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	sub, err := uc.subRepo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("usecase: get active subscription: %w", err)
	}
	if !sub.IsActive() {
		return nil, nil
	}
	return sub, nil
}

func (uc *subscriptionUseCase) GetExpiringIn3Days(ctx context.Context) ([]*domain.Subscription, error) {
	from := time.Now().Add(72 * time.Hour)
	to := from.Add(24 * time.Hour)
	subs, err := uc.subRepo.ListExpiringBetween(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("usecase: get expiring in 3 days: %w", err)
	}
	return subs, nil
}

// AddBonusDays продлевает подписку на days от max(now, текущий expires_at). Если подписки нет — создаёт Basic на days.
func (uc *subscriptionUseCase) AddBonusDays(ctx context.Context, userID uuid.UUID, days int) error {
	if days <= 0 {
		return fmt.Errorf("usecase: add bonus days: invalid days: %d", days)
	}
	sub, err := uc.subRepo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			if _, aerr := uc.ActivateSubscription(ctx, userID, domain.TierBasic, days); aerr != nil {
				return fmt.Errorf("usecase: add bonus days activate: %w", aerr)
			}
			return nil
		}
		return fmt.Errorf("usecase: add bonus days: %w", err)
	}
	base := time.Now()
	if sub.ExpiresAt.After(base) {
		base = sub.ExpiresAt
	}
	newExp := base.AddDate(0, 0, days)
	updated := &domain.Subscription{
		ID:        sub.ID,
		UserID:    userID,
		Tier:      sub.Tier,
		ExpiresAt: newExp,
	}
	if err := uc.subRepo.Update(ctx, updated); err != nil {
		return fmt.Errorf("usecase: add bonus days update: %w", err)
	}
	if err := uc.applyTierSideEffects(ctx, userID, updated.Tier, &newExp); err != nil {
		return fmt.Errorf("usecase: add bonus days side effects: %w", err)
	}
	return nil
}

func (uc *subscriptionUseCase) applyTierSideEffects(ctx context.Context, userID uuid.UUID, tier domain.SubscriptionTier, expiresAt *time.Time) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	limits := domain.TierLimitsMap[tier]
	user.DeviceLimit = limits.Devices

	if uc.panel != nil {
		res, err := uc.panel.SyncUserAccess(ctx, user, tier, expiresAt)
		if err != nil {
			return fmt.Errorf("vpn panel sync: %w", err)
		}
		if res != nil {
			if res.ClientUUID != "" {
				user.PanelClientUUID = res.ClientUUID
			}
			if res.SubID != "" {
				user.PanelSubID = res.SubID
			}
		}
	}
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}
