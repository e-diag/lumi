package bot

import (
	"context"
	"fmt"

	tgbot "github.com/go-telegram/bot"
)

func (h *Handler) handleNodes(ctx context.Context, b *tgbot.Bot, chatID int64, messageID int) {
	nodes, err := h.nodeUC.GetAllNodes(ctx)
	if err != nil {
		h.edit(ctx, b, chatID, messageID, "Ошибка получения нод", backMenu())
		return
	}
	text := "🌐 Ноды\n\n"
	for _, n := range nodes {
		state := "🔴 offline"
		if n.Active {
			state = "🟢 online"
		}
		text += fmt.Sprintf("%s (%s)\n%s, latency: %dms\n\n", n.Name, n.Region, state, n.LatencyMs)
	}
	h.edit(ctx, b, chatID, messageID, text, twoButtons("🔄 Проверить", "nodes_check", "← Назад", "back"))
}
