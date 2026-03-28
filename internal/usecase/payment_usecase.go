package usecase

import (
	"context"
	"fmt"
	"log/slog"
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
	notify      PaymentSuccessNotifier
}

// NewPaymentUseCase создаёт реализацию PaymentUseCase.
// notify может быть nil — тогда сообщения в Telegram после оплаты не отправляются.
func NewPaymentUseCase(paymentRepo repository.PaymentRepository, subUC SubscriptionUseCase, gateway PaymentGateway, baseURL string, notify PaymentSuccessNotifier) PaymentUseCase {
	return &paymentUseCase{
		paymentRepo: paymentRepo,
		subUC:       subUC,
		gateway:     gateway,
		baseURL:     baseURL,
		notify:      notify,
	}
}

func (uc *paymentUseCase) notifyPaymentSuccess(ctx context.Context, userID uuid.UUID) {
	if uc.notify == nil {
		return
	}
	if err := uc.notify.NotifySubscriptionPaid(ctx, userID); err != nil {
		slog.Warn("payment: telegram notify failed", "user_id", userID, "error", err)
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
	if event.Object.ID == "" {
		return fmt.Errorf("usecase: handle webhook: empty provider id")
	}

	switch event.Object.Status {
	case "succeeded":
		p, err := uc.applySucceededByYookassaID(ctx, event.Object.ID)
		if err != nil {
			return err
		}
		if p == nil {
			return nil
		}
		if err := uc.finalizeSucceededPayment(ctx, p); err != nil {
			return fmt.Errorf("usecase: handle webhook: finalize: %w", err)
		}
		return nil
	case "canceled":
		if _, _, err := uc.paymentRepo.ClaimCanceledByYookassaID(ctx, event.Object.ID); err != nil {
			return fmt.Errorf("usecase: handle webhook: claim canceled: %w", err)
		}
		return nil
	default:
		return nil
	}
}

// applySucceededByYookassaID переводит платёж в succeeded (идемпотентно) и возвращает актуальную строку.
func (uc *paymentUseCase) applySucceededByYookassaID(ctx context.Context, yookassaID string) (*domain.Payment, error) {
	p, claimed, err := uc.paymentRepo.ClaimSucceededByYookassaID(ctx, yookassaID)
	if err != nil {
		return nil, fmt.Errorf("usecase: claim succeeded: %w", err)
	}
	if p == nil {
		slog.Warn("payment webhook: unknown yookassa id", "yookassa_id", yookassaID)
		return nil, nil
	}
	if !claimed && p.Status != domain.PaymentSucceeded {
		return nil, nil
	}
	return p, nil
}

// applySucceededByID — то же для внутреннего UUID (воркер опроса ЮKassa).
func (uc *paymentUseCase) applySucceededByID(ctx context.Context, paymentID uuid.UUID) (*domain.Payment, error) {
	p, claimed, err := uc.paymentRepo.ClaimSucceededByID(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("usecase: claim succeeded by id: %w", err)
	}
	if p == nil {
		return nil, nil
	}
	if !claimed && p.Status != domain.PaymentSucceeded {
		return nil, nil
	}
	return p, nil
}

// finalizeSucceededPayment однократно активирует подписку по успешному платежу (ledger + release при ошибке).
func (uc *paymentUseCase) finalizeSucceededPayment(ctx context.Context, p *domain.Payment) error {
	if p.Status != domain.PaymentSucceeded {
		return nil
	}
	consumed, err := uc.paymentRepo.ConsumePaymentActivation(ctx, p.ID)
	if err != nil {
		return fmt.Errorf("usecase: consume payment activation: %w", err)
	}
	if !consumed {
		return nil
	}
	_, subErr := uc.subUC.ActivateSubscription(ctx, p.UserID, p.Tier, p.DurationDays)
	if subErr != nil {
		if relErr := uc.paymentRepo.ReleasePaymentActivation(ctx, p.ID); relErr != nil {
			slog.Error("payment: release activation after failure", "payment_id", p.ID, "error", relErr)
		}
		return fmt.Errorf("usecase: activate subscription: %w", subErr)
	}
	slog.Info("payment: subscription activated", "payment_id", p.ID, "user_id", p.UserID, "tier", p.Tier, "days", p.DurationDays)
	uc.notifyPaymentSuccess(ctx, p.UserID)
	return nil
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
		p2, err := uc.applySucceededByID(ctx, paymentID)
		if err != nil {
			return fmt.Errorf("usecase: check payment: apply succeeded: %w", err)
		}
		if p2 == nil {
			return nil
		}
		if err := uc.finalizeSucceededPayment(ctx, p2); err != nil {
			return fmt.Errorf("usecase: check payment: finalize: %w", err)
		}
	case "canceled":
		if _, _, err := uc.paymentRepo.ClaimCanceledByID(ctx, paymentID); err != nil {
			return fmt.Errorf("usecase: check payment: claim canceled: %w", err)
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

