package domain

import (
	"time"

	"github.com/google/uuid"
)

// MigrationStatus — статус записи миграции из старого MTProto-бота.
type MigrationStatus string

const (
	MigrationPending   MigrationStatus = "pending"
	MigrationCompleted MigrationStatus = "completed"
	MigrationFailed    MigrationStatus = "failed"
)

// MigrationRecord — запись о миграции пользователя из старого бота (Фаза 6).
// Заглушка — будет заполнена при реализации cmd/migrator.
type MigrationRecord struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	OldTelegramID int64     `gorm:"not null;index"`
	NewUserID     uuid.UUID `gorm:"type:uuid;not null;index"`
	OldTier       string    `gorm:"size:32"`
	NewTier       SubscriptionTier
	DaysMigrated  int
	Status        MigrationStatus `gorm:"not null;default:'pending'"`
	ErrorMessage  string          `gorm:"size:512"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
