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
func (m *mockSubRepo) CountExpired(ctx context.Context, now time.Time) (int64, error) {
	args := m.Called(ctx, now)
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

type mockVPNPanel struct{ mock.Mock }

func (m *mockVPNPanel) SyncUserAccess(ctx context.Context, user *domain.User, tier domain.SubscriptionTier, expiresAt *time.Time) (*usecase.PanelSyncResult, error) {
	args := m.Called(ctx, user, tier, expiresAt)
	if v := args.Get(0); v != nil {
		return v.(*usecase.PanelSyncResult), args.Error(1)
	}
	return nil, args.Error(1)
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
	panel := &mockVPNPanel{}

	subRepo.On("GetByUserID", mock.Anything, userID).Return(existing, nil).Once()
	subRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Subscription")).Return(nil).Once()

	userRepo.On("GetByID", mock.Anything, userID).Return(user, nil).Once()
	userRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()
	panel.On("SyncUserAccess", mock.Anything, mock.AnythingOfType("*domain.User"), mock.Anything, mock.Anything).Return(&usecase.PanelSyncResult{}, nil).Once()

	uc := usecase.NewSubscriptionUseCase(subRepo, userRepo, panel)
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
	panel := &mockVPNPanel{}

	subRepo.On("ListExpiredBefore", mock.Anything, mock.Anything).Return([]*domain.Subscription{expired}, nil).Once()
	subRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Subscription")).Return(nil).Once()

	userRepo.On("GetByID", mock.Anything, userID).Return(user, nil).Once()
	userRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()
	panel.On("SyncUserAccess", mock.Anything, mock.AnythingOfType("*domain.User"), mock.Anything, mock.Anything).Return(&usecase.PanelSyncResult{}, nil).Once()

	uc := usecase.NewSubscriptionUseCase(subRepo, userRepo, panel)
	require.NoError(t, uc.ExpireOld(ctx))
}

func TestSubscriptionUseCase_AddBonusDays_NoSubscription_CreatesBasic(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	userID := uuid.New()

	subRepo := &mockSubRepo{}
	userRepo := &mockUserRepo{}
	panel := &mockVPNPanel{}

	subRepo.On("GetByUserID", mock.Anything, userID).Return(nil, domain.ErrSubscriptionNotFound).Twice()
	subRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Subscription")).Return(nil).Once()
	userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID, DeviceLimit: 1}, nil).Once()
	userRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()
	panel.On("SyncUserAccess", mock.Anything, mock.AnythingOfType("*domain.User"), mock.Anything, mock.Anything).Return(&usecase.PanelSyncResult{}, nil).Once()

	uc := usecase.NewSubscriptionUseCase(subRepo, userRepo, panel)
	require.NoError(t, uc.AddBonusDays(ctx, userID, 3))
}

func TestSubscriptionUseCase_AddBonusDays_ExtendsFromMaxNowOrExpiry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	userID := uuid.New()
	future := time.Now().Add(48 * time.Hour)
	sub := &domain.Subscription{
		ID:        uuid.New(),
		UserID:    userID,
		Tier:      domain.TierPremium,
		ExpiresAt: future,
	}

	subRepo := &mockSubRepo{}
	userRepo := &mockUserRepo{}
	panel := &mockVPNPanel{}

	subRepo.On("GetByUserID", mock.Anything, userID).Return(sub, nil).Once()
	subRepo.On("Update", mock.Anything, mock.MatchedBy(func(s *domain.Subscription) bool {
		return s.UserID == userID && s.ExpiresAt.After(future)
	})).Return(nil).Once()
	userRepo.On("GetByID", mock.Anything, userID).Return(&domain.User{ID: userID}, nil).Once()
	userRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()
	panel.On("SyncUserAccess", mock.Anything, mock.AnythingOfType("*domain.User"), mock.Anything, mock.Anything).Return(&usecase.PanelSyncResult{}, nil).Once()

	uc := usecase.NewSubscriptionUseCase(subRepo, userRepo, panel)
	require.NoError(t, uc.AddBonusDays(ctx, userID, 5))
}
