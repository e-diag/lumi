package bot

import (
	"context"
	"fmt"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Handler) handleStats(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int) {
	stats, err := h.statsUC.GetDashboardStats(ctx)
	if err != nil {
		h.edit(ctx, b, chatID, messageID, "Ошибка получения статистики", backMenu())
		return
	}
	text := fmt.Sprintf(
		"📊 FreeWay — статистика\n\n"+
			"👥 Всего: %d\n"+
			"├ 🆓 Free: %d\n"+
			"├ 🔵 Basic: %d\n"+
			"└ 💎 Premium: %d\n\n"+
			"💰 Сегодня: %.0f ₽ (%d платежей)\n"+
			"💰 Месяц: %.0f ₽\n\n",
		stats.TotalUsers, stats.FreeUsers, stats.BasicUsers, stats.PremiumUsers,
		stats.RevenueToday, stats.PaymentsToday, stats.RevenueMonth,
	)
	for _, n := range stats.Nodes {
		state := "🔴"
		if n.IsOnline {
			state = "🟢"
		}
		text += fmt.Sprintf("🌐 %s: %s %dмс / %d онлайн\n", n.Name, state, n.LatencyMs, n.Online)
	}
	h.edit(ctx, b, chatID, messageID, text, twoButtons("🔄 Обновить", "stats_refresh", "← Назад", "back"))
}

func twoButtons(aText, aCb, bText, bCb string) models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: aText, CallbackData: aCb},
				{Text: bText, CallbackData: bCb},
			},
		},
	}
}

