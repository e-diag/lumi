package middleware

import (
	"net/http"
	"sync"
	"time"
)

const (
	authRatePerMinute = 12.0 // ~5 сек между попытками в среднем
	authBurst         = 6.0  // короткая серия при отладке WebApp
)

type authRLEntry struct {
	tokens float64
	last   time.Time
}

// TelegramAuthRateLimit ограничивает POST /api/v1/auth/tg по IP (перебор init_data).
func TelegramAuthRateLimit(next http.Handler) http.Handler {
	var mu sync.Mutex
	m := make(map[string]*authRLEntry)
	perSec := authRatePerMinute / 60.0
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := ClientIP(r)
		now := time.Now()
		mu.Lock()
		e, ok := m[ip]
		if !ok {
			e = &authRLEntry{tokens: authBurst, last: now}
			m[ip] = e
		}
		elapsed := now.Sub(e.last).Seconds()
		e.tokens = minFloat(authBurst, e.tokens+elapsed*perSec)
		e.last = now
		if e.tokens < 1 {
			mu.Unlock()
			w.Header().Set("Retry-After", "10")
			http.Error(w, `{"error":"rate limit"}`, http.StatusTooManyRequests)
			return
		}
		e.tokens -= 1
		if len(m) > 200000 {
			m = make(map[string]*authRLEntry)
		}
		mu.Unlock()
		next.ServeHTTP(w, r)
	})
}
