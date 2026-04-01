package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/freeway-vpn/backend/internal/domain"
)

func (h *Handler) handleGrantChooseTier(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, adminID int64, data string) {
	parts := strings.Split(data, ":")
	if len(parts) != 2 {
		return
	}
	tgID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}
	h.mu.Lock()
	h.pendingGrantUID[adminID] = grantState{TelegramID: tgID}
	h.mu.Unlock()

	h.edit(ctx, b, chatID, messageID, "Выберите тариф, затем отправьте дней текстом (например, 30).", models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "🔵 Basic", CallbackData: "grant_tier:basic:" + parts[1]},
				{Text: "💎 Premium", CallbackData: "grant_tier:premium:" + parts[1]},
			},
			{{Text: "← Назад", CallbackData: "back"}},
		},
	})
}

func (h *Handler) handleGrantConfirm(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, adminID int64, data string) {
	parts := strings.Split(data, ":")
	if len(parts) != 3 {
		return
	}
	tier := domain.SubscriptionTier(parts[1])
	tgID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return
	}
	h.mu.Lock()
	st := h.pendingGrantUID[adminID]
	st.Tier = tier
	st.TelegramID = tgID
	h.pendingGrantUID[adminID] = st
	h.mu.Unlock()
	h.edit(ctx, b, chatID, messageID, fmt.Sprintf("Выбран тариф %s.\nТеперь отправьте количество дней текстом (например, 30).", tier), backMenu())
}

func (h *Handler) handleRevoke(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, data string) {
	parts := strings.Split(data, ":")
	if len(parts) != 2 {
		return
	}
	tgID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}
	u, err := h.userUC.GetByTelegramID(ctx, tgID)
	if err != nil {
		h.edit(ctx, b, chatID, messageID, "Пользователь не найден", backMenu())
		return
	}
	if err := h.subUC.DeactivateSubscription(ctx, u.ID); err != nil {
		h.edit(ctx, b, chatID, messageID, "Не удалось отозвать подписку", backMenu())
		return
	}
	h.edit(ctx, b, chatID, messageID, "Подписка отозвана", backMenu())
}

