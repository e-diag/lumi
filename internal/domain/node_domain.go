package domain

import (
	"time"

	"github.com/google/uuid"
)

// NodeDomain — доменное имя для подключения к ноде (ротация, обход блокировок по SNI/Host).
type NodeDomain struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	NodeID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Domain    string    `gorm:"size:253;not null"`
	IsActive  bool      `gorm:"not null;default:true"`
	IsBlocked bool      `gorm:"not null;default:false"` // помечается воркером при деградации
	// LastCheckedAt — последняя проверка доступности.
	LastCheckedAt *time.Time
	// LastSuccessAt — последний успешный TCP+TLS probe.
	LastSuccessAt *time.Time
	// ConsecutiveFails — подряд неудачных проверок (сбрасывается при успехе).
	ConsecutiveFails int `gorm:"not null;default:0"`
	// Weight — вес для взвешенного случайного выбора (≥1).
	Weight int `gorm:"not null;default:1"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
