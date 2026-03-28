package domain

import (
	"time"

	"github.com/google/uuid"
)

// PaymentActivation — факт однократной активации подписки по платежу (защита от гонок webhook / worker).
type PaymentActivation struct {
	PaymentID uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time
}
