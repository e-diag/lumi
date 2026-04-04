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
func (m *mockPaymentRepo) ClaimSucceededByYookassaID(ctx context.Context, yid string) (*domain.Payment, bool, error) {
	args := m.Called(ctx, yid)
	var p *domain.Payment
	if v := args.Get(0); v != nil {
		p = v.(*domain.Payment)
	}
	return p, args.Bool(1), args.Error(2)
}
func (m *mockPaymentRepo) ClaimCanceledByYookassaID(ctx context.Context, yid string) (*domain.Payment, bool, error) {
	args := m.Called(ctx, yid)
	var p *domain.Payment
	if v := args.Get(0); v != nil {
		p = v.(*domain.Payment)
	}
	return p, args.Bool(1), args.Error(2)
}
func (m *mockPaymentRepo) ClaimSucceededByID(ctx context.Context, id uuid.UUID) (*domain.Payment, bool, error) {
	args := m.Called(ctx, id)
	var p *domain.Payment
	if v := args.Get(0); v != nil {
		p = v.(*domain.Payment)
	}
	return p, args.Bool(1), args.Error(2)
}
func (m *mockPaymentRepo) ClaimCanceledByID(ctx context.Context, id uuid.UUID) (*domain.Payment, bool, error) {
	args := m.Called(ctx, id)
	var p *domain.Payment
	if v := args.Get(0); v != nil {
		p = v.(*domain.Payment)
	}
	return p, args.Bool(1), args.Error(2)
}
func (m *mockPaymentRepo) ConsumePaymentActivation(ctx context.Context, paymentID uuid.UUID) (bool, error) {
	args := m.Called(ctx, paymentID)
	return args.Bool(0), args.Error(1)
}
func (m *mockPaymentRepo) ReleasePaymentActivation(ctx context.Context, paymentID uuid.UUID) error {
	args := m.Called(ctx, paymentID)
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
func (m *mockSubUC) AddBonusDays(ctx context.Context, userID uuid.UUID, days int) error {
	args := m.Called(ctx, userID, days)
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

			uc := usecase.NewPaymentUseCase(pRepo, nil, subUC, gw, "https://example.com", nil)
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

func TestPaymentUseCase_CreatePayment_NoGateway(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc := usecase.NewPaymentUseCase(&mockPaymentRepo{}, nil, &mockSubUC{}, nil, "https://example.com", nil)
	_, _, err := uc.CreatePayment(ctx, uuid.New(), domain.TierBasic, 30)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gateway not configured")
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

	pSucceeded := *p
	pSucceeded.Status = domain.PaymentSucceeded
	pRepo.On("ClaimSucceededByYookassaID", mock.Anything, "prov-1").Return(&pSucceeded, true, nil).Once()
	pRepo.On("ConsumePaymentActivation", mock.Anything, p.ID).Return(true, nil).Once()
	subUC.On("ActivateSubscription", mock.Anything, userID, domain.TierPremium, 30).
		Return(&domain.Subscription{ID: uuid.New(), UserID: userID, Tier: domain.TierPremium}, nil).Once()

	uc := usecase.NewPaymentUseCase(pRepo, nil, subUC, gw, "https://example.com", nil)

	var ev usecase.WebhookEvent
	ev.Object.ID = "prov-1"
	ev.Object.Status = "succeeded"
	require.NoError(t, uc.HandleWebhook(ctx, ev))
}

func TestPaymentUseCase_HandleWebhook_Succeeded_SecondDelivery_DoesNotActivateTwice(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	p := &domain.Payment{
		ID:           uuid.New(),
		UserID:       userID,
		YookassaID:   "prov-dup",
		Tier:         domain.TierBasic,
		DurationDays: 30,
		Status:       domain.PaymentPending,
	}

	pRepo := &mockPaymentRepo{}
	subUC := &mockSubUC{}
	gw := &mockGateway{}

	pSucceeded := *p
	pSucceeded.Status = domain.PaymentSucceeded
	// Первый webhook: claim прошёл, активация выполнена.
	pRepo.On("ClaimSucceededByYookassaID", mock.Anything, "prov-dup").Return(&pSucceeded, true, nil).Once()
	pRepo.On("ConsumePaymentActivation", mock.Anything, p.ID).Return(true, nil).Once()
	subUC.On("ActivateSubscription", mock.Anything, userID, domain.TierBasic, 30).
		Return(&domain.Subscription{ID: uuid.New(), UserID: userID, Tier: domain.TierBasic}, nil).Once()

	uc := usecase.NewPaymentUseCase(pRepo, nil, subUC, gw, "https://example.com", nil)

	var ev usecase.WebhookEvent
	ev.Object.ID = "prov-dup"
	ev.Object.Status = "succeeded"
	require.NoError(t, uc.HandleWebhook(ctx, ev))

	// Повторная доставка: платёж уже succeeded — ConsumePaymentActivation вернёт false, подписка не дублируется.
	pRepo.On("ClaimSucceededByYookassaID", mock.Anything, "prov-dup").Return(&pSucceeded, false, nil).Once()
	pRepo.On("ConsumePaymentActivation", mock.Anything, p.ID).Return(false, nil).Once()
	require.NoError(t, uc.HandleWebhook(ctx, ev))

	subUC.AssertNumberOfCalls(t, "ActivateSubscription", 1)
}

func TestPaymentUseCase_ListByFilter_CapsPageSize(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pRepo := &mockPaymentRepo{}
	pRepo.On("ListByFilter", mock.Anything, "", mock.Anything, mock.Anything, 200, 0).
		Return([]*domain.Payment{}, int64(0), nil).Once()
	uc := usecase.NewPaymentUseCase(pRepo, nil, &mockSubUC{}, &mockGateway{}, "https://example.com", nil)
	_, _, err := uc.ListByFilter(ctx, "", "month", 1, 9999)
	require.NoError(t, err)
	pRepo.AssertExpectations(t)
}
