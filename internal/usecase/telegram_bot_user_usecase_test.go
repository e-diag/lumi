package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubTelegramSubUC минимальная реализация для проверки вызова Activate при триале и бонуса рефереру.
type stubTelegramSubUC struct {
	activateCalls int
	bonusCalls    int
}

func (s *stubTelegramSubUC) GetUserSubscription(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	return nil, domain.ErrSubscriptionNotFound
}
func (s *stubTelegramSubUC) ActivateSubscription(ctx context.Context, userID uuid.UUID, tier domain.SubscriptionTier, days int) (*domain.Subscription, error) {
	s.activateCalls++
	return domain.NewSubscription(userID, tier, days), nil
}
func (s *stubTelegramSubUC) ExtendSubscription(ctx context.Context, userID uuid.UUID, days int) (*domain.Subscription, error) {
	return nil, nil
}
func (s *stubTelegramSubUC) DeactivateSubscription(ctx context.Context, userID uuid.UUID) error { return nil }
func (s *stubTelegramSubUC) ExpireOld(ctx context.Context) error                              { return nil }
func (s *stubTelegramSubUC) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	return nil, nil
}
func (s *stubTelegramSubUC) GetExpiringIn3Days(ctx context.Context) ([]*domain.Subscription, error) {
	return nil, nil
}
func (s *stubTelegramSubUC) AddBonusDays(ctx context.Context, userID uuid.UUID, days int) error {
	s.bonusCalls++
	return nil
}

type memUserRepo struct {
	users []*domain.User
}

func (m *memUserRepo) Create(ctx context.Context, user *domain.User) error {
	m.users = append(m.users, user)
	return nil
}
func (m *memUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}
func (m *memUserRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	for _, u := range m.users {
		if u.TelegramID == telegramID {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}
func (m *memUserRepo) GetBySubToken(ctx context.Context, token string) (*domain.User, error) { return nil, domain.ErrUserNotFound }
func (m *memUserRepo) Count(ctx context.Context) (int64, error)                               { return int64(len(m.users)), nil }
func (m *memUserRepo) List(ctx context.Context, query string, limit, offset int) ([]*domain.User, int64, error) {
	return m.users, int64(len(m.users)), nil
}
func (m *memUserRepo) CountCreatedBetween(ctx context.Context, from, to time.Time) (int64, error) {
	return 0, nil
}
func (m *memUserRepo) Update(ctx context.Context, user *domain.User) error {
	for i, u := range m.users {
		if u.ID == user.ID {
			m.users[i] = user
			return nil
		}
	}
	return domain.ErrUserNotFound
}
func (m *memUserRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

func TestTelegramBotUser_OnStart_NewUser_GrantsTrial(t *testing.T) {
	t.Parallel()
	repo := &memUserRepo{}
	sub := &stubTelegramSubUC{}
	uc := usecase.NewTelegramBotUserUseCase(repo, sub, nil, 0, 0)

	u, out, err := uc.OnStart(context.Background(), 424242, "testuser", nil, usecase.TelegramClientMeta{})
	require.NoError(t, err)
	require.NotNil(t, u)
	require.True(t, out.IsNewUser)
	require.True(t, out.TrialGranted)
	assert.Equal(t, 1, sub.activateCalls)
	assert.Equal(t, 0, sub.bonusCalls)
	assert.True(t, u.WelcomeBonusUsed)
}

func TestTelegramBotUser_OnStart_Referral_GrantsBonusToInviter(t *testing.T) {
	t.Parallel()
	inviter := domain.NewUser(900001, "inviter")
	repo := &memUserRepo{users: []*domain.User{inviter}}
	sub := &stubTelegramSubUC{}
	uc := usecase.NewTelegramBotUserUseCase(repo, sub, nil, 0, 0)

	refID := inviter.ID
	u, _, err := uc.OnStart(context.Background(), 900002, "newbie", &refID, usecase.TelegramClientMeta{})
	require.NoError(t, err)
	require.NotNil(t, u)
	require.NotNil(t, u.ReferredBy)
	assert.Equal(t, inviter.ID, *u.ReferredBy)
	assert.Equal(t, 1, sub.activateCalls)
	assert.Equal(t, 1, sub.bonusCalls)
}
