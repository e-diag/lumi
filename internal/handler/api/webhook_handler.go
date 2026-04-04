package api

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/netip"

	"github.com/freeway-vpn/backend/internal/logredact"
	"github.com/freeway-vpn/backend/internal/usecase"
)

// WebhookHandler обрабатывает публичные вебхуки.
type WebhookHandler struct {
	paymentUC usecase.PaymentUseCase
}

func NewWebhookHandler(paymentUC usecase.PaymentUseCase) *WebhookHandler {
	return &WebhookHandler{paymentUC: paymentUC}
}

var yookassaAllowed = mustParseCIDRs([]string{
	"185.71.76.0/27",
	"185.71.77.0/27",
	"77.75.153.0/25",
	"77.75.154.128/25",
})

// YookassaWebhook: POST /api/v1/payments/webhook
func (h *WebhookHandler) YookassaWebhook(w http.ResponseWriter, r *http.Request) {
	ip := net.ParseIP(remoteIP(r))
	if ip == nil {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	if !isAllowedIP(addr, yookassaAllowed) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	var event usecase.WebhookEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}

	if err := h.paymentUC.HandleWebhook(r.Context(), event); err != nil {
		slog.Error("yookassa webhook failed", "error", err, "provider_id", logredact.ProviderPaymentIDForLog(event.Object.ID))
		// 200 OK, чтобы ЮKassa не долбила бесконечно из-за временных ошибок в нашей бизнес-логике.
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func remoteIP(r *http.Request) string {
	// chi middleware RealIP уже проставляет r.RemoteAddr.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func mustParseCIDRs(cidrs []string) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(cidrs))
	for _, s := range cidrs {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			// Неверный CIDR в константе — просто пропускаем.
			continue
		}
		out = append(out, p)
	}
	return out
}

func isAllowedIP(ip netip.Addr, prefixes []netip.Prefix) bool {
	for _, p := range prefixes {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}
