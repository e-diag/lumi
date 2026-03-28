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

type mockPaymentRepo struct{ mock.Mock }

func (m *mockPaymentRepo) Create(ctx context.Context, p *domain.Payment) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}
func (m *mockPaymentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.Payment), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockPaymentRepo) GetByYookassaID(ctx context.Context, yid string) (*domain.Payment, error) {
	args := m.Called(ctx, yid)
	if v := args.Get(0); v != nil {
		return v.(*domain.Payment), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockPaymentRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Payment, error) {
	args := m.Called(ctx, userID)
	if v := args.Get(0); v != nil {
		return v.([]*domain.Payment), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockPaymentRepo) ListPendingOlderThan(ctx context.Context, t time.Time) ([]*domain.Payment, error) {
	args := m.Called(ctx, t)
	if v := args.Get(0); v != nil {
		return v.([]*domain.Payment), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockPaymentRepo) ListByFilter(ctx context.Context, status string, from, to *time.Time, limit, offset int) ([]*domain.Payment, int64, error) {
	args := m.Called(ctx, status, from, to, limit, offset)
	if v := args.Get(0); v != nil {
		return v.([]*domain.Payment), args.Get(1).(int64), args.Error(2)
	}
	return nil, 0, args.Error(2)
}
func (m *mockPaymentRepo) SumSucceededBetween(ctx context.Context, from, to time.Time) (int64, error) {
	args := m.Called(ctx, from, to)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockPaymentRepo) CountSucceededBetween(ctx context.Context, from, to time.Time) (int64, error) {
	args := m.Called(ctx, from, to)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockPaymentRepo) Update(ctx context.Context, p *domain.Payment) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

type mockSubUC struct{ mock.Mock }

func (m *mockSubUC) GetUserSubscription(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	args := m.Called(ctx, userID)
	if v := args.Get(0); v != nil {
		return v.(*domain.Subscription), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockSubUC) ActivateSubscription(ctx context.Context, userID uuid.UUID, tier domain.SubscriptionTier, days int) (*domain.Subscription, error) {
	args := m.Called(ctx, userID, tier, days)
	if v := args.Get(0); v != nil {
		return v.(*domain.Subscription), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockSubUC) ExtendSubscription(ctx context.Context, userID uuid.UUID, days int) (*domain.Subscription, error) {
	args := m.Called(ctx, userID, days)
	if v := args.Get(0); v != nil {
		return v.(*domain.Subscription), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockSubUC) ExpireOld(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
func (m *mockSubUC) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	args := m.Called(ctx, userID)
	if v := args.Get(0); v != nil {
		return v.(*domain.Subscription), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockSubUC) GetExpiringIn3Days(ctx context.Context) ([]*domain.Subscription, error) {
	args := m.Called(ctx)
	if v := args.Get(0); v != nil {
		return v.([]*domain.Subscription), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockSubUC) DeactivateSubscription(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

type mockGateway struct{ mock.Mock }

func (m *mockGateway) CreatePayment(ctx context.Context, req usecase.PaymentGatewayCreateRequest) (*usecase.PaymentGatewayPayment, error) {
	args := m.Called(ctx, req)
	if v := args.Get(0); v != nil {
		return v.(*usecase.PaymentGatewayPayment), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockGateway) GetPayment(ctx context.Context, id string) (*usecase.PaymentGatewayPayment, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*usecase.PaymentGatewayPayment), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestPaymentUseCase_CreatePayment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tier    domain.SubscriptionTier
		days    int
		wantErr bool
	}{
		{name: "basic_30", tier: domain.TierBasic, days: 30, wantErr: false},
		{name: "premium_7", tier: domain.TierPremium, days: 7, wantErr: false},
		{name: "free_rejected", tier: domain.TierFree, days: 30, wantErr: true},
		{name: "days_invalid", tier: domain.TierBasic, days: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			userID := uuid.New()

			pRepo := &mockPaymentRepo{}
			subUC := &mockSubUC{}
			gw := &mockGateway{}

			if !tt.wantErr {
				gw.On("CreatePayment", mock.Anything, mock.AnythingOfType("usecase.PaymentGatewayCreateRequest")).
					Return(&usecase.PaymentGatewayPayment{
						ID:              "prov-1",
						Status:          "pending",
						ConfirmationURL: "https://pay.example/1",
					}, nil).Once()

				pRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(nil).Once()
			}

			uc := usecase.NewPaymentUseCase(pRepo, subUC, gw, "https://example.com")
			p, url, err := uc.CreatePayment(ctx, userID, tt.tier, tt.days)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, p)
			assert.NotEmpty(t, url)
			assert.Equal(t, userID, p.UserID)
			assert.Equal(t, domain.PaymentPending, p.Status)
		})
	}
}

func TestPaymentUseCase_HandleWebhook_Succeeded_ActivatesSubscription(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	p := &domain.Payment{
		ID:           uuid.New(),
		UserID:       userID,
		YookassaID:   "prov-1",
		Tier:         domain.TierPremium,
		DurationDays: 30,
		Status:       domain.PaymentPending,
	}

	pRepo := &mockPaymentRepo{}
	subUC := &mockSubUC{}
	gw := &mockGateway{}

	pRepo.On("GetByYookassaID", mock.Anything, "prov-1").Return(p, nil).Once()
	pRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(nil).Once()
	subUC.On("ActivateSubscription", mock.Anything, userID, domain.TierPremium, 30).
		Return(&domain.Subscription{ID: uuid.New(), UserID: userID, Tier: domain.TierPremium}, nil).Once()

	uc := usecase.NewPaymentUseCase(pRepo, subUC, gw, "https://example.com")

	var ev usecase.WebhookEvent
	ev.Object.ID = "prov-1"
	ev.Object.Status = "succeeded"
	require.NoError(t, uc.HandleWebhook(ctx, ev))
}

