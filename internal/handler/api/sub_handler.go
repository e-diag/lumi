// Пакет api содержит HTTP-обработчики REST API.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/freeway-vpn/backend/internal/domain"
	apimw "github.com/freeway-vpn/backend/internal/handler/api/middleware"
	"github.com/freeway-vpn/backend/internal/usecase"
	"github.com/go-chi/chi/v5"
)

// SubHandler обрабатывает запросы к subscription endpoint.
type SubHandler struct {
	userUC      usecase.UserUseCase
	configUC    usecase.ConfigUseCase
	probe       usecase.AccessProbeUseCase
	profileName string // название для заголовка profile-title
}

// NewSubHandler создаёт SubHandler.
// probe может быть nil — мягкий учёт обращений отключён.
func NewSubHandler(userUC usecase.UserUseCase, configUC usecase.ConfigUseCase, profileName string, probe usecase.AccessProbeUseCase) *SubHandler {
	return &SubHandler{
		userUC:      userUC,
		configUC:    configUC,
		probe:       probe,
		profileName: profileName,
	}
}

// GetSubscription обрабатывает GET /sub/{token}.
// Возвращает base64-текст подписки (обычно список vless:// и др.) для Happ / v2RayTun и совместимых клиентов.
func (h *SubHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		http.Error(w, `{"error":"token required"}`, http.StatusBadRequest)
		return
	}

	user, err := h.userUC.GetBySubToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		slog.Error("sub handler: get user by token", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	encoded, err := h.configUC.GenerateSubscription(r.Context(), user.ID)
	if err != nil {
		slog.Error("sub handler: generate subscription", "user_id", user.ID, "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	if h.probe != nil {
		ip := apimw.ClientIP(r)
		if err := h.probe.RecordSubscriptionFetch(r.Context(), user.ID, ip, r.UserAgent()); err != nil {
			slog.Warn("sub handler: access probe", "user_id", user.ID, "error", err)
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("profile-title", h.profileName)
	w.Header().Set("profile-update-interval", "24")
	w.Header().Set("subscription-userinfo", "upload=0; download=0; total=0; expire=0")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(encoded))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
