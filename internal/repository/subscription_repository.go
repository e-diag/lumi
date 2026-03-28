package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type subscriptionRepository struct {
	db *gorm.DB
}

// NewSubscriptionRepository создаёт реализацию SubscriptionRepository на основе GORM.
func NewSubscriptionRepository(db *gorm.DB) SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

func (r *subscriptionRepository) Create(ctx context.Context, sub *domain.Subscription) error {
	if err := r.db.WithContext(ctx).Create(sub).Error; err != nil {
		return fmt.Errorf("repository: subscription create: %w", err)
	}
	return nil
}

func (r *subscriptionRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	var sub domain.Subscription
	err := r.db.WithContext(ctx).First(&sub, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: subscription get by user id: %w", err)
	}
	return &sub, nil
}

func (r *subscriptionRepository) ListExpiredBefore(ctx context.Context, t time.Time) ([]*domain.Subscription, error) {
	var subs []*domain.Subscription
	if err := r.db.WithContext(ctx).
		Where("expires_at < ?", t).
		Find(&subs).Error; err != nil {
		return nil, fmt.Errorf("repository: subscription list expired: %w", err)
	}
	return subs, nil
}

func (r *subscriptionRepository) ListExpiringBetween(ctx context.Context, from, to time.Time) ([]*domain.Subscription, error) {
	var subs []*domain.Subscription
	if err := r.db.WithContext(ctx).
		Where("expires_at >= ? AND expires_at < ?", from, to).
		Find(&subs).Error; err != nil {
		return nil, fmt.Errorf("repository: subscription list expiring between: %w", err)
	}
	return subs, nil
}

func (r *subscriptionRepository) CountActive(ctx context.Context, now time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&domain.Subscription{}).
		Where("expires_at > ?", now).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("repository: subscription count active: %w", err)
	}
	return count, nil
}

func (r *subscriptionRepository) CountActiveByTier(ctx context.Context, tier domain.SubscriptionTier, now time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&domain.Subscription{}).
		Where("tier = ? AND expires_at > ?", tier, now).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("repository: subscription count active by tier: %w", err)
	}
	return count, nil
}

func (r *subscriptionRepository) Update(ctx context.Context, sub *domain.Subscription) error {
	if err := r.db.WithContext(ctx).Save(sub).Error; err != nil {
		return fmt.Errorf("repository: subscription update: %w", err)
	}
	return nil
}

func (r *subscriptionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&domain.Subscription{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("repository: subscription delete: %w", err)
	}
	return nil
}
