package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository создаёт реализацию UserRepository на основе GORM.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("repository: user create: %w", err)
	}
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Preload("Subscription").First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: user get by id: %w", err)
	}
	return &user, nil
}

func (r *userRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Preload("Subscription").First(&user, "telegram_id = ?", telegramID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: user get by telegram id: %w", err)
	}
	return &user, nil
}

func (r *userRepository) GetBySubToken(ctx context.Context, token string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Preload("Subscription").First(&user, "sub_token = ?", token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: user get by sub token: %w", err)
	}
	return &user, nil
}

func (r *userRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.User{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("repository: user count: %w", err)
	}
	return count, nil
}

func (r *userRepository) List(ctx context.Context, query string, limit, offset int) ([]*domain.User, int64, error) {
	dbq := r.db.WithContext(ctx).Model(&domain.User{})
	if query != "" {
		like := "%" + query + "%"
		dbq = dbq.Where("CAST(telegram_id AS TEXT) ILIKE ? OR username ILIKE ?", like, like)
	}

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("repository: user list count: %w", err)
	}

	var users []*domain.User
	if err := dbq.Preload("Subscription").Order("created_at DESC").Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("repository: user list: %w", err)
	}
	return users, total, nil
}

func (r *userRepository) CountCreatedBetween(ctx context.Context, from, to time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.User{}).
		Where("created_at >= ? AND created_at < ?", from, to).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("repository: user count created between: %w", err)
	}
	return count, nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("repository: user update: %w", err)
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&domain.User{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("repository: user delete: %w", err)
	}
	return nil
}
