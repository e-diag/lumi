package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/repository"
	"github.com/google/uuid"
)

type paymentUseCase struct {
	paymentRepo repository.PaymentRepository
	subUC       SubscriptionUseCase
	gateway     PaymentGateway
	baseURL     string
}

// NewPaymentUseCase создаёт реализацию PaymentUseCase.
func NewPaymentUseCase(paymentRepo repository.PaymentRepository, subUC SubscriptionUseCase, gateway PaymentGateway, baseURL string) PaymentUseCase {
	return &paymentUseCase{
		paymentRepo: paymentRepo,
		subUC:       subUC,
		gateway:     gateway,
		baseURL:     baseURL,
	}
}

func (uc *paymentUseCase) CreatePayment(ctx context.Context, userID uuid.UUID, tier domain.SubscriptionTier, days int) (*domain.Payment, string, error) {
	if tier != domain.TierBasic && tier != domain.TierPremium {
		return nil, "", fmt.Errorf("usecase: create payment: invalid tier: %s", tier)
	}
	if days <= 0 {
		return nil, "", fmt.Errorf("usecase: create payment: invalid days: %d", days)
	}

	amountKopeks, err := calcAmountKopeks(tier, days)
	if err != nil {
		return nil, "", fmt.Errorf("usecase: create payment: %w", err)
	}
	amountValue := formatKopeks(amountKopeks)

	providerPayment, err := uc.gateway.CreatePayment(ctx, PaymentGatewayCreateRequest{
		AmountValue: amountValue,
		Currency:    "RUB",
		ReturnURL:   uc.baseURL + "/payment/return",
		Description: fmt.Sprintf("FreeWay VPN %s (%d days)", tier, days),
		Metadata: map[string]string{
			"user_id": userID.String(),
			"tier":    string(tier),
			"days":    strconv.Itoa(days),
		},
		Capture: true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("usecase: create payment: gateway: %w", err)
	}

	p := &domain.Payment{
		ID:              uuid.New(),
		UserID:          userID,
		YookassaID:      providerPayment.ID,
		AmountRub:       int(math.Round(float64(amountKopeks) / 100.0)), // временно: домен хранит рубли (обновим в рамках фазы 2)
		Tier:            tier,
		DurationDays:    days,
		Status:          domain.PaymentPending,
		ConfirmationURL: providerPayment.ConfirmationURL,
	}

	if err := uc.paymentRepo.Create(ctx, p); err != nil {
		return nil, "", fmt.Errorf("usecase: create payment: save: %w", err)
	}

	return p, p.ConfirmationURL, nil
}

func (uc *paymentUseCase) HandleWebhook(ctx context.Context, event WebhookEvent) error {
	// ЮKassa шлёт: event=payment.succeeded/payment.canceled, object.status=succeeded/canceled
	if event.Object.ID == "" {
		return fmt.Errorf("usecase: handle webhook: empty provider id")
	}

	p, err := uc.paymentRepo.GetByYookassaID(ctx, event.Object.ID)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			// Не нашли платеж — не ретраим бесконечно. Логика idempotent на уровне обработчика.
			return nil
		}
		return fmt.Errorf("usecase: handle webhook: get payment: %w", err)
	}

	switch p.Status {
	case domain.PaymentSucceeded, domain.PaymentCanceled:
		return nil
	}

	switch event.Object.Status {
	case "succeeded":
		p.Status = domain.PaymentSucceeded
		if err := uc.paymentRepo.Update(ctx, p); err != nil {
			return fmt.Errorf("usecase: handle webhook: update payment: %w", err)
		}
		if _, err := uc.subUC.ActivateSubscription(ctx, p.UserID, p.Tier, p.DurationDays); err != nil {
			return fmt.Errorf("usecase: handle webhook: activate subscription: %w", err)
		}
		return nil
	case "canceled":
		p.Status = domain.PaymentCanceled
		if err := uc.paymentRepo.Update(ctx, p); err != nil {
			return fmt.Errorf("usecase: handle webhook: update payment: %w", err)
		}
		return nil
	default:
		return nil
	}
}

func (uc *paymentUseCase) GetPendingPayments(ctx context.Context) ([]*domain.Payment, error) {
	t := time.Now().Add(-5 * time.Minute)
	payments, err := uc.paymentRepo.ListPendingOlderThan(ctx, t)
	if err != nil {
		return nil, fmt.Errorf("usecase: get pending payments: %w", err)
	}
	return payments, nil
}

func (uc *paymentUseCase) CheckAndUpdatePayment(ctx context.Context, paymentID uuid.UUID) error {
	p, err := uc.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return fmt.Errorf("usecase: check payment: get by id: %w", err)
	}
	if p.Status != domain.PaymentPending {
		return nil
	}

	providerPayment, err := uc.gateway.GetPayment(ctx, p.YookassaID)
	if err != nil {
		return fmt.Errorf("usecase: check payment: gateway: %w", err)
	}

	switch providerPayment.Status {
	case "succeeded":
		p.Status = domain.PaymentSucceeded
		if err := uc.paymentRepo.Update(ctx, p); err != nil {
			return fmt.Errorf("usecase: check payment: update: %w", err)
		}
		if _, err := uc.subUC.ActivateSubscription(ctx, p.UserID, p.Tier, p.DurationDays); err != nil {
			return fmt.Errorf("usecase: check payment: activate subscription: %w", err)
		}
	case "canceled":
		p.Status = domain.PaymentCanceled
		if err := uc.paymentRepo.Update(ctx, p); err != nil {
			return fmt.Errorf("usecase: check payment: update: %w", err)
		}
	default:
		// pending — ничего
	}
	return nil
}

func (uc *paymentUseCase) GetByID(ctx context.Context, paymentID uuid.UUID) (*domain.Payment, error) {
	p, err := uc.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("usecase: get payment by id: %w", err)
	}
	return p, nil
}

func (uc *paymentUseCase) CancelStale(ctx context.Context, paymentID uuid.UUID) error {
	p, err := uc.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return fmt.Errorf("usecase: cancel stale: get: %w", err)
	}
	if p.Status != domain.PaymentPending {
		return nil
	}
	p.Status = domain.PaymentCanceled
	if err := uc.paymentRepo.Update(ctx, p); err != nil {
		return fmt.Errorf("usecase: cancel stale: update: %w", err)
	}
	return nil
}

func (uc *paymentUseCase) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Payment, error) {
	payments, err := uc.paymentRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("usecase: list payments by user: %w", err)
	}
	return payments, nil
}

func (uc *paymentUseCase) ListByFilter(ctx context.Context, status string, period string, page, pageSize int) ([]*domain.Payment, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	now := time.Now()
	var from *time.Time
	var to *time.Time
	switch period {
	case "today":
		t := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		from = &t
	case "week":
		t := now.AddDate(0, 0, -7)
		from = &t
	case "month":
		t := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		from = &t
	}
	to = &now

	rows, total, err := uc.paymentRepo.ListByFilter(ctx, status, from, to, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase: list payments by filter: %w", err)
	}
	return rows, total, nil
}

func calcAmountKopeks(tier domain.SubscriptionTier, days int) (int64, error) {
	price30, ok := map[domain.SubscriptionTier]int64{
		domain.TierBasic:   14900,
		domain.TierPremium: 29900,
	}[tier]
	if !ok {
		return 0, fmt.Errorf("unknown tier: %s", tier)
	}
	if days <= 0 {
		return 0, fmt.Errorf("invalid days: %d", days)
	}

	// Пропорционально: price30 * days / 30. Округляем до копейки (математика целыми).
	amount := price30 * int64(days)
	kopeks := amount / 30
	rem := amount % 30
	if rem*2 >= 30 {
		kopeks++
	}
	return kopeks, nil
}

func formatKopeks(kopeks int64) string {
	rub := kopeks / 100
	kop := kopeks % 100
	return fmt.Sprintf("%d.%02d", rub, kop)
}

