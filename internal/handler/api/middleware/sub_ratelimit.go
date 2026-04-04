package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	subRatePerSecond = 1.0  // ~60 запросов в минуту в среднем
	subBurst         = 18.0 // кратковременный всплеск (импорт подписки в нескольких клиентах)
)

type subRLEntry struct {
	tokens float64
	last   time.Time
}

// SubscriptionRateLimit защищает GET /sub/{token} от перебора токенов и перегрузки генерации.
func SubscriptionRateLimit(next http.Handler) http.Handler {
	var mu sync.Mutex
	m := make(map[string]*subRLEntry)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := subscriptionClientIP(r)
		now := time.Now()
		mu.Lock()
		e, ok := m[ip]
		if !ok {
			e = &subRLEntry{tokens: subBurst, last: now}
			m[ip] = e
		}
		elapsed := now.Sub(e.last).Seconds()
		e.tokens = minFloat(subBurst, e.tokens+elapsed*subRatePerSecond)
		e.last = now
		if e.tokens < 1 {
			mu.Unlock()
			w.Header().Set("Retry-After", "5")
			http.Error(w, `{"error":"rate limit"}`, http.StatusTooManyRequests)
			return
		}
		e.tokens -= 1
		if len(m) > 200000 {
			m = make(map[string]*subRLEntry)
		}
		mu.Unlock()
		next.ServeHTTP(w, r)
	})
}

// ClientIP возвращает IP клиента с учётом заголовков прокси (как для rate limit).
func ClientIP(r *http.Request) string {
	return subscriptionClientIP(r)
}

func subscriptionClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			return strings.TrimSpace(parts[0])
		}
	}
	if xr := r.Header.Get("X-Real-Ip"); xr != "" {
		return strings.TrimSpace(xr)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
