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
	AppURLIOS          string
	AppURLAndroid      string
	PaymentDefaultDays int
}

// Callback-префиксы пользовательского сценария (лимит Telegram 64 байта).
const (
	cbUserMenu  = "u:m"
	cbUserConn  = "u:c"
	cbUserProf  = "u:p"
	cbUserPay   = "u:$"
	cbUserInst  = "u:i"
	cbUserRef   = "u:r"
	// legacy callbacks (старые клавиатуры в чате)
	cbUserSpeed = "u:sp"
	cbUserHelp  = "u:hp"

	cbConnIOS  = "u:ci"
	cbConnAnd  = "u:ca"
	cbConnCopy = "u:cc"

	cbBuyTierBasic   = "u:tb"
	cbBuyTierPremium = "u:tp"
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

	text := composeWelcomeText(out)
	params := &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: userMainMenu(),
	}
	if _, err := b.SendMessage(ctx, params); err != nil {
		slog.Error("bot send user menu failed", "error", err)
	}
}

func composeWelcomeText(out *usecase.TelegramStartOutcome) string {
	if out == nil {
		return "FreeWay — YouTube, Telegram и ChatGPT без лишних настроек.\n\nВыберите действие:"
	}
	var b strings.Builder
	b.WriteString("FreeWay — YouTube, Telegram и ChatGPT без лишних настроек.\n\n")
	if out.IsNewUser && out.TrialGranted {
		b.WriteString("Вам включили 3 пробных дня.\n\n")
	}
	if out.TrialSkippedByIP {
		b.WriteString("С этой сети пробный период уже выдавали. Можно сразу купить доступ в меню.\n\n")
	}
	b.WriteString("Выберите действие:")
	return b.String()
}

// userMainMenu — главное меню пользователя (закрытая альфа).
func userMainMenu() models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "🔌 Подключиться", CallbackData: cbUserConn}},
			{
				{Text: "👤 Профиль", CallbackData: cbUserProf},
				{Text: "💳 Оплатить", CallbackData: cbUserPay},
			},
			{
				{Text: "📖 Инструкция", CallbackData: cbUserInst},
				{Text: "🎁 Пригласить друга", CallbackData: cbUserRef},
			},
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
		h.editUserText(ctx, b, chatID, msg.ID, "Главное меню", userMainMenu())
	case cbUserConn:
		h.handleConnectStart(ctx, b, chatID, msg.ID, fromID)
	case cbConnIOS:
		h.showConnectDetail(ctx, b, chatID, msg.ID, fromID, "ios")
	case cbConnAnd:
		h.showConnectDetail(ctx, b, chatID, msg.ID, fromID, "android")
	case cbConnCopy:
		h.handleConnectCopy(ctx, b, chatID, msg.ID, fromID)
	case cbUserProf:
		h.handleUserProfile(ctx, b, chatID, msg.ID, fromID)
	case cbUserPay:
		h.editUserText(ctx, b, chatID, msg.ID, "Выберите тариф:", tierPickKeyboard())
	case cbBuyTierBasic:
		h.editUserText(ctx, b, chatID, msg.ID, "Базовый — выберите срок:", periodBuyKeyboard("b"))
	case cbBuyTierPremium:
		h.editUserText(ctx, b, chatID, msg.ID, "Премиум — выберите срок:", periodBuyKeyboard("p"))
	case cbUserInst:
		h.editUserText(ctx, b, chatID, msg.ID, instructionText(), userMainMenu())
	case cbUserSpeed:
		h.handleSpeedTest(ctx, b, chatID, msg.ID)
	case cbUserHelp:
		h.editUserText(ctx, b, chatID, msg.ID, instructionText(), userMainMenu())
	case cbUserRef:
		h.handleUserReferral(ctx, b, chatID, msg.ID, fromID)
	default:
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
		h.editUserText(ctx, b, chatID, messageID, "Профиль не найден. Нажмите /start.", userMainMenu())
		return
	}
	subURL := h.subscriptionURL(u.SubToken)
	h.setLastSubURL(telegramID, subURL)
	h.editUserText(ctx, b, chatID, messageID, "Выберите ваш телефон:", connectPlatformKeyboard())
}

func (h *Handler) showConnectDetail(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, telegramID int64, platform string) {
	u, err := h.userUC.GetByTelegramID(ctx, telegramID)
	if err != nil {
		h.editUserText(ctx, b, chatID, messageID, "Профиль не найден. Нажмите /start.", userMainMenu())
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
			h.editUserText(ctx, b, chatID, messageID, "Профиль не найден. Нажмите /start.", userMainMenu())
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

func instructionText() string {
	return `📖 Как подключиться

1) Нажмите «Подключиться» → выберите iPhone или Android.
2) Установите клиент (v2rayNG / V2Box) по кнопке «Скачать», если ещё не стоит.
3) Нажмите «Подключиться» в боте — откроется импорт или скопируйте ссылку подписки (кнопка «Показать ссылку»).
4) В «Оплатить» можно продлить доступ картой.

Устройства: одна ссылка подписки до 3 устройств (мягкий лимит, без жёсткой блокировки на этой фазе).

Подробнее: docs/SUPPORTED_CLIENTS.md в репозитории.`
}

func (h *Handler) handleUserProfile(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, telegramID int64) {
	u, err := h.userUC.GetByTelegramID(ctx, telegramID)
	if err != nil {
		h.editUserText(ctx, b, chatID, messageID, "Профиль не найден. Нажмите /start.", userMainMenu())
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
		connHint = "Оформите доступ в разделе «Оплатить»."
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
	h.editUserText(ctx, b, chatID, messageID, text, userMainMenu())
}

func (h *Handler) handleUserReferral(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, telegramID int64) {
	u, err := h.userUC.GetByTelegramID(ctx, telegramID)
	if err != nil {
		h.editUserText(ctx, b, chatID, messageID, "Профиль не найден. Нажмите /start.", userMainMenu())
		return
	}
	link := BuildReferralLink(h.pub.BotUsername, u.ID)
	if link == "" {
		h.editUserText(ctx, b, chatID, messageID,
			"Приглашения пока недоступны: задайте имя бота в настройках (TELEGRAM_BOT_USERNAME).",
			userMainMenu())
		return
	}
	text := fmt.Sprintf(
		"🎁 Пригласите друга\n\n%s\n\nКогда друг запустит бота по ссылке, вы получите +3 дня к подписке (если это новый пользователь и не та же сеть).",
		link,
	)
	h.editUserText(ctx, b, chatID, messageID, text, userMainMenu())
}

func (h *Handler) handleUserPayment(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, telegramID int64, tier domain.SubscriptionTier, days int) {
	if h.paymentUC == nil {
		h.editUserText(ctx, b, chatID, messageID, "Оплата временно недоступна. Попробуйте позже.", userMainMenu())
		return
	}
	u, err := h.userUC.GetByTelegramID(ctx, telegramID)
	if err != nil {
		h.editUserText(ctx, b, chatID, messageID, "Профиль не найден. Нажмите /start.", userMainMenu())
		return
	}
	p, payURL, err := h.paymentUC.CreatePayment(ctx, u.ID, tier, days)
	if err != nil {
		slog.Error("bot user payment create failed", "error", err, "user_id", u.ID)
		h.editUserText(ctx, b, chatID, messageID, "Не удалось создать платёж. Попробуйте позже.", userMainMenu())
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
		h.sendPlain(ctx, b, chatID, "Access denied")
		return
	}
	params := &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        "FreeWay Manager Panel",
		ReplyMarkup: managerMainMenu(),
	}
	if _, err := b.SendMessage(ctx, params); err != nil {
		slog.Error("bot send manager panel failed", "error", err)
	}
}
