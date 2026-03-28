package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/repository"
	"github.com/google/uuid"
)

type userUseCase struct {
	userRepo repository.UserRepository
}

// NewUserUseCase создаёт реализацию UserUseCase.
func NewUserUseCase(userRepo repository.UserRepository) UserUseCase {
	return &userUseCase{userRepo: userRepo}
}

// Register регистрирует нового пользователя или возвращает существующего.
func (uc *userUseCase) Register(ctx context.Context, telegramID int64, username string) (*domain.User, error) {
	existing, err := uc.userRepo.GetByTelegramID(ctx, telegramID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("usecase: register user: %w", err)
	}

	user := domain.NewUser(telegramID, username)
	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("usecase: register user: %w", err)
	}
	return user, nil
}

// GetByTelegramID возвращает пользователя по Telegram ID.
func (uc *userUseCase) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	user, err := uc.userRepo.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("usecase: get user by telegram id: %w", err)
	}
	return user, nil
}

// GetByID возвращает пользователя по UUID.
func (uc *userUseCase) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := uc.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("usecase: get user by id: %w", err)
	}
	return user, nil
}

func (uc *userUseCase) List(ctx context.Context, query string, page, pageSize int) ([]*domain.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize
	users, total, err := uc.userRepo.List(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase: list users: %w", err)
	}
	return users, total, nil
}
