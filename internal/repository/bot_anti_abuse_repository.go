package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type botAntiAbuseRepository struct {
	db *gorm.DB
}

// NewBotAntiAbuseRepository создаёт репозиторий антиабьюза бота.
func NewBotAntiAbuseRepository(db *gorm.DB) BotAntiAbuseRepository {
	return &botAntiAbuseRepository{db: db}
}

func (r *botAntiAbuseRepository) CountTrialGrantsByIP(ctx context.Context, ip string) (int64, error) {
	if ip == "" {
		return 0, nil
	}
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.BotTrialSignup{}).
		Where("ip = ? AND trial_granted = ?", ip, true).
		Count(&n).Error
	if err != nil {
		return 0, fmt.Errorf("repository: count trial by ip: %w", err)
	}
	return n, nil
}

func (r *botAntiAbuseRepository) InsertTrialSignup(ctx context.Context, row *domain.BotTrialSignup) error {
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return fmt.Errorf("repository: insert trial signup: %w", err)
	}
	return nil
}

func (r *botAntiAbuseRepository) GetFirstTrialSignupIPForUser(ctx context.Context, userID uuid.UUID) (string, error) {
	var row domain.BotTrialSignup
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND ip <> ''", userID).
		Order("created_at ASC").
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", fmt.Errorf("repository: first trial ip: %w", err)
	}
	return row.IP, nil
}

func (r *botAntiAbuseRepository) CountReferralGrantsSince(ctx context.Context, inviterID uuid.UUID, since time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.ReferralBonusLog{}).
		Where("inviter_id = ? AND created_at >= ?", inviterID, since).
		Count(&n).Error
	if err != nil {
		return 0, fmt.Errorf("repository: count referral grants: %w", err)
	}
	return n, nil
}

func (r *botAntiAbuseRepository) InsertReferralGrant(ctx context.Context, row *domain.ReferralBonusLog) error {
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return fmt.Errorf("repository: insert referral grant: %w", err)
	}
	return nil
}
