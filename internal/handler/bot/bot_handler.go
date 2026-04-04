package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/repository"
	"github.com/freeway-vpn/backend/internal/usecase"
)

type Handler struct {
	statsUC    usecase.StatsUseCase
	userUC     usecase.UserUseCase
	subUC      usecase.SubscriptionUseCase
	paymentUC  usecase.PaymentUseCase
	nodeUC     usecase.NodeUseCase
	routingUC  usecase.RoutingUseCase
	botUserUC  usecase.TelegramBotUserUseCase
	configUC      usecase.ConfigUseCase
	settingsRepo  repository.ProductSettingsRepository
	pub           PublicSettings
	startRL    *telegramWindowLimiter
	callbackRL *telegramWindowLimiter
	adminIDs   map[int64]struct{}

	// Простое in-memory состояние FSM для рассылок/выдачи подписок.
	mu              sync.Mutex
	broadcastDraft  map[int64]string
	pendingGrantUID map[int64]grantState
	pendingRouting  map[int64]string
	lastSubURL      map[int64]string
}

type grantState struct {
	TelegramID int64
	Tier       domain.SubscriptionTier
}

func NewHandler(
	statsUC usecase.StatsUseCase,
	userUC usecase.UserUseCase,
	subUC usecase.SubscriptionUseCase,
	paymentUC usecase.PaymentUseCase,
	nodeUC usecase.NodeUseCase,
	routingUC usecase.RoutingUseCase,
	botUserUC usecase.TelegramBotUserUseCase,
	configUC usecase.ConfigUseCase,
	pub PublicSettings,
	settingsRepo repository.ProductSettingsRepository,
	adminIDs []int64,
) *Handler {
	m := make(map[int64]struct{}, len(adminIDs))
	for _, id := range adminIDs {
		m[id] = struct{}{}
	}
	if pub.PaymentDefaultDays <= 0 {
		pub.PaymentDefaultDays = 30
	}
	return &Handler{
		statsUC:         statsUC,
		userUC:          userUC,
		subUC:           subUC,
		paymentUC:       paymentUC,
		nodeUC:          nodeUC,
		routingUC:       routingUC,
		botUserUC:       botUserUC,
		configUC:        configUC,
		settingsRepo:    settingsRepo,
		pub:             pub,
		startRL:         newTelegramWindowLimiter(),
		callbackRL:      newTelegramWindowLimiter(),
		adminIDs:        m,
		broadcastDraft:  map[int64]string{},
		pendingGrantUID: map[int64]grantState{},
		pendingRouting:  map[int64]string{},
		lastSubURL:      map[int64]string{},
	}
}

func (h *Handler) Register(b *tgbot.Bot) {
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/start", tgbot.MatchTypePrefix, h.onPublicStart)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/manager", tgbot.MatchTypePrefix, h.onManagerCommand)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/user", tgbot.MatchTypePrefix, h.onUserCommand)
	b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, "", tgbot.MatchTypePrefix, h.onCallback)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "", tgbot.MatchTypePrefix, h.onTextMessage)
}

func (h *Handler) isAdmin(update *models.Update) bool {
	if update == nil || update.Message == nil || update.Message.From == nil {
		if update != nil && update.CallbackQuery != nil {
			_, ok := h.adminIDs[update.CallbackQuery.From.ID]
			return ok
		}
		return false
	}
	_, ok := h.adminIDs[update.Message.From.ID]
	return ok
}

func managerMainMenu() models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "📊 Статистика", CallbackData: "stats"},
				{Text: "🌐 Ноды", CallbackData: "nodes"},
			},
			{
				{Text: "👥 Пользователи", CallbackData: "users"},
				{Text: "💰 Финансы", CallbackData: "finance"},
			},
			{
				{Text: "📢 Рассылка", CallbackData: "broadcast"},
				{Text: "⚙️ Настройки", CallbackData: "settings"},
			},
			{
				{Text: "🗺 Роутинг", CallbackData: "routing_main"},
			},
		},
	}
}

func backMenu() models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "← Назад", CallbackData: "back"}},
		},
	}
}

func (h *Handler) onCallback(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if update.CallbackQuery == nil || update.CallbackQuery.Message.Message == nil {
		return
	}
	data := update.CallbackQuery.Data

	// Пользовательское меню (не требует прав менеджера).
	if strings.HasPrefix(data, "u:") {
		h.onUserCallback(ctx, b, update)
		return
	}

	if !h.isAdmin(update) {
		_, _ = b.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "Недостаточно прав",
		})
		return
	}

	msg := update.CallbackQuery.Message.Message
	chatID := msg.Chat.ID

	switch {
	case data == "back":
		h.edit(ctx, b, chatID, msg.ID, "FreeWay Manager Panel", managerMainMenu())
	case data == "stats" || data == "stats_refresh":
		h.handleStats(ctx, b, chatID, msg.ID)
	case data == "nodes" || data == "nodes_check":
		h.handleNodes(ctx, b, chatID, msg.ID)
	case data == "finance":
		h.handleFinance(ctx, b, chatID, msg.ID)
	case data == "broadcast":
		h.mu.Lock()
		h.broadcastDraft[update.CallbackQuery.From.ID] = ""
		h.mu.Unlock()
		h.handleBroadcastPrompt(ctx, b, chatID, msg.ID)
	case data == "broadcast_send":
		h.handleBroadcastSend(ctx, b, update.CallbackQuery.From.ID, chatID, msg.ID)
	case data == "broadcast_cancel":
		h.handleBroadcastCancel(ctx, b, update.CallbackQuery.From.ID, chatID, msg.ID)
	case data == "users":
		h.edit(ctx, b, chatID, msg.ID, "Используйте команду /user <telegram_id>", backMenu())
	case data == "settings":
		h.edit(ctx, b, chatID, msg.ID, "⚙️ Настройки\nПока нет настраиваемых параметров.", backMenu())
	case strings.HasPrefix(data, "grant:"):
		h.handleGrantChooseTier(ctx, b, chatID, msg.ID, update.CallbackQuery.From.ID, data)
	case strings.HasPrefix(data, "grant_tier:"):
		h.handleGrantConfirm(ctx, b, chatID, msg.ID, update.CallbackQuery.From.ID, data)
	case strings.HasPrefix(data, "revoke:"):
		h.handleRevoke(ctx, b, chatID, msg.ID, data)
	case data == "routing_main" || data == "routing_refresh":
		h.handleRoutingMain(ctx, b, chatID, msg.ID)
	case data == "routing_add_prompt":
		h.handleRoutingAddPrompt(ctx, b, chatID, msg.ID, update.CallbackQuery.From.ID)
	case strings.HasPrefix(data, "routing_add_action:"):
		h.handleRoutingAddAction(ctx, b, chatID, msg.ID, update.CallbackQuery.From.ID, data)
	case data == "routing_update_now":
		h.handleRoutingUpdateNow(ctx, b, chatID, msg.ID)
	}

	_, _ = b.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
}

func (h *Handler) onTextMessage(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if !h.isAdmin(update) || update.Message == nil {
		return
	}
	adminID := update.Message.From.ID
	text := strings.TrimSpace(update.Message.Text)
	if strings.HasPrefix(text, "/") {
		return
	}

	h.mu.Lock()
	_, waitingBroadcast := h.broadcastDraft[adminID]
	grantState, waitingGrant := h.pendingGrantUID[adminID]
	_, waitingRouting := h.pendingRouting[adminID]
	h.mu.Unlock()

	if waitingBroadcast {
		h.mu.Lock()
		h.broadcastDraft[adminID] = text
		h.mu.Unlock()
		h.send(ctx, b, update.Message.Chat.ID, "Превью:\n\n"+text, models.InlineKeyboardMarkup{
			InlineKeyboard: broadcastPreviewKeyboard().InlineKeyboard,
		})
		return
	}

	if waitingGrant {
		days, err := strconv.Atoi(text)
		if err != nil || days <= 0 {
			h.send(ctx, b, update.Message.Chat.ID, "Введите число дней (например, 30).", backMenu())
			return
		}
		u, err := h.userUC.GetByTelegramID(ctx, grantState.TelegramID)
		if err != nil {
			h.send(ctx, b, update.Message.Chat.ID, "Пользователь не найден", backMenu())
			return
		}
		_, err = h.subUC.ActivateSubscription(ctx, u.ID, grantState.Tier, days)
		if err != nil {
			h.send(ctx, b, update.Message.Chat.ID, "Не удалось выдать подписку", backMenu())
			return
		}
		h.mu.Lock()
		delete(h.pendingGrantUID, adminID)
		h.mu.Unlock()
		h.send(ctx, b, update.Message.Chat.ID, fmt.Sprintf("Подписка %s выдана на %d дней", grantState.Tier, days), backMenu())
	}
	if waitingRouting {
		domainValue := strings.ToLower(strings.TrimSpace(text))
		h.mu.Lock()
		h.pendingRouting[adminID] = domainValue
		h.mu.Unlock()
		h.send(ctx, b, update.Message.Chat.ID, "Выберите действие для домена "+domainValue, models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "🇪🇺 proxy_eu", CallbackData: "routing_add_action:proxy_eu"},
					{Text: "🇺🇸 proxy_usa", CallbackData: "routing_add_action:proxy_usa"},
				},
				{
					{Text: "↩️ direct", CallbackData: "routing_add_action:direct"},
					{Text: "❌ Отмена", CallbackData: "back"},
				},
			},
		})
	}
}

func (h *Handler) onUserCommand(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	if !h.isAdmin(update) || update.Message == nil {
		return
	}
	parts := strings.Fields(update.Message.Text)
	if len(parts) != 2 {
		h.send(ctx, b, update.Message.Chat.ID, "Использование: /user <telegram_id>", backMenu())
		return
	}
	tgID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		h.send(ctx, b, update.Message.Chat.ID, "Неверный telegram_id", backMenu())
		return
	}
	u, err := h.userUC.GetByTelegramID(ctx, tgID)
	if err != nil {
		h.send(ctx, b, update.Message.Chat.ID, "Пользователь не найден", backMenu())
		return
	}
	sub, _ := h.subUC.GetUserSubscription(ctx, u.ID)
	payments, _ := h.paymentUC.ListByUser(ctx, u.ID)
	totalSpent := 0
	for _, p := range payments {
		if p.Status == domain.PaymentSucceeded {
			totalSpent += p.AmountRub
		}
	}
	tier := "🆓 Free"
	activeUntil := "—"
	daysLeft := 0
	if sub != nil && sub.IsActive() {
		switch sub.Tier {
		case domain.TierBasic:
			tier = "🔵 Basic"
		case domain.TierPremium:
			tier = "💎 Premium"
		}
		activeUntil = sub.ExpiresAt.Format("02 Jan 2006")
		daysLeft = sub.DaysLeft()
	}

	text := fmt.Sprintf("👤 @%s (%d)\nТариф: %s\nАктивна до: %s | Осталось: %d дней\nПлатежей: %d / Потрачено: %d ₽",
		u.Username, u.TelegramID, tier, activeUntil, daysLeft, len(payments), totalSpent)

	h.send(ctx, b, update.Message.Chat.ID, text, models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "✅ Выдать подписку", CallbackData: "grant:" + strconv.FormatInt(tgID, 10)},
				{Text: "❌ Отозвать", CallbackData: "revoke:" + strconv.FormatInt(tgID, 10)},
			},
			{{Text: "← Назад", CallbackData: "back"}},
		},
	})
}

func (h *Handler) send(ctx context.Context, b *tgbot.Bot, chatID int64, text string, keyboard models.InlineKeyboardMarkup) {
	_, err := b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: keyboard,
	})
	if err != nil {
		slog.Error("bot send failed", "error", err)
	}
}

func (h *Handler) edit(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, text string, keyboard models.InlineKeyboardMarkup) {
	_, err := b.EditMessageText(ctx, &tgbot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        text,
		ReplyMarkup: keyboard,
	})
	if err != nil {
		slog.Error("bot edit failed", "error", err)
	}
}
