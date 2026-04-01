package bot

import (
	"context"
	"fmt"
	"strings"

	tgbot "github.com/go-telegram/bot"
)

func (h *Handler) handleFinance(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int) {
	fin, err := h.statsUC.GetFinanceStats(ctx, "month")
	if err != nil {
		h.edit(ctx, b, chatID, messageID, "Ошибка получения финансовой статистики", backMenu())
		return
	}
	var sb strings.Builder
	sb.WriteString("💰 Финансы (месяц)\n\n")
	sb.WriteString(fmt.Sprintf("Выручка: %.0f ₽\nПлатежей: %d\nСредний чек: %.0f ₽\n\n", fin.Revenue, fin.Payments, fin.AverageCheck))
	sb.WriteString("Последние 5 платежей:\n")
	for _, p := range fin.Recent {
		sb.WriteString(fmt.Sprintf("• %s | %s | %d ₽\n", p.YookassaID, p.Tier, p.AmountRub))
	}
	h.edit(ctx, b, chatID, messageID, sb.String(), backMenu())
}

