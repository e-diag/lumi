package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/freeway-vpn/backend/internal/usecase"
)

type RoutingHandler struct {
	routingUC usecase.RoutingUseCase

	mu        sync.RWMutex
	cachedAt  time.Time
	cachedTTL time.Duration
	cached    *usecase.RoutingLists
}

func NewRoutingHandler(routingUC usecase.RoutingUseCase) *RoutingHandler {
	return &RoutingHandler{
		routingUC: routingUC,
		cachedTTL: time.Hour,
	}
}

// GET /api/v1/routing/lists
func (h *RoutingHandler) GetLists(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	if h.cached != nil && time.Since(h.cachedAt) < h.cachedTTL {
		list := h.cached
		h.mu.RUnlock()
		respondRoutingJSON(w, list)
		return
	}
	h.mu.RUnlock()

	list, err := h.routingUC.GetLists(r.Context())
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	h.mu.Lock()
	h.cached = list
	h.cachedAt = time.Now()
	h.mu.Unlock()

	respondRoutingJSON(w, list)
}

func (h *RoutingHandler) InvalidateCache() {
	h.mu.Lock()
	h.cached = nil
	h.cachedAt = time.Time{}
	h.mu.Unlock()
}

func respondRoutingJSON(w http.ResponseWriter, list *usecase.RoutingLists) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"version":            list.Version,
		"proxy_eu":           list.ProxyEU,
		"proxy_usa":          list.ProxyUSA,
		"direct":             list.Direct,
		"direct_strict_mode": list.DirectStrictMode,
		"meta":               list.Meta,
	})
}
