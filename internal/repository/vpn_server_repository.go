package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type vpnServerRepository struct{ db *gorm.DB }

func NewVPNServerRepository(db *gorm.DB) VPNServerRepository {
	return &vpnServerRepository{db: db}
}

func (r *vpnServerRepository) Create(ctx context.Context, s *domain.VPNServer) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *vpnServerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.VPNServer, error) {
	var s domain.VPNServer
	err := r.db.WithContext(ctx).First(&s, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("vpn server not found: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("repository: vpn server get: %w", err)
	}
	return &s, nil
}

func (r *vpnServerRepository) Count(ctx context.Context) (int64, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&domain.VPNServer{}).Count(&n).Error; err != nil {
		return 0, fmt.Errorf("repository: vpn server count: %w", err)
	}
	return n, nil
}

func (r *vpnServerRepository) ListAll(ctx context.Context) ([]*domain.VPNServer, error) {
	var list []*domain.VPNServer
	if err := r.db.WithContext(ctx).Order("region ASC, name ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("repository: vpn server list: %w", err)
	}
	return list, nil
}

func (r *vpnServerRepository) Update(ctx context.Context, s *domain.VPNServer) error {
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *vpnServerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.VPNServer{}, "id = ?", id).Error
}
