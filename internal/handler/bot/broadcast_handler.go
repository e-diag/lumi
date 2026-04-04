package bot

import (
	"context"
	"fmt"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Handler) handleBroadcastPrompt(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int) {
	// Ждём следующее текстовое сообщение от администратора как текст рассылки.
	// ID админа сохраняется в callback handler.
	h.edit(ctx, b, chatID, messageID, "Введите текст рассылки следующим сообщением.", backMenu())
}

func (h *Handler) handleBroadcastSend(ctx context.Context, b *tgbot.Bot, adminID int64, chatID int64, messageID int) {
	h.mu.Lock()
	text := h.broadcastDraft[adminID]
	delete(h.broadcastDraft, adminID)
	h.mu.Unlock()
	if text == "" {
		h.edit(ctx, b, chatID, messageID, "Нет текста для рассылки", backMenu())
		return
	}

	const pageSize = 500
	page := 1
	sent := 0
	errorsCount := 0
	for {
		users, total, err := h.userUC.List(ctx, "", page, pageSize)
		if err != nil {
			h.edit(ctx, b, chatID, messageID, "Ошибка получения пользователей", backMenu())
			return
		}
		for _, u := range users {
			_, err := b.SendMessage(ctx, &tgbot.SendMessageParams{
				ChatID: u.TelegramID,
				Text:   text,
			})
			if err != nil {
				errorsCount++
			} else {
				sent++
			}
		}
		if int64(page*pageSize) >= total {
			break
		}
		page++
	}
	h.edit(ctx, b, chatID, messageID, fmt.Sprintf("Отправлено: %d, Ошибок: %d", sent, errorsCount), backMenu())
}

func (h *Handler) handleBroadcastCancel(ctx context.Context, b *tgbot.Bot, adminID int64, chatID int64, messageID int) {
	h.mu.Lock()
	delete(h.broadcastDraft, adminID)
	h.mu.Unlock()
	h.edit(ctx, b, chatID, messageID, "Рассылка отменена", backMenu())
}

func broadcastPreviewKeyboard() models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "✅ Отправить", CallbackData: "broadcast_send"},
				{Text: "❌ Отмена", CallbackData: "broadcast_cancel"},
			},
		},
	}
}
