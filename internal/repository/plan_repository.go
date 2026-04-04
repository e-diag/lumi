package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type planRepository struct{ db *gorm.DB }

func NewPlanRepository(db *gorm.DB) PlanRepository {
	return &planRepository{db: db}
}

func (r *planRepository) Create(ctx context.Context, p *domain.Plan) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *planRepository) GetByCode(ctx context.Context, code string) (*domain.Plan, error) {
	var p domain.Plan
	err := r.db.WithContext(ctx).First(&p, "code = ?", code).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrPlanNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: plan get by code: %w", err)
	}
	return &p, nil
}

func (r *planRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	var p domain.Plan
	err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrPlanNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: plan get: %w", err)
	}
	return &p, nil
}

func (r *planRepository) ListActive(ctx context.Context) ([]*domain.Plan, error) {
	var list []*domain.Plan
	if err := r.db.WithContext(ctx).Where("active = ?", true).Order("sort_order ASC, tier ASC, duration_days ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("repository: plan list active: %w", err)
	}
	return list, nil
}

func (r *planRepository) ListAll(ctx context.Context) ([]*domain.Plan, error) {
	var list []*domain.Plan
	if err := r.db.WithContext(ctx).Order("sort_order ASC, tier ASC, duration_days ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("repository: plan list all: %w", err)
	}
	return list, nil
}

func (r *planRepository) Update(ctx context.Context, p *domain.Plan) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *planRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Plan{}, "id = ?", id).Error
}
