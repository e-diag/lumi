package usecase

import (
	"context"
	"fmt"

	"github.com/freeway-vpn/backend/internal/repository"
	"github.com/google/uuid"
)

type accessProbeUseCase struct {
	repo repository.AccessProbeRepository
}

// NewAccessProbeUseCase создаёт сценарий мягкого учёта обращений к подписке.
func NewAccessProbeUseCase(repo repository.AccessProbeRepository) AccessProbeUseCase {
	return &accessProbeUseCase{repo: repo}
}

// RecordSubscriptionFetch сохраняет последнее обращение к GET /sub (до 3 записей на пользователя).
func (uc *accessProbeUseCase) RecordSubscriptionFetch(ctx context.Context, userID uuid.UUID, ip, userAgent string) error {
	if err := uc.repo.Append(ctx, userID, ip, userAgent, 3); err != nil {
		return fmt.Errorf("usecase: access probe: %w", err)
	}
	return nil
}
