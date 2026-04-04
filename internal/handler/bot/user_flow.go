package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/usecase"
)

// PublicSettings — публичные URL и параметры пользовательского сценария бота.
type PublicSettings struct {
	BaseURL            string
	BotUsername        string
	AppURLIOS          string // ссылка на Happ в App Store
	AppURLAndroid      string // ссылка на v2RayTun в Google Play / RuStore
	PaymentDefaultDays int
	SupportURL         string // поддержка (t.me/..., https://...)
	// PaymentsEnabled — ЮKassa настроена в bootstrap; иначе кнопка оплаты скрыта, ключи и триал работают.
	PaymentsEnabled bool
}

// Callback-префиксы пользовательского сценария (лимит Telegram 64 байта).
const (
	cbUserMenu = "u:m"
	cbUserConn = "u:c"
	cbUserProf = "u:p"
	cbUserPay  = "u:$"
	cbUserInst = "u:i"
	cbUserRef  = "u:r"
	cbUserKeys = "u:k"
	cbUserSup  = "u:s"
	// legacy callbacks (старые клавиатуры в чате)
	cbUserSpeed = "u:sp"
	cbUserHelp  = "u:hp"

	cbConnHapp = "u:ch"
	cbConnV2   = "u:cv"
	cbConnCopy = "u:cc"

	cbBuyTierBasic   = "u:tb"
	cbBuyTierPremium = "u:tp"
	cbPlanPrefix     = "u:pl:"
)

func (h *Handler) subscriptionURL(subToken string) string {
	base := strings.TrimSuffix(strings.TrimSpace(h.pub.BaseURL), "/")
	if base == "" {
		return "/sub/" + subToken
	}
	return base + "/sub/" + subToken
}

// onPublicStart — команда /start для всех пользователей (не менеджерский вход).
func (h *Handler) onPublicStart(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}
	from := update.Message.From
	chatID := update.Message.Chat.ID

	if !h.startRL.Allow(from.ID, 15, time.Minute) {
		h.sendPlain(ctx, b, chatID, "Слишком много запросов. Подождите минуту и попробуйте снова.")
		return
	}

	username := strings.TrimSpace(from.Username)
	if username == "" {
		username = strings.TrimSpace(strings.TrimSpace(from.FirstName + " " + from.LastName))
	}

	refID := ParseReferralUserID(update.Message.Text)
	clientMeta := ExtractTelegramClientMeta(update)
	_, out, err := h.botUserUC.OnStart(ctx, from.ID, username, refID, clientMeta)
	if err != nil {
		slog.Error("bot user on start failed", "error", err, "telegram_id", from.ID)
		h.sendPlain(ctx, b, chatID, "Не удалось зарегистрировать профиль. Попробуйте позже или напишите в поддержку.")
		return
	}

	text := composeWelcomeText(out, h.pub.PaymentsEnabled)
	params := &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: h.userMainMenu(),
	}
	if _, err := b.SendMessage(ctx, params); err != nil {
		slog.Error("bot send user menu failed", "error", err)
	}
}

func composeWelcomeText(out *usecase.TelegramStartOutcome, paymentsEnabled bool) string {
	if out == nil {
		return "FreeWay VPN — доступ через Telegram, ключи для Happ и v2RayTun.\n\nВыберите действие:"
	}
	var b strings.Builder
	b.WriteString("FreeWay VPN — доступ через Telegram, ключи для Happ и v2RayTun.\n\n")
	if out.IsNewUser && out.TrialGranted && out.TrialDays > 0 {
		b.WriteString(fmt.Sprintf("Вам включили %d %s.\n\n", out.TrialDays, trialDaysWordRu(out.TrialDays)))
	}
	if out.TrialSkippedByIP {
		if paymentsEnabled {
			b.WriteString("С этой сети пробный период уже выдавали. Можно сразу купить доступ в меню.\n\n")
		} else {
			b.WriteString("С этой сети пробный период уже выдавали. Для доступа свяжитесь с поддержкой или администратором.\n\n")
		}
	}
	if out.TrialSkippedGlobal {
		if paymentsEnabled {
			b.WriteString("Сейчас достигнут лимит пробных периодов на сервисе. Оформите платный доступ в меню.\n\n")
		} else {
			b.WriteString("Сейчас достигнут лимит пробных периодов на сервисе. Напишите в поддержку.\n\n")
		}
	}
	b.WriteString("Выберите действие:")
	return b.String()
}

func trialDaysWordRu(n int) string {
	n = n % 100
	if n >= 11 && n <= 14 {
		return "пробных дней"
	}
	switch n % 10 {
	case 1:
		return "пробный день"
	case 2, 3, 4:
		return "пробных дня"
	default:
		return "пробных дней"
	}
}

// userMainMenu — главное меню пользователя (Telegram-first).
func (h *Handler) userMainMenu() models.InlineKeyboardMarkup {
	rowProfile := []models.InlineKeyboardButton{{Text: "👤 Профиль", CallbackData: cbUserProf}}
	if h.pub.PaymentsEnabled {
		rowProfile = append(rowProfile, models.InlineKeyboardButton{Text: "💳 Купить подписку", CallbackData: cbUserPay})
	}
	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "🔌 Подключить VPN", CallbackData: cbUserConn}},
			rowProfile,
			{{Text: "🔑 Мои ключи / подписка", CallbackData: cbUserKeys}},
			{
				{Text: "📖 Инструкция", CallbackData: cbUserInst},
				{Text: "🎁 Пригласить друга", CallbackData: cbUserRef},
			},
			{{Text: "💬 Поддержка", CallbackData: cbUserSup}},
		},
	}
}

func tierPickKeyboard() models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "🔵 Базовый", CallbackData: cbBuyTierBasic}},
			{{Text: "💎 Премиум", CallbackData: cbBuyTierPremium}},
			{{Text: "← Меню", CallbackData: cbUserMenu}},
		},
	}
}

func periodBuyKeyboard(tierLetter string) models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "1 месяц", CallbackData: fmt.Sprintf("u:p1:%s", tierLetter)}},
			{{Text: "3 месяца", CallbackData: fmt.Sprintf("u:p3:%s", tierLetter)}},
			{{Text: "12 месяцев", CallbackData: fmt.Sprintf("u:p12:%s", tierLetter)}},
			{{Text: "← К тарифам", CallbackData: cbUserPay}},
			{{Text: "🏠 Меню", CallbackData: cbUserMenu}},
		},
	}
}

// onUserCallback обрабатывает callback пользовательского меню (префикс u:).
func (h *Handler) onUserCallback(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.CallbackQuery == nil || update.CallbackQuery.Message.Message == nil {
		return
	}
	cq := update.CallbackQuery
	msg := cq.Message.Message
	chatID := msg.Chat.ID
	fromID := cq.From.ID
	data := cq.Data

	if !h.callbackRL.Allow(fromID, 40, time.Minute) {
		h.answerCQ(ctx, b, cq.ID, "Слишком много нажатий. Подождите немного.")
		return
	}

	switch data {
	case cbUserMenu:
		h.editUserText(ctx, b, chatID, msg.ID, "Главное меню", h.userMainMenu())
	case cbUserKeys:
		h.handleUserKeys(ctx, b, chatID, msg.ID, fromID)
	case cbUserSup:
		h.handleUserSupport(ctx, b, chatID, msg.ID)
	case cbUserConn:
		h.handleConnectStart(ctx, b, chatID, msg.ID, fromID)
	case cbConnHapp:
		h.showConnectDetail(ctx, b, chatID, msg.ID, fromID, "happ")
	case cbConnV2:
		h.showConnectDetail(ctx, b, chatID, msg.ID, fromID, "v2raytun")
	case cbConnCopy:
		h.handleConnectCopy(ctx, b, chatID, msg.ID, fromID)
	case cbUserProf:
		h.handleUserProfile(ctx, b, chatID, msg.ID, fromID)
	case cbUserPay:
		h.showPlanPicker(ctx, b, chatID, msg.ID)
	case cbBuyTierBasic:
		h.editUserText(ctx, b, chatID, msg.ID, "Базовый — выберите срок:", periodBuyKeyboard("b"))
	case cbBuyTierPremium:
		h.editUserText(ctx, b, chatID, msg.ID, "Премиум — выберите срок:", periodBuyKeyboard("p"))
	case cbUserInst:
		h.editUserText(ctx, b, chatID, msg.ID, h.instructionText(), h.userMainMenu())
	case cbUserSpeed:
		h.handleSpeedTest(ctx, b, chatID, msg.ID)
	case cbUserHelp:
		h.editUserText(ctx, b, chatID, msg.ID, h.instructionText(), h.userMainMenu())
	case cbUserRef:
		h.handleUserReferral(ctx, b, chatID, msg.ID, fromID)
	default:
		if strings.HasPrefix(data, cbPlanPrefix) {
			code := strings.TrimPrefix(data, cbPlanPrefix)
			h.handleUserPaymentPlan(ctx, b, chatID, msg.ID, fromID, code)
			break
		}
		if tier, days, ok := parsePayCallback(data); ok {
			h.handleUserPayment(ctx, b, chatID, msg.ID, fromID, tier, days)
		}
	}
	h.answerCQ(ctx, b, cq.ID, "")
}

func parsePayCallback(data string) (domain.SubscriptionTier, int, bool) {
	if !strings.HasPrefix(data, "u:p") {
		return "", 0, false
	}
	rest := strings.TrimPrefix(data, "u:p")
	idx := strings.Index(rest, ":")
	if idx <= 0 || idx >= len(rest)-1 {
		return "", 0, false
	}
	monthsStr := rest[:idx]
	tierLetter := rest[idx+1:]
	m, err := strconv.Atoi(monthsStr)
	if err != nil {
		return "", 0, false
	}
	var days int
	switch m {
	case 1:
		days = 30
	case 3:
		days = 90
	case 12:
		days = 365
	default:
		return "", 0, false
	}
	switch tierLetter {
	case "b":
		return domain.TierBasic, days, true
	case "p":
		return domain.TierPremium, days, true
	default:
		return "", 0, false
	}
}

func (h *Handler) handleConnectStart(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, telegramID int64) {
	u, err := h.userUC.GetByTelegramID(ctx, telegramID)
	if err != nil {
		h.editUserText(ctx, b, chatID, messageID, "Профиль не найден. Нажмите /start.", h.userMainMenu())
		return
	}
	subURL := h.subscriptionURL(u.SubToken)
	h.setLastSubURL(telegramID, subURL)
	h.editUserText(ctx, b, chatID, messageID, "Выберите приложение — так мы покажем нужный формат и шаги импорта:", connectPlatformKeyboard())
}

func (h *Handler) showConnectDetail(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, telegramID int64, platform string) {
	u, err := h.userUC.GetByTelegramID(ctx, telegramID)
	if err != nil {
		h.editUserText(ctx, b, chatID, messageID, "Профиль не найден. Нажмите /start.", h.userMainMenu())
		return
	}
	subURL := h.subscriptionURL(u.SubToken)
	h.setLastSubURL(telegramID, subURL)
	if _, err := h.configUC.GenerateSubscription(ctx, u.ID); err != nil {
		slog.Error("bot connect: generate subscription check failed", "error", err, "user_id", u.ID)
		text := "Сейчас сервис временно недоступен. Попробуйте позже.\n\nЕсли приложение уже стоит, ссылка ниже может помочь после восстановления."
		h.editUserText(ctx, b, chatID, messageID, text, connectDetailKeyboard(subURL, h.pub.AppURLIOS, h.pub.AppURLAndroid, platform))
		return
	}
	text := connectInstructionText(platform, subURL)
	h.editUserText(ctx, b, chatID, messageID, text, connectDetailKeyboard(subURL, h.pub.AppURLIOS, h.pub.AppURLAndroid, platform))
}

func (h *Handler) handleConnectCopy(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, telegramID int64) {
	url := h.getLastSubURL(telegramID)
	if url == "" {
		u, err := h.userUC.GetByTelegramID(ctx, telegramID)
		if err != nil {
			h.editUserText(ctx, b, chatID, messageID, "Профиль не найден. Нажмите /start.", h.userMainMenu())
			return
		}
		url = h.subscriptionURL(u.SubToken)
	}
	text := "Ссылка для ручного ввода:\n\n" + url + "\n\nНажмите и удерживайте → Копировать → в приложении выберите импорт по ссылке."
	h.editUserText(ctx, b, chatID, messageID, text, connectPlatformKeyboard())
}

func (h *Handler) handleSpeedTest(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int) {
	kb := models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "🌐 Открыть проверку", URL: "https://fast.com"}},
			{{Text: "🏠 Меню", CallbackData: cbUserMenu}},
		},
	}
	h.editUserText(ctx, b, chatID, messageID,
		"⚡ Проверка скорости\n\nВключите защиту в приложении и откройте проверку в браузере.",
		kb)
}

func (h *Handler) instructionText() string {
	payLine := "4) «Купить подписку» — оплата картой (ЮKassa), если включена в настройках сервиса."
	if !h.pub.PaymentsEnabled {
		payLine = "4) Продление доступа — через поддержку или администратора (онлайн-оплата в боте не настроена)."
	}
	return `📖 Как подключиться

1) «Подключить VPN» → Happ (iOS) или v2RayTun (Android).
2) Установите приложение по кнопке «Скачать», если ещё не стоит.
3) «Импорт подписки» / «Подключиться» — откроется клиент, либо скопируйте ссылку подписки.
` + payLine + `

Одна подписка — до 3 устройств (лимит в панели 3x-ui).

Подробнее: docs/SUPPORTED_CLIENTS.md`
}

func (h *Handler) handleUserProfile(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, telegramID int64) {
	u, err := h.userUC.GetByTelegramID(ctx, telegramID)
	if err != nil {
		h.editUserText(ctx, b, chatID, messageID, "Профиль не найден. Нажмите /start.", h.userMainMenu())
		return
	}
	sub, err := h.subUC.GetUserSubscription(ctx, u.ID)
	statusLine := "Нужна подписка"
	expires := "—"
	tier := domain.TierFree
	if err != nil {
		if !errors.Is(err, domain.ErrSubscriptionNotFound) {
			slog.Error("bot profile get sub", "error", err)
			statusLine = "Не удалось загрузить статус"
		}
	} else if sub != nil {
		tier = sub.Tier
		if sub.IsActive() {
			statusLine = "Активна"
			expires = sub.ExpiresAt.In(time.Local).Format("02.01.2006")
		} else {
			statusLine = "Истекла"
			expires = sub.ExpiresAt.In(time.Local).Format("02.01.2006")
		}
	}
	limits := domain.TierLimitsMap[tier]
	devices := fmt.Sprintf("до %d устройств", limits.Devices)
	subURL := h.subscriptionURL(u.SubToken)
	connHint := "Откройте «Подключиться», чтобы получить ссылку для клиента."
	if statusLine == "Нужна подписка" || statusLine == "Истекла" {
		if h.pub.PaymentsEnabled {
			connHint = "Оформите доступ в разделе «Купить подписку»."
		} else {
			connHint = "Свяжитесь с поддержкой или администратором для продления доступа."
		}
	}

	text := fmt.Sprintf(
		"👤 Профиль\n\n"+
			"Статус: %s\n"+
			"До: %s\n"+
			"Устройства: %s\n"+
			"Ссылка подписки:\n%s\n"+
			"Подключение: %s",
		statusLine,
		expires,
		devices,
		subURL,
		connHint,
	)
	h.editUserText(ctx, b, chatID, messageID, text, h.userMainMenu())
}

func (h *Handler) handleUserReferral(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, telegramID int64) {
	u, err := h.userUC.GetByTelegramID(ctx, telegramID)
	if err != nil {
		h.editUserText(ctx, b, chatID, messageID, "Профиль не найден. Нажмите /start.", h.userMainMenu())
		return
	}
	link := BuildReferralLink(h.pub.BotUsername, u.ID)
	if link == "" {
		h.editUserText(ctx, b, chatID, messageID,
			"Приглашения пока недоступны: задайте имя бота в настройках (TELEGRAM_BOT_USERNAME).",
			h.userMainMenu())
		return
	}
	bonus := 3
	if h.settingsRepo != nil {
		if s, err := h.settingsRepo.Get(ctx); err == nil && s.ReferralBonusDays > 0 {
			bonus = s.ReferralBonusDays
		}
	}
	text := fmt.Sprintf(
		"🎁 Пригласите друга\n\n%s\n\nКогда друг запустит бота по ссылке, вы получите +%d %s к подписке (если это новый пользователь и не та же сеть).",
		link,
		bonus,
		referralBonusWordRu(bonus),
	)
	h.editUserText(ctx, b, chatID, messageID, text, h.userMainMenu())
}

func referralBonusWordRu(n int) string {
	n = n % 100
	if n >= 11 && n <= 14 {
		return "дней"
	}
	switch n % 10 {
	case 1:
		return "день"
	case 2, 3, 4:
		return "дня"
	default:
		return "дней"
	}
}

func (h *Handler) showPlanPicker(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int) {
	if !h.pub.PaymentsEnabled {
		h.editUserText(ctx, b, chatID, messageID,
			"Онлайн-оплата не настроена. Ссылку подписки для Happ / v2RayTun смотрите в «Мои ключи / подписка»; продление — через поддержку или администратора.",
			h.userMainMenu())
		return
	}
	if h.paymentUC == nil {
		h.editUserText(ctx, b, chatID, messageID, "Оплата временно недоступна.", h.userMainMenu())
		return
	}
	plans, err := h.paymentUC.ListActivePlans(ctx)
	if err != nil {
		slog.Error("bot list plans failed", "error", err)
		h.editUserText(ctx, b, chatID, messageID, "Не удалось загрузить тарифы. Попробуйте позже.", h.userMainMenu())
		return
	}
	if len(plans) == 0 {
		h.editUserText(ctx, b, chatID, messageID, "Выберите тип подписки и срок:", tierPickKeyboard())
		return
	}
	h.editUserText(ctx, b, chatID, messageID, "Выберите тариф:", plansBuyKeyboard(plans))
}

func plansBuyKeyboard(plans []*domain.Plan) models.InlineKeyboardMarkup {
	rows := make([][]models.InlineKeyboardButton, 0, len(plans)+1)
	for _, p := range plans {
		label := fmt.Sprintf("%s — %d ₽", p.Name, p.PriceKopeks/100)
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: label, CallbackData: cbPlanPrefix + p.Code},
		})
	}
	rows = append(rows, []models.InlineKeyboardButton{{Text: "← Меню", CallbackData: cbUserMenu}})
	return models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (h *Handler) handleUserPaymentPlan(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, telegramID int64, planCode string) {
	if !h.pub.PaymentsEnabled || h.paymentUC == nil {
		h.editUserText(ctx, b, chatID, messageID, "Онлайн-оплата не настроена.", h.userMainMenu())
		return
	}
	u, err := h.userUC.GetByTelegramID(ctx, telegramID)
	if err != nil {
		h.editUserText(ctx, b, chatID, messageID, "Профиль не найден. Нажмите /start.", h.userMainMenu())
		return
	}
	p, payURL, err := h.paymentUC.CreatePaymentByPlanCode(ctx, u.ID, planCode)
	if err != nil {
		slog.Error("bot user payment by plan failed", "error", err, "user_id", u.ID, "plan", planCode)
		h.editUserText(ctx, b, chatID, messageID, "Не удалось создать платёж. Проверьте тариф или попробуйте позже.", h.userMainMenu())
		return
	}
	_ = p
	planTitle := planCode
	if plans, lerr := h.paymentUC.ListActivePlans(ctx); lerr == nil {
		for _, pl := range plans {
			if pl.Code == planCode {
				planTitle = pl.Name
				break
			}
		}
	}
	text := fmt.Sprintf("💳 %s\n\nПосле оплаты доступ включится автоматически (обычно до пары минут).", planTitle)
	kb := models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "💳 Оплатить", URL: payURL}},
			{{Text: "🏠 Меню", CallbackData: cbUserMenu}},
		},
	}
	h.editUserText(ctx, b, chatID, messageID, text, kb)
}

func (h *Handler) handleUserPayment(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, telegramID int64, tier domain.SubscriptionTier, days int) {
	if !h.pub.PaymentsEnabled || h.paymentUC == nil {
		h.editUserText(ctx, b, chatID, messageID, "Онлайн-оплата не настроена.", h.userMainMenu())
		return
	}
	u, err := h.userUC.GetByTelegramID(ctx, telegramID)
	if err != nil {
		h.editUserText(ctx, b, chatID, messageID, "Профиль не найден. Нажмите /start.", h.userMainMenu())
		return
	}
	p, payURL, err := h.paymentUC.CreatePayment(ctx, u.ID, tier, days)
	if err != nil {
		slog.Error("bot user payment create failed", "error", err, "user_id", u.ID)
		h.editUserText(ctx, b, chatID, messageID, "Не удалось создать платёж. Попробуйте позже.", h.userMainMenu())
		return
	}
	_ = p
	tierRu := "Базовый"
	if tier == domain.TierPremium {
		tierRu = "Премиум"
	}
	text := fmt.Sprintf("💳 %s, %d дней\n\nПосле оплаты доступ включится автоматически (обычно до пары минут).", tierRu, days)
	kb := models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "💳 Оплатить", URL: payURL}},
			{{Text: "🏠 Меню", CallbackData: cbUserMenu}},
		},
	}
	h.editUserText(ctx, b, chatID, messageID, text, kb)
}

func (h *Handler) handleUserKeys(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, telegramID int64) {
	u, err := h.userUC.GetByTelegramID(ctx, telegramID)
	if err != nil {
		h.editUserText(ctx, b, chatID, messageID, "Профиль не найден. Нажмите /start.", h.userMainMenu())
		return
	}
	subURL := h.subscriptionURL(u.SubToken)
	h.setLastSubURL(telegramID, subURL)
	sub, serr := h.subUC.GetUserSubscription(ctx, u.ID)
	status := "нет данных"
	expires := "—"
	if serr == nil && sub != nil {
		if sub.IsActive() {
			status = "активна"
			expires = sub.ExpiresAt.In(time.Local).Format("02.01.2006 15:04")
		} else {
			status = "истекла"
			expires = sub.ExpiresAt.In(time.Local).Format("02.01.2006 15:04")
		}
	}
	text := fmt.Sprintf(
		"🔑 Мои ключи / подписка\n\n"+
			"Статус: %s\n"+
			"До: %s\n\n"+
			"Ссылка подписки (импорт в Happ / v2RayTun):\n%s\n\n"+
			"Если импорт не сработал — скопируйте ссылку вручную в приложении.",
		status,
		expires,
		subURL,
	)
	kb := models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "📋 Скопировать ссылку", CallbackData: cbConnCopy}},
			{{Text: "🏠 Меню", CallbackData: cbUserMenu}},
		},
	}
	h.editUserText(ctx, b, chatID, messageID, text, kb)
}

func (h *Handler) handleUserSupport(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int) {
	h.editUserText(ctx, b, chatID, messageID, h.supportMessageText(ctx), h.userMainMenu())
}

func (h *Handler) supportMessageText(ctx context.Context) string {
	if h.settingsRepo != nil {
		s, err := h.settingsRepo.Get(ctx)
		if err == nil {
			if u := strings.TrimSpace(s.SupportURL); u != "" {
				return "Поддержка:\n" + u
			}
		}
	}
	if s := strings.TrimSpace(h.pub.SupportURL); s != "" {
		return "Поддержка:\n" + s
	}
	return "Напишите в поддержку через контакт, который выдал администратор, или оставьте заявку по ссылке из канала проекта."
}

func (h *Handler) sendPlain(ctx context.Context, b *tgbot.Bot, chatID int64, text string) {
	_, err := b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: text})
	if err != nil {
		slog.Error("bot send plain failed", "error", err)
	}
}

func (h *Handler) answerCQ(ctx context.Context, b *tgbot.Bot, cqID string, text string) {
	_, _ = b.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: cqID,
		Text:            text,
	})
}

func (h *Handler) editUserText(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, text string, kb models.InlineKeyboardMarkup) {
	_, err := b.EditMessageText(ctx, &tgbot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        text,
		ReplyMarkup: kb,
	})
	if err != nil {
		slog.Error("bot user edit failed", "error", err)
	}
}

func (h *Handler) setLastSubURL(telegramID int64, url string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.lastSubURL == nil {
		h.lastSubURL = make(map[int64]string)
	}
	h.lastSubURL[telegramID] = url
}

func (h *Handler) getLastSubURL(telegramID int64) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.lastSubURL[telegramID]
}

// onManagerCommand — /manager: панель менеджера только для admin_ids.
func (h *Handler) onManagerCommand(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	if !h.isAdmin(update) {
		h.sendPlain(ctx, b, chatID, "Доступ запрещён.")
		return
	}
	params := &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        "Панель менеджера FreeWay VPN",
		ReplyMarkup: managerMainMenu(),
	}
	if _, err := b.SendMessage(ctx, params); err != nil {
		slog.Error("bot send manager panel failed", "error", err)
	}
}
