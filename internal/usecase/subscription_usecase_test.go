package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockSubRepo struct{ mock.Mock }

func (m *mockSubRepo) Create(ctx context.Context, sub *domain.Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}
func (m *mockSubRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	args := m.Called(ctx, userID)
	if v := args.Get(0); v != nil {
		return v.(*domain.Subscription), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockSubRepo) ListExpiredBefore(ctx context.Context, t time.Time) ([]*domain.Subscription, error) {
	args := m.Called(ctx, t)
	if v := args.Get(0); v != nil {
		return v.([]*domain.Subscription), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockSubRepo) ListExpiringBetween(ctx context.Context, from, to time.Time) ([]*domain.Subscription, error) {
	args := m.Called(ctx, from, to)
	if v := args.Get(0); v != nil {
		return v.([]*domain.Subscription), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockSubRepo) CountActive(ctx context.Context, now time.Time) (int64, error) {
	args := m.Called(ctx, now)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockSubRepo) CountActiveByTier(ctx context.Context, tier domain.SubscriptionTier, now time.Time) (int64, error) {
	args := m.Called(ctx, tier, now)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockSubRepo) Update(ctx context.Context, sub *domain.Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}
func (m *mockSubRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}
func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockUserRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	args := m.Called(ctx, telegramID)
	if v := args.Get(0); v != nil {
		return v.(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockUserRepo) GetBySubToken(ctx context.Context, token string) (*domain.User, error) {
	args := m.Called(ctx, token)
	if v := args.Get(0); v != nil {
		return v.(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockUserRepo) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockUserRepo) List(ctx context.Context, query string, limit, offset int) ([]*domain.User, int64, error) {
	args := m.Called(ctx, query, limit, offset)
	if v := args.Get(0); v != nil {
		return v.([]*domain.User), args.Get(1).(int64), args.Error(2)
	}
	return nil, 0, args.Error(2)
}
func (m *mockUserRepo) CountCreatedBetween(ctx context.Context, from, to time.Time) (int64, error) {
	args := m.Called(ctx, from, to)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockUserRepo) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}
func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockRemnawave struct{ mock.Mock }

func (m *mockRemnawave) CreateUser(ctx context.Context, userUUID, username string, tier domain.SubscriptionTier) error {
	args := m.Called(ctx, userUUID, username, tier)
	return args.Error(0)
}
func (m *mockRemnawave) DeleteUser(ctx context.Context, userUUID string) error {
	args := m.Called(ctx, userUUID)
	return args.Error(0)
}
func (m *mockRemnawave) UpdateUserExpiry(ctx context.Context, userUUID string, expiresAt *time.Time) error {
	args := m.Called(ctx, userUUID, expiresAt)
	return args.Error(0)
}

func TestSubscriptionUseCase_ActivateSubscription_ExtendsIfActive(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	user := &domain.User{ID: userID, TelegramID: 1, Username: "u1"}
	existing := &domain.Subscription{
		ID:        uuid.New(),
		UserID:    userID,
		Tier:      domain.TierBasic,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	subRepo := &mockSubRepo{}
	userRepo := &mockUserRepo{}
	rem := &mockRemnawave{}

	subRepo.On("GetByUserID", mock.Anything, userID).Return(existing, nil).Once()
	subRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Subscription")).Return(nil).Once()

	userRepo.On("GetByID", mock.Anything, userID).Return(user, nil).Once()
	userRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()
	rem.On("UpdateUserExpiry", mock.Anything, userID.String(), mock.Anything).Return(nil).Once()

	uc := usecase.NewSubscriptionUseCase(subRepo, userRepo, rem)
	got, err := uc.ActivateSubscription(ctx, userID, domain.TierPremium, 30)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, domain.TierPremium, got.Tier)
	assert.True(t, got.ExpiresAt.After(existing.ExpiresAt))
}

func TestSubscriptionUseCase_ExpireOld_DowngradesToFree(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Now()
	userID := uuid.New()
	user := &domain.User{ID: userID, TelegramID: 1, Username: "u1"}
	expired := &domain.Subscription{
		ID:        uuid.New(),
		UserID:    userID,
		Tier:      domain.TierPremium,
		ExpiresAt: now.Add(-time.Hour),
	}

	subRepo := &mockSubRepo{}
	userRepo := &mockUserRepo{}
	rem := &mockRemnawave{}

	subRepo.On("ListExpiredBefore", mock.Anything, mock.Anything).Return([]*domain.Subscription{expired}, nil).Once()
	subRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Subscription")).Return(nil).Once()

	userRepo.On("GetByID", mock.Anything, userID).Return(user, nil).Once()
	userRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()
	rem.On("UpdateUserExpiry", mock.Anything, userID.String(), mock.Anything).Return(nil).Once()

	uc := usecase.NewSubscriptionUseCase(subRepo, userRepo, rem)
	require.NoError(t, uc.ExpireOld(ctx))
}

