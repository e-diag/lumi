package domain

import (
	"time"

	"github.com/google/uuid"
)

// BotTrialSignup — аудит выдачи триала из бота (антиспам по IP / UA).
type BotTrialSignup struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	TelegramID   int64     `gorm:"not null;index"`
	UserID       *uuid.UUID `gorm:"type:uuid;index"`
	IP           string    `gorm:"size:128"`
	UserAgent    string    `gorm:"size:512"`
	TrialGranted bool      `gorm:"not null;default:false"`
	CreatedAt    time.Time
}

// ReferralBonusLog — начисление +3 дня пригласившему (лимит в месяц).
type ReferralBonusLog struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	InviterID  uuid.UUID `gorm:"type:uuid;not null;index"`
	GranteeID  uuid.UUID `gorm:"type:uuid;not null"`
	CreatedAt  time.Time
}

// UserAccessProbe — мягкий учёт обращений к подписке (последние записи по пользователю).
type UserAccessProbe struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index"`
	IP         string    `gorm:"size:128"`
	UserAgent  string    `gorm:"size:256"`
	CreatedAt  time.Time
}
