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

type paymentRepository struct {
	db *gorm.DB
}

// NewPaymentRepository создаёт реализацию PaymentRepository на основе GORM.
func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	if err := r.db.WithContext(ctx).Create(payment).Error; err != nil {
		return fmt.Errorf("repository: payment create: %w", err)
	}
	return nil
}

func (r *paymentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	var payment domain.Payment
	err := r.db.WithContext(ctx).First(&payment, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrPaymentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: payment get by id: %w", err)
	}
	return &payment, nil
}

func (r *paymentRepository) GetByYookassaID(ctx context.Context, yookassaID string) (*domain.Payment, error) {
	var payment domain.Payment
	err := r.db.WithContext(ctx).First(&payment, "yookassa_id = ?", yookassaID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrPaymentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: payment get by yookassa id: %w", err)
	}
	return &payment, nil
}

func (r *paymentRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Payment, error) {
	var payments []*domain.Payment
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&payments).Error; err != nil {
		return nil, fmt.Errorf("repository: payment get by user id: %w", err)
	}
	return payments, nil
}

func (r *paymentRepository) ListPendingOlderThan(ctx context.Context, t time.Time) ([]*domain.Payment, error) {
	var payments []*domain.Payment
	if err := r.db.WithContext(ctx).
		Where("status = ? AND created_at < ?", domain.PaymentPending, t).
		Order("created_at ASC").
		Find(&payments).Error; err != nil {
		return nil, fmt.Errorf("repository: payment list pending older than: %w", err)
	}
	return payments, nil
}

func (r *paymentRepository) ListByFilter(ctx context.Context, status string, from, to *time.Time, limit, offset int) ([]*domain.Payment, int64, error) {
	dbq := r.db.WithContext(ctx).Model(&domain.Payment{})
	if status != "" {
		dbq = dbq.Where("status = ?", status)
	}
	if from != nil {
		dbq = dbq.Where("created_at >= ?", *from)
	}
	if to != nil {
		dbq = dbq.Where("created_at < ?", *to)
	}

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("repository: payment list by filter count: %w", err)
	}

	var payments []*domain.Payment
	if err := dbq.Order("created_at DESC").Limit(limit).Offset(offset).Find(&payments).Error; err != nil {
		return nil, 0, fmt.Errorf("repository: payment list by filter: %w", err)
	}
	return payments, total, nil
}

func (r *paymentRepository) SumSucceededBetween(ctx context.Context, from, to time.Time) (int64, error) {
	type row struct {
		Sum int64
	}
	var result row
	if err := r.db.WithContext(ctx).Model(&domain.Payment{}).
		Select("COALESCE(SUM(amount_rub), 0) AS sum").
		Where("status = ? AND created_at >= ? AND created_at < ?", domain.PaymentSucceeded, from, to).
		Scan(&result).Error; err != nil {
		return 0, fmt.Errorf("repository: payment sum succeeded between: %w", err)
	}
	return result.Sum, nil
}

func (r *paymentRepository) CountSucceededBetween(ctx context.Context, from, to time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.Payment{}).
		Where("status = ? AND created_at >= ? AND created_at < ?", domain.PaymentSucceeded, from, to).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("repository: payment count succeeded between: %w", err)
	}
	return count, nil
}

func (r *paymentRepository) Update(ctx context.Context, payment *domain.Payment) error {
	if err := r.db.WithContext(ctx).Save(payment).Error; err != nil {
		return fmt.Errorf("repository: payment update: %w", err)
	}
	return nil
}
