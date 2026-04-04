package domain

import (
	"time"

	"github.com/google/uuid"
)

// PaymentStatus — статус платежа.
type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "pending"
	PaymentSucceeded PaymentStatus = "succeeded"
	PaymentCanceled  PaymentStatus = "canceled"
)

// Payment — платёж через ЮKassa.
type Payment struct {
	ID              uuid.UUID        `gorm:"type:uuid;primaryKey"`
	UserID          uuid.UUID        `gorm:"type:uuid;not null;index"`
	PlanID          *uuid.UUID       `gorm:"type:uuid;index"`
	YookassaID      string           `gorm:"uniqueIndex;size:64"` // ID платежа в ЮKassa
	AmountRub       int              `gorm:"not null"`            // сумма в рублях
	Tier            SubscriptionTier `gorm:"not null"`
	DurationDays    int              `gorm:"not null"`
	Status          PaymentStatus    `gorm:"not null;default:'pending'"`
	ConfirmationURL string           `gorm:"size:512"` // ссылка для оплаты
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
