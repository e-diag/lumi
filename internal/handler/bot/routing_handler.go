package bot

import (
	"context"
	"fmt"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/freeway-vpn/backend/internal/domain"
)

func (h *Handler) handleRoutingMain(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int) {
	if h.routingUC == nil {
		h.edit(ctx, b, chatID, messageID, "Routing usecase недоступен", backMenu())
		return
	}
	l, err := h.routingUC.GetLists(ctx)
	if err != nil {
		h.edit(ctx, b, chatID, messageID, "Ошибка загрузки routing-списков", backMenu())
		return
	}
	total := len(l.ProxyEU) + len(l.ProxyUSA) + len(l.Direct)
	text := fmt.Sprintf("🗺 Маршрутизация\n\n📊 Доменов в базе: %d\n📅 Обновлено: %s\n\n🇪🇺 proxy_eu: %d\n🇺🇸 proxy_usa: %d\n↩️ direct: %d",
		total, l.Version, len(l.ProxyEU), len(l.ProxyUSA), len(l.Direct))
	h.edit(ctx, b, chatID, messageID, text, models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "➕ Добавить домен", CallbackData: "routing_add_prompt"},
				{Text: "🔄 Обновить сейчас", CallbackData: "routing_update_now"},
			},
			{{Text: "← Назад", CallbackData: "back"}},
		},
	})
}

func (h *Handler) handleRoutingAddPrompt(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, adminID int64) {
	h.mu.Lock()
	h.pendingRouting[adminID] = ""
	h.mu.Unlock()
	h.edit(ctx, b, chatID, messageID, "Отправьте домен текстом (например, example.com)", backMenu())
}

func (h *Handler) handleRoutingUpdateNow(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int) {
	if err := h.routingUC.UpdateFromAntifilter(ctx); err != nil {
		h.edit(ctx, b, chatID, messageID, "Обновление не удалось", backMenu())
		return
	}
	h.edit(ctx, b, chatID, messageID, "Списки успешно обновлены", backMenu())
}

func (h *Handler) handleRoutingAddAction(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int, adminID int64, data string) {
	h.mu.Lock()
	domainValue := h.pendingRouting[adminID]
	delete(h.pendingRouting, adminID)
	h.mu.Unlock()
	if domainValue == "" {
		h.edit(ctx, b, chatID, messageID, "Сначала введите домен", backMenu())
		return
	}
	actionRaw := strings.TrimPrefix(data, "routing_add_action:")
	action := domain.RouteAction(actionRaw)
	if action != domain.ActionProxyEU && action != domain.ActionProxyUSA && action != domain.ActionDirect {
		h.edit(ctx, b, chatID, messageID, "Неверное действие", backMenu())
		return
	}
	if err := h.routingUC.AddDomain(ctx, domainValue, action); err != nil {
		h.edit(ctx, b, chatID, messageID, "Ошибка добавления домена", backMenu())
		return
	}
	h.edit(ctx, b, chatID, messageID, fmt.Sprintf("Домен %s добавлен в %s", domainValue, action), backMenu())
}
