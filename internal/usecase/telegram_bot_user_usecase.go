package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/repository"
	"github.com/google/uuid"
)

type telegramBotUserUseCase struct {
	userRepo  repository.UserRepository
	subUC     SubscriptionUseCase
	abuseRepo repository.BotAntiAbuseRepository
	settingsRepo             repository.ProductSettingsRepository
	// maxTrialsPerIP — максимум выданных триалов на один IP за всё время; 0 = не проверять.
	maxTrialsPerIP int
	// referralBonusMaxPerMonth — максимум реферальных +3 дня у одного пригласившего в календарном месяце; 0 = без лимита.
	referralBonusMaxPerMonth int
	// trialGlobalCapPer24h — лимит выданных триалов за последние 24 ч по всему сервису; 0 = без лимита.
	trialGlobalCapPer24h int
}

// NewTelegramBotUserUseCase создаёт сценарий онбординга пользователя Telegram-бота.
// abuseRepo может быть nil — проверки по IP/реферальному лимиту отключаются.
// settingsRepo может быть nil — триал и реферал по умолчанию 3 дня.
func NewTelegramBotUserUseCase(
	userRepo repository.UserRepository,
	subUC SubscriptionUseCase,
	abuseRepo repository.BotAntiAbuseRepository,
	maxTrialsPerIP int,
	referralBonusMaxPerMonth int,
	settingsRepo repository.ProductSettingsRepository,
	trialGlobalCapPer24h int,
) TelegramBotUserUseCase {
	return &telegramBotUserUseCase{
		userRepo:                 userRepo,
		subUC:                    subUC,
		abuseRepo:                abuseRepo,
		settingsRepo:             settingsRepo,
		maxTrialsPerIP:           maxTrialsPerIP,
		referralBonusMaxPerMonth: referralBonusMaxPerMonth,
		trialGlobalCapPer24h:     trialGlobalCapPer24h,
	}
}

// OnStart обрабатывает вход пользователя: регистрация, триал, реферальный бонус пригласившему.
func (uc *telegramBotUserUseCase) OnStart(ctx context.Context, telegramID int64, username string, referrerUserID *uuid.UUID, client TelegramClientMeta) (*domain.User, *TelegramStartOutcome, error) {
	ip := strings.TrimSpace(client.IP)
	ua := strings.TrimSpace(client.UserAgent)

	existing, err := uc.userRepo.GetByTelegramID(ctx, telegramID)
	if err == nil {
		return uc.finishWelcomeForExisting(ctx, existing, ip, ua)
	}
	if !errors.Is(err, domain.ErrUserNotFound) {
		return nil, nil, fmt.Errorf("usecase: telegram bot on start: %w", err)
	}

	var inviter *domain.User
	if referrerUserID != nil {
		inv, ierr := uc.userRepo.GetByID(ctx, *referrerUserID)
		if ierr == nil && inv.TelegramID != telegramID {
			inviter = inv
		}
	}

	u := domain.NewUser(telegramID, username)
	if inviter != nil {
		rid := inviter.ID
		u.ReferredBy = &rid
	}
	if err := uc.userRepo.Create(ctx, u); err != nil {
		return nil, nil, fmt.Errorf("usecase: telegram bot create user: %w", err)
	}

	out := &TelegramStartOutcome{IsNewUser: true, TrialGranted: false}
	trialDays, _ := uc.productPolicy(ctx)
	out.TrialDays = trialDays

	trialOK := uc.canGrantTrialByIP(ctx, ip)
	if !trialOK {
		u.WelcomeBonusUsed = true
		if err := uc.userRepo.Update(ctx, u); err != nil {
			return u, out, fmt.Errorf("usecase: telegram bot mark welcome skipped ip: %w", err)
		}
		out.TrialSkippedByIP = true
		_ = uc.recordTrialSignup(ctx, u.ID, telegramID, ip, ua, false)
		return u, out, nil
	}
	if !uc.canGrantTrialGlobally(ctx) {
		u.WelcomeBonusUsed = true
		if err := uc.userRepo.Update(ctx, u); err != nil {
			return u, out, fmt.Errorf("usecase: telegram bot mark welcome skipped global: %w", err)
		}
		out.TrialSkippedGlobal = true
		_ = uc.recordTrialSignup(ctx, u.ID, telegramID, ip, ua, false)
		return u, out, nil
	}

	if _, err := uc.subUC.ActivateSubscription(ctx, u.ID, domain.TierBasic, trialDays); err != nil {
		return u, out, fmt.Errorf("usecase: telegram bot welcome trial: %w", err)
	}
	u.WelcomeBonusUsed = true
	if err := uc.userRepo.Update(ctx, u); err != nil {
		return u, out, fmt.Errorf("usecase: telegram bot mark welcome used: %w", err)
	}
	out.TrialGranted = true
	if err := uc.recordTrialSignup(ctx, u.ID, telegramID, ip, ua, true); err != nil {
		slog.Warn("telegram bot: trial audit insert failed", "error", err)
	}

	if inviter != nil {
		uc.tryReferralBonus(ctx, inviter, u, ip)
	}
	return u, out, nil
}

func (uc *telegramBotUserUseCase) canGrantTrialByIP(ctx context.Context, ip string) bool {
	if uc.abuseRepo == nil || uc.maxTrialsPerIP <= 0 || ip == "" {
		return true
	}
	n, err := uc.abuseRepo.CountTrialGrantsByIP(ctx, ip)
	if err != nil {
		slog.Warn("telegram bot: count trials by ip failed", "error", err)
		return true
	}
	return n < int64(uc.maxTrialsPerIP)
}

func (uc *telegramBotUserUseCase) canGrantTrialGlobally(ctx context.Context) bool {
	if uc.abuseRepo == nil || uc.trialGlobalCapPer24h <= 0 {
		return true
	}
	since := time.Now().Add(-24 * time.Hour)
	n, err := uc.abuseRepo.CountTrialGrantsGloballySince(ctx, since)
	if err != nil {
		slog.Warn("telegram bot: count trials global failed", "error", err)
		return true
	}
	return n < int64(uc.trialGlobalCapPer24h)
}

func (uc *telegramBotUserUseCase) productPolicy(ctx context.Context) (trialDays, referralBonus int) {
	trialDays, referralBonus = 3, 3
	if uc.settingsRepo == nil {
		return
	}
	s, err := uc.settingsRepo.Get(ctx)
	if err != nil {
		slog.Warn("telegram bot: product settings load failed", "error", err)
		return
	}
	if s.TrialDays > 0 {
		trialDays = s.TrialDays
	}
	if s.ReferralBonusDays > 0 {
		referralBonus = s.ReferralBonusDays
	}
	return
}

func (uc *telegramBotUserUseCase) recordTrialSignup(ctx context.Context, userID uuid.UUID, telegramID int64, ip, ua string, granted bool) error {
	if uc.abuseRepo == nil {
		return nil
	}
	row := &domain.BotTrialSignup{
		ID:           uuid.New(),
		TelegramID:   telegramID,
		UserID:       &userID,
		IP:           truncateRunes(ip, 128),
		UserAgent:    truncateRunes(ua, 512),
		TrialGranted: granted,
		CreatedAt:    time.Now(),
	}
	return uc.abuseRepo.InsertTrialSignup(ctx, row)
}

func (uc *telegramBotUserUseCase) tryReferralBonus(ctx context.Context, inviter, grantee *domain.User, granteeIP string) {
	if uc.abuseRepo == nil {
		uc.applyReferralBonus(ctx, inviter, grantee.ID)
		return
	}
	if granteeIP != "" {
		invIP, err := uc.abuseRepo.GetFirstTrialSignupIPForUser(ctx, inviter.ID)
		if err != nil {
			slog.Warn("telegram bot: inviter trial ip lookup failed", "error", err)
		}
		if invIP != "" && invIP == granteeIP {
			slog.Info("telegram bot: referral skipped (same ip as inviter trial)", "inviter_id", inviter.ID)
			return
		}
	}
	if uc.referralBonusMaxPerMonth > 0 {
		start := monthStart(time.Now())
		n, err := uc.abuseRepo.CountReferralGrantsSince(ctx, inviter.ID, start)
		if err != nil {
			slog.Warn("telegram bot: referral monthly count failed", "error", err)
		} else if n >= int64(uc.referralBonusMaxPerMonth) {
			slog.Info("telegram bot: referral skipped (monthly cap)", "inviter_id", inviter.ID)
			return
		}
	}
	uc.applyReferralBonus(ctx, inviter, grantee.ID)
}

func (uc *telegramBotUserUseCase) applyReferralBonus(ctx context.Context, inviter *domain.User, granteeID uuid.UUID) {
	_, bonus := uc.productPolicy(ctx)
	if err := uc.subUC.AddBonusDays(ctx, inviter.ID, bonus); err != nil {
		slog.Warn("telegram bot: referrer bonus failed", "inviter_id", inviter.ID, "error", err)
		return
	}
	if uc.abuseRepo != nil {
		log := &domain.ReferralBonusLog{
			ID:        uuid.New(),
			InviterID: inviter.ID,
			GranteeID: granteeID,
			CreatedAt: time.Now(),
		}
		if err := uc.abuseRepo.InsertReferralGrant(ctx, log); err != nil {
			slog.Warn("telegram bot: referral grant log failed", "error", err)
		}
	}
}

func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func truncateRunes(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}

// finishWelcomeForExisting доначивает приветственный триал только если в БД нет строки подписки (сбой после Create, регистрация только через API).
func (uc *telegramBotUserUseCase) finishWelcomeForExisting(ctx context.Context, u *domain.User, ip, ua string) (*domain.User, *TelegramStartOutcome, error) {
	trialDays, _ := uc.productPolicy(ctx)
	if u.WelcomeBonusUsed {
		return u, &TelegramStartOutcome{IsNewUser: false, TrialGranted: false}, nil
	}
	sub, err := uc.subUC.GetUserSubscription(ctx, u.ID)
	if err != nil {
		if !errors.Is(err, domain.ErrSubscriptionNotFound) {
			return nil, nil, fmt.Errorf("usecase: telegram bot existing get sub: %w", err)
		}
		if !uc.canGrantTrialByIP(ctx, ip) {
			u.WelcomeBonusUsed = true
			if err := uc.userRepo.Update(ctx, u); err != nil {
				return nil, nil, fmt.Errorf("usecase: telegram bot deferred welcome skip ip: %w", err)
			}
			_ = uc.recordTrialSignup(ctx, u.ID, u.TelegramID, ip, ua, false)
			return u, &TelegramStartOutcome{IsNewUser: false, TrialGranted: false, TrialSkippedByIP: true}, nil
		}
		if !uc.canGrantTrialGlobally(ctx) {
			u.WelcomeBonusUsed = true
			if err := uc.userRepo.Update(ctx, u); err != nil {
				return nil, nil, fmt.Errorf("usecase: telegram bot deferred welcome skip global: %w", err)
			}
			_ = uc.recordTrialSignup(ctx, u.ID, u.TelegramID, ip, ua, false)
			return u, &TelegramStartOutcome{IsNewUser: false, TrialGranted: false, TrialSkippedGlobal: true}, nil
		}
		if _, aerr := uc.subUC.ActivateSubscription(ctx, u.ID, domain.TierBasic, trialDays); aerr != nil {
			return u, &TelegramStartOutcome{IsNewUser: false, TrialGranted: false}, fmt.Errorf("usecase: telegram bot deferred welcome trial: %w", aerr)
		}
		u.WelcomeBonusUsed = true
		if uerr := uc.userRepo.Update(ctx, u); uerr != nil {
			return nil, nil, fmt.Errorf("usecase: telegram bot deferred welcome update: %w", uerr)
		}
		if err := uc.recordTrialSignup(ctx, u.ID, u.TelegramID, ip, ua, true); err != nil {
			slog.Warn("telegram bot: deferred trial audit insert failed", "error", err)
		}
		if u.ReferredBy != nil {
			inv, ierr := uc.userRepo.GetByID(ctx, *u.ReferredBy)
			if ierr == nil && inv.TelegramID != u.TelegramID {
				uc.tryReferralBonus(ctx, inv, u, ip)
			}
		}
		return u, &TelegramStartOutcome{IsNewUser: false, TrialGranted: true, TrialDays: trialDays}, nil
	}
	if sub.IsActive() {
		u.WelcomeBonusUsed = true
		if err := uc.userRepo.Update(ctx, u); err != nil {
			return nil, nil, fmt.Errorf("usecase: telegram bot sync welcome flag: %w", err)
		}
		return u, &TelegramStartOutcome{IsNewUser: false, TrialGranted: false}, nil
	}
	u.WelcomeBonusUsed = true
	if err := uc.userRepo.Update(ctx, u); err != nil {
		return nil, nil, fmt.Errorf("usecase: telegram bot mark welcome consumed: %w", err)
	}
	return u, &TelegramStartOutcome{IsNewUser: false, TrialGranted: false}, nil
}
