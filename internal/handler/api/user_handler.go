package api

import (
	"errors"
	"net/http"

	"github.com/freeway-vpn/backend/internal/domain"
	apimw "github.com/freeway-vpn/backend/internal/handler/api/middleware"
	"github.com/freeway-vpn/backend/internal/usecase"
)

// UserHandler — профиль и подписка для клиента с JWT.
type UserHandler struct {
	userUC usecase.UserUseCase
	subUC  usecase.SubscriptionUseCase
}

// NewUserHandler создаёт UserHandler.
func NewUserHandler(userUC usecase.UserUseCase, subUC usecase.SubscriptionUseCase) *UserHandler {
	return &UserHandler{userUC: userUC, subUC: subUC}
}

// GetMe обрабатывает GET /api/v1/users/me.
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, err := apimw.UserIDFromContext(r.Context())
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	u, err := h.userUC.GetByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":           u.ID.String(),
		"telegram_id":  u.TelegramID,
		"username":     u.Username,
		"sub_token":    u.SubToken,
		"device_limit": u.DeviceLimit,
		"force_cdn":    u.ForceCDN,
	})
}

// GetSubscription обрабатывает GET /api/v1/users/me/subscription.
func (h *UserHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	userID, err := apimw.UserIDFromContext(r.Context())
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	sub, err := h.subUC.GetUserSubscription(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			writeJSON(w, http.StatusOK, map[string]any{
				"tier":       string(domain.TierFree),
				"active":     false,
				"expires_at": nil,
				"days_left":  0,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tier":       string(sub.Tier),
		"active":     sub.IsActive(),
		"expires_at": sub.ExpiresAt,
		"days_left":  sub.DaysLeft(),
	})
}
