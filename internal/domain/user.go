package domain

import (
	"time"

	"github.com/google/uuid"
)

// User — зарегистрированный пользователь FreeWay VPN.
type User struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	TelegramID int64     `gorm:"uniqueIndex;not null"`
	Username   string    `gorm:"size:64"`
	SubToken   string    `gorm:"uniqueIndex;not null"` // токен для /sub/{token}
	DeviceLimit int      `gorm:"not null;default:1"`    // лимит устройств по тарифу
	CreatedAt  time.Time
	UpdatedAt  time.Time

	Subscription *Subscription `gorm:"foreignKey:UserID"`
}

// NewUser создаёт нового пользователя с уникальным UUID и sub-токеном.
func NewUser(telegramID int64, username string) *User {
	return &User{
		ID:         uuid.New(),
		TelegramID: telegramID,
		Username:   username,
		SubToken:   uuid.New().String(),
	}
}
