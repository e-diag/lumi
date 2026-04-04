package domain

import "time"

// ProductSettings — одна строка (id=1): политика триала, реферала, текст поддержки для бота/панели.
type ProductSettings struct {
	ID                uint `gorm:"primaryKey"`
	TrialDays         int  `gorm:"not null;default:3"`
	ReferralBonusDays int  `gorm:"not null;default:3"`
	SupportURL        string `gorm:"size:512"`
	UpdatedAt         time.Time
}
