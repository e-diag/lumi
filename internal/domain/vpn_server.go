package domain

import (
	"time"

	"github.com/google/uuid"
)

// VPNServer — запись сервера/панели для учёта в админке (основной провижининг пока из env XUI_*).
type VPNServer struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name          string    `gorm:"size:128;not null"`
	Region        string    `gorm:"size:64"`
	XUIBaseURL    string    `gorm:"size:512"` // без секрета; для справки и будущего multi-panel
	InboundID     int       `gorm:"not null;default:0"`
	Active        bool      `gorm:"not null;default:true"`
	Notes         string    `gorm:"size:1024"`
	LastCheckedAt *time.Time
	Healthy       bool `gorm:"not null;default:false"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
