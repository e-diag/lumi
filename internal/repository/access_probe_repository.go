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
	// Один запрос вместо цикла count/delete — меньше нагрузки на БД и нет риска долгого цикла при аномалиях.
	if err := r.db.WithContext(ctx).Exec(`
WITH counts AS (
	SELECT COUNT(*)::bigint AS c FROM user_access_probes WHERE user_id = ?
),
to_delete AS (
	SELECT id FROM user_access_probes
	WHERE user_id = ?
	ORDER BY created_at ASC
	LIMIT GREATEST(0, (SELECT c FROM counts) - ?)
)
DELETE FROM user_access_probes WHERE id IN (SELECT id FROM to_delete)
`, userID, userID, keep).Error; err != nil {
		return fmt.Errorf("repository: access probe trim: %w", err)
	}
	return nil
}
