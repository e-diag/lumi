package database

import (
	"fmt"
	"log/slog"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// seedCatalog — product_settings (id=1), тарифы plans, при пустом каталоге серверов — не трогаем.
func seedCatalog(db *gorm.DB) error {
	settings := &domain.ProductSettings{ID: 1, TrialDays: 3, ReferralBonusDays: 3}
	if err := db.Where("id = ?", 1).FirstOrCreate(settings).Error; err != nil {
		return fmt.Errorf("seed product_settings: %w", err)
	}

	var planCount int64
	if err := db.Model(&domain.Plan{}).Count(&planCount).Error; err != nil {
		return fmt.Errorf("seed plans count: %w", err)
	}
	if planCount > 0 {
		return nil
	}

	type spec struct {
		code  string
		name  string
		tier  domain.SubscriptionTier
		days  int
		order int
	}
	specs := []spec{
		{"basic_30", "Базовый — 30 дней", domain.TierBasic, 30, 10},
		{"basic_90", "Базовый — 90 дней", domain.TierBasic, 90, 11},
		{"basic_365", "Базовый — 12 мес.", domain.TierBasic, 365, 12},
		{"premium_30", "Премиум — 30 дней", domain.TierPremium, 30, 20},
		{"premium_90", "Премиум — 90 дней", domain.TierPremium, 90, 21},
		{"premium_365", "Премиум — 12 мес.", domain.TierPremium, 365, 22},
	}
	for _, sp := range specs {
		kopeks, err := domain.SubscriptionPriceKopeks(sp.tier, sp.days)
		if err != nil {
			return fmt.Errorf("seed plan %s: %w", sp.code, err)
		}
		p := &domain.Plan{
			ID:           uuid.New(),
			Code:         sp.code,
			Name:         sp.name,
			Tier:         sp.tier,
			DurationDays: sp.days,
			PriceKopeks:  kopeks,
			Active:       true,
			SortOrder:    sp.order,
		}
		if err := db.Create(p).Error; err != nil {
			return fmt.Errorf("seed plan create %s: %w", sp.code, err)
		}
	}
	slog.Info("plans seeded", "count", len(specs))
	return nil
}
