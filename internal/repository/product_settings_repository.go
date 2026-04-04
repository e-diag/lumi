package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/freeway-vpn/backend/internal/domain"
	"gorm.io/gorm"
)

type productSettingsRepository struct{ db *gorm.DB }

func NewProductSettingsRepository(db *gorm.DB) ProductSettingsRepository {
	return &productSettingsRepository{db: db}
}

func (r *productSettingsRepository) Get(ctx context.Context) (*domain.ProductSettings, error) {
	var s domain.ProductSettings
	err := r.db.WithContext(ctx).First(&s, "id = ?", 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("repository: product settings missing: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("repository: product settings get: %w", err)
	}
	return &s, nil
}

func (r *productSettingsRepository) Upsert(ctx context.Context, s *domain.ProductSettings) error {
	s.ID = 1
	return r.db.WithContext(ctx).Save(s).Error
}
