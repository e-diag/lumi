package bot

import (
	"sync"
	"time"

	"github.com/freeway-vpn/backend/internal/usecase"
	"github.com/go-telegram/bot/models"
)

// telegramWindowLimiter — скользящее окно событий на один Telegram ID.
type telegramWindowLimiter struct {
	mu sync.Mutex
	m  map[int64][]time.Time
}

func newTelegramWindowLimiter() *telegramWindowLimiter {
	return &telegramWindowLimiter{m: make(map[int64][]time.Time)}
}

// Allow возвращает true, если запрос разрешён (не больше max событий за window).
func (r *telegramWindowLimiter) Allow(telegramID int64, max int, window time.Duration) bool {
	if max <= 0 {
		return true
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)
	prev := r.m[telegramID]
	var kept []time.Time
	for _, t := range prev {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= max {
		return false
	}
	kept = append(kept, now)
	r.m[telegramID] = kept
	return true
}

// ExtractTelegramClientMeta извлекает IP/UA, если они доступны.
// У long polling Bot API IP пользователя в update не передаётся — поля будут пустыми;
// при работе через webhook за reverse-proxy можно расширить разбор заголовков на стороне приёма update.
func ExtractTelegramClientMeta(update *models.Update) usecase.TelegramClientMeta {
	_ = update
	return usecase.TelegramClientMeta{}
}
