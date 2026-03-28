package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type accessProbeRepository struct {
	db *gorm.DB
}

// NewAccessProbeRepository создаёт репозиторий проб доступа к подписке.
func NewAccessProbeRepository(db *gorm.DB) AccessProbeRepository {
	return &accessProbeRepository{db: db}
}

func (r *accessProbeRepository) Append(ctx context.Context, userID uuid.UUID, ip, userAgent string, keep int) error {
	if keep <= 0 {
		keep = 3
	}
	ua := userAgent
	if len(ua) > 256 {
		ua = ua[:256]
	}
	row := &domain.UserAccessProbe{
		ID:        uuid.New(),
		UserID:    userID,
		IP:        ip,
		UserAgent: ua,
		CreatedAt: time.Now(),
	}
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return fmt.Errorf("repository: access probe append: %w", err)
	}
	for {
		var n int64
		if err := r.db.WithContext(ctx).Model(&domain.UserAccessProbe{}).
			Where("user_id = ?", userID).Count(&n).Error; err != nil {
			return fmt.Errorf("repository: access probe count: %w", err)
		}
		if n <= int64(keep) {
			return nil
		}
		var oldest domain.UserAccessProbe
		if err := r.db.WithContext(ctx).
			Where("user_id = ?", userID).
			Order("created_at ASC").
			First(&oldest).Error; err != nil {
			return fmt.Errorf("repository: access probe oldest: %w", err)
		}
		if err := r.db.WithContext(ctx).Delete(&oldest).Error; err != nil {
			return fmt.Errorf("repository: access probe delete oldest: %w", err)
		}
	}
}
