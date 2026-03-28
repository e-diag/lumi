package domain

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionTier — тариф подписки.
type SubscriptionTier string

const (
	TierFree    SubscriptionTier = "free"
	TierBasic   SubscriptionTier = "basic"
	TierPremium SubscriptionTier = "premium"
)

// TierLimits описывает ограничения тарифа.
type TierLimits struct {
	SpeedMbps  int            // 0 = без ограничений
	Devices    int            // максимум устройств
	Regions    []NodeRegion   // доступные регионы
}

// TierLimitsMap — лимиты для каждого тарифа.
var TierLimitsMap = map[SubscriptionTier]TierLimits{
	TierFree: {
		SpeedMbps: 1,
		Devices:   1,
		Regions:   []NodeRegion{RegionEU},
	},
	TierBasic: {
		SpeedMbps: 10,
		Devices:   2,
		Regions:   []NodeRegion{RegionEU, RegionUSA},
	},
	TierPremium: {
		SpeedMbps: 0,
		Devices:   5,
		Regions:   []NodeRegion{RegionEU, RegionUSA, RegionCDN},
	},
}

// Subscription — активная подписка пользователя.
type Subscription struct {
	ID        uuid.UUID        `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID        `gorm:"type:uuid;not null;index"`
	Tier      SubscriptionTier `gorm:"not null"`
	ExpiresAt time.Time        `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsActive возвращает true, если подписка ещё не истекла.
func (s *Subscription) IsActive() bool {
	return time.Now().Before(s.ExpiresAt)
}

// DaysLeft возвращает количество оставшихся дней подписки.
func (s *Subscription) DaysLeft() int {
	if !s.IsActive() {
		return 0
	}
	return int(time.Until(s.ExpiresAt).Hours() / 24)
}

// NewSubscription создаёт новую подписку на указанное количество дней.
func NewSubscription(userID uuid.UUID, tier SubscriptionTier, days int) *Subscription {
	return &Subscription{
		ID:        uuid.New(),
		UserID:    userID,
		Tier:      tier,
		ExpiresAt: time.Now().AddDate(0, 0, days),
	}
}
