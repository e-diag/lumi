package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	apimw "github.com/freeway-vpn/backend/internal/handler/api/middleware"
	"github.com/freeway-vpn/backend/internal/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// PaymentHandler — payment-endpoints (Фаза 2).
type PaymentHandler struct {
	paymentUC usecase.PaymentUseCase
	subUC     usecase.SubscriptionUseCase
}

// NewPaymentHandler создаёт PaymentHandler.
func NewPaymentHandler(paymentUC usecase.PaymentUseCase) *PaymentHandler {
	return &PaymentHandler{paymentUC: paymentUC}
}

func NewPaymentHandlerWithSubscription(paymentUC usecase.PaymentUseCase, subUC usecase.SubscriptionUseCase) *PaymentHandler {
	return &PaymentHandler{paymentUC: paymentUC, subUC: subUC}
}

type createPaymentRequest struct {
	Tier string `json:"tier"`
	Days int    `json:"days"`
}

// CreatePayment: POST /api/v1/payments
func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromCtx(w, r)
	if !ok {
		return
	}

	var req createPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	p, url, err := h.paymentUC.CreatePayment(r.Context(), userID, domain.SubscriptionTier(req.Tier), req.Days)
	if err != nil {
		slog.Error("payment create failed", "error", err, "user_id", userID)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot create payment"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"payment_url": url,
		"payment_id":  p.ID.String(),
	})
}

// GetPaymentStatus: GET /api/v1/payments/{id}/status
func (h *PaymentHandler) GetPaymentStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromCtx(w, r)
	if !ok {
		return
	}

	idStr := chi.URLParam(r, "id")
	paymentID, err := uuid.Parse(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	// Обновляем статус “лениво” при запросе.
	if err := h.paymentUC.CheckAndUpdatePayment(r.Context(), paymentID); err != nil {
		slog.Error("payment status check failed", "error", err, "payment_id", paymentID)
	}

	p, err := h.paymentUC.GetByID(r.Context(), paymentID)
	if err != nil {
		slog.Error("payment get failed", "error", err, "payment_id", paymentID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if p.UserID != userID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	resp := map[string]any{
		"status": p.Status,
	}
	if h.subUC != nil {
		if sub, err := h.subUC.GetUserSubscription(r.Context(), userID); err == nil && sub != nil {
			resp["active_until"] = sub.ExpiresAt.UTC().Format(time.RFC3339)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func getUserIDFromCtx(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw, _ := r.Context().Value(apimw.ContextKeyUserID).(string)
	if raw == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return uuid.Nil, false
	}
	return id, true
}
