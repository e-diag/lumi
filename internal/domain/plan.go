package domain

import (
	"time"

	"github.com/google/uuid"
)

// Plan — тариф для оплаты (источник цены и длительности для ЮKassa и бота).
type Plan struct {
	ID           uuid.UUID        `gorm:"type:uuid;primaryKey"`
	Code         string           `gorm:"uniqueIndex;size:64;not null"`
	Name         string           `gorm:"size:128;not null"`
	Tier         SubscriptionTier `gorm:"not null"`
	DurationDays int              `gorm:"not null"`
	PriceKopeks  int64            `gorm:"not null"` // полная сумма за период в копейках
	Active       bool             `gorm:"not null;default:true"`
	SortOrder    int              `gorm:"not null;default:0"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
