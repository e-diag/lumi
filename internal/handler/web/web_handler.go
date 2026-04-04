package web

import (
	"crypto/subtle"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/repository"
	"github.com/freeway-vpn/backend/internal/usecase"
)

type WebHandler struct {
	statsUC      usecase.StatsUseCase
	userUC       usecase.UserUseCase
	subUC        usecase.SubscriptionUseCase
	nodeUC       usecase.NodeUseCase
	paymentUC    usecase.PaymentUseCase
	routingUC    usecase.RoutingUseCase
	planRepo     repository.PlanRepository
	settingsRepo repository.ProductSettingsRepository
	serverRepo   repository.VPNServerRepository
	adminToken   string
	templates    *template.Template
}

func NewWebHandler(
	statsUC usecase.StatsUseCase,
	userUC usecase.UserUseCase,
	subUC usecase.SubscriptionUseCase,
	nodeUC usecase.NodeUseCase,
	paymentUC usecase.PaymentUseCase,
	routingUC usecase.RoutingUseCase,
	planRepo repository.PlanRepository,
	settingsRepo repository.ProductSettingsRepository,
	serverRepo repository.VPNServerRepository,
	adminToken string,
	templateDir string,
) (*WebHandler, error) {
	t, err := template.ParseGlob(filepath.Join(templateDir, "*.html"))
	if err != nil {
		return nil, fmt.Errorf("web: parse templates: %w", err)
	}
	return &WebHandler{
		statsUC:      statsUC,
		userUC:       userUC,
		subUC:        subUC,
		nodeUC:       nodeUC,
		paymentUC:    paymentUC,
		routingUC:    routingUC,
		planRepo:     planRepo,
		settingsRepo: settingsRepo,
		serverRepo:   serverRepo,
		adminToken:   adminToken,
		templates:    t,
	}, nil
}

func (h *WebHandler) RegisterRoutes(r chi.Router) {
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/admin/login", h.LoginPage)
	r.Post("/admin/login", h.LoginPost)

	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Get("/admin/", h.DashboardPage)
		r.Get("/admin/users", h.UsersPage)
		r.Get("/admin/nodes", h.NodesPage)
		r.Get("/admin/payments", h.PaymentsPage)
		r.Get("/admin/plans", h.PlansPage)
		r.Get("/admin/settings", h.SettingsPage)
		r.Get("/admin/servers", h.ServersPage)
		r.Get("/admin/routing", h.RoutingPage)

		r.Get("/admin/api/stats", h.StatsFragment)
		r.Get("/admin/api/users", h.UsersFragment)
		r.Post("/admin/api/users/{id}/grant", h.GrantUser)
		r.Post("/admin/api/users/{id}/revoke", h.RevokeUser)
		r.Get("/admin/api/nodes", h.NodesFragment)
		r.Post("/admin/api/nodes/{id}/check", h.NodeCheckFragment)
		r.Get("/admin/api/payments", h.PaymentsFragment)
		r.Get("/admin/api/routing", h.RoutingFragment)
		r.Post("/admin/api/routing/update", h.RoutingUpdate)
		r.Post("/admin/api/routing/add", h.RoutingAdd)
		r.Post("/admin/api/routing/delete", h.RoutingDelete)

		r.Get("/admin/api/plans", h.PlansFragment)
		r.Post("/admin/api/plans/{id}/toggle", h.PlanToggle)
		r.Post("/admin/api/plans/{id}/price", h.PlanPrice)
		r.Post("/admin/api/settings", h.SettingsSave)
		r.Get("/admin/api/servers", h.ServersFragment)
		r.Post("/admin/api/servers", h.ServerAdd)
		r.Post("/admin/api/servers/{id}/toggle", h.ServerToggle)
	})
}

func (h *WebHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if subtle.ConstantTimeCompare([]byte(auth), []byte("Bearer "+h.adminToken)) == 1 {
			next.ServeHTTP(w, r)
			return
		}
		cookie, _ := r.Cookie("admin_session")
		if cookie != nil && validateSessionToken(cookie.Value, h.adminToken) {
			if r.Method == http.MethodPost {
				if !sameOrigin(r) {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
				csrfCookie, _ := r.Cookie("admin_csrf")
				if csrfCookie == nil || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(csrfCookie.Value)) != 1 {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
			}
			next.ServeHTTP(w, r)
			return
		}

		http.Redirect(w, r, "/admin/login", http.StatusFound)
	})
}

func (h *WebHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "login.html", nil); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) LoginPost(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	if token != h.adminToken {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("invalid token"))
		return
	}
	sessionToken, csrfToken, err := newSessionToken(h.adminToken)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_session",
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_csrf",
		Value:    csrfToken,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})
	http.Redirect(w, r, "/admin/", http.StatusFound)
}

func (h *WebHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	stats, err := h.statsUC.GetDashboardStats(r.Context())
	if err != nil {
		slog.Error("web dashboard stats failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := h.templates.ExecuteTemplate(w, "dashboard.html", map[string]any{"Stats": stats}); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) UsersPage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "users.html", nil); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) NodesPage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "nodes.html", nil); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) PaymentsPage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "payments.html", nil); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) RoutingPage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "routing.html", nil); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) PlansPage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "plans.html", nil); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	s, err := h.settingsRepo.Get(r.Context())
	if err != nil {
		slog.Error("web settings get", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := h.templates.ExecuteTemplate(w, "settings.html", map[string]any{"S": s}); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) ServersPage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "servers.html", nil); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) StatsFragment(w http.ResponseWriter, r *http.Request) {
	stats, err := h.statsUC.GetDashboardStats(r.Context())
	if err != nil {
		http.Error(w, "<div class='card'>Ошибка статистики</div>", http.StatusInternalServerError)
		return
	}
	_ = h.templates.ExecuteTemplate(w, "stats_fragment.html", map[string]any{"Stats": stats})
}

func (h *WebHandler) UsersFragment(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	page := parseIntDefault(r.URL.Query().Get("page"), 1)
	users, total, err := h.userUC.List(r.Context(), q, page, 50)
	if err != nil {
		http.Error(w, "<div>Ошибка загрузки пользователей</div>", http.StatusInternalServerError)
		return
	}
	_ = h.templates.ExecuteTemplate(w, "users_table.html", map[string]any{
		"Users": users,
		"Total": total,
		"Page":  page,
	})
}

func (h *WebHandler) GrantUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	tier := domain.SubscriptionTier(r.FormValue("tier"))
	if tier == "" {
		tier = domain.TierBasic
	}
	days := parseIntDefault(r.FormValue("days"), 30)
	if tier != domain.TierBasic && tier != domain.TierPremium {
		http.Error(w, "invalid tier", http.StatusBadRequest)
		return
	}
	if days <= 0 || days > 3650 {
		http.Error(w, "invalid days", http.StatusBadRequest)
		return
	}
	if _, err := h.subUC.ActivateSubscription(r.Context(), id, tier, days); err != nil {
		http.Error(w, "grant failed", http.StatusInternalServerError)
		return
	}
	h.userRow(w, r, id)
}

func (h *WebHandler) RevokeUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := h.subUC.DeactivateSubscription(r.Context(), id); err != nil {
		http.Error(w, "revoke failed", http.StatusInternalServerError)
		return
	}
	h.userRow(w, r, id)
}

func (h *WebHandler) NodesFragment(w http.ResponseWriter, r *http.Request) {
	nodes, err := h.nodeUC.GetAllNodes(r.Context())
	if err != nil {
		http.Error(w, "<div>Ошибка загрузки нод</div>", http.StatusInternalServerError)
		return
	}
	_ = h.templates.ExecuteTemplate(w, "nodes_fragment.html", map[string]any{"Nodes": nodes})
}

func (h *WebHandler) NodeCheckFragment(w http.ResponseWriter, r *http.Request) {
	// Принудительная проверка в рамках фазы 3: просто возвращаем обновлённый список.
	h.NodesFragment(w, r)
}

func (h *WebHandler) PaymentsFragment(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	period := r.URL.Query().Get("period")
	page := parseIntDefault(r.URL.Query().Get("page"), 1)
	rows, total, err := h.paymentUC.ListByFilter(r.Context(), status, period, page, 50)
	if err != nil {
		http.Error(w, "<div>Ошибка загрузки платежей</div>", http.StatusInternalServerError)
		return
	}
	sum := 0
	for _, p := range rows {
		if p.Status == domain.PaymentSucceeded {
			sum += p.AmountRub
		}
	}
	_ = h.templates.ExecuteTemplate(w, "payments_table.html", map[string]any{
		"Payments": rows,
		"Total":    total,
		"Sum":      sum,
	})
}

func (h *WebHandler) RoutingFragment(w http.ResponseWriter, r *http.Request) {
	l, err := h.routingUC.GetLists(r.Context())
	if err != nil {
		http.Error(w, "<div>Ошибка routing</div>", http.StatusInternalServerError)
		return
	}
	_ = h.templates.ExecuteTemplate(w, "routing_fragment.html", map[string]any{"List": l})
}

func (h *WebHandler) RoutingUpdate(w http.ResponseWriter, r *http.Request) {
	if err := h.routingUC.UpdateFromAntifilter(r.Context()); err != nil {
		http.Error(w, "<div>Ошибка обновления</div>", http.StatusInternalServerError)
		return
	}
	h.RoutingFragment(w, r)
}

func (h *WebHandler) RoutingAdd(w http.ResponseWriter, r *http.Request) {
	d := strings.TrimSpace(r.FormValue("domain"))
	action := domain.RouteAction(r.FormValue("action"))
	if d == "" {
		http.Error(w, "<div>domain required</div>", http.StatusBadRequest)
		return
	}
	if action == "" {
		action = domain.ActionProxyEU
	}
	if action != domain.ActionProxyEU && action != domain.ActionProxyUSA && action != domain.ActionDirect && action != domain.ActionBlock {
		http.Error(w, "<div>invalid action</div>", http.StatusBadRequest)
		return
	}
	if err := h.routingUC.AddDomain(r.Context(), d, action); err != nil {
		http.Error(w, "<div>Ошибка добавления</div>", http.StatusInternalServerError)
		return
	}
	h.RoutingFragment(w, r)
}

func (h *WebHandler) RoutingDelete(w http.ResponseWriter, r *http.Request) {
	d := strings.TrimSpace(r.FormValue("domain"))
	if d == "" {
		http.Error(w, "<div>domain required</div>", http.StatusBadRequest)
		return
	}
	if err := h.routingUC.RemoveDomain(r.Context(), d); err != nil {
		http.Error(w, "<div>Ошибка удаления</div>", http.StatusInternalServerError)
		return
	}
	h.RoutingFragment(w, r)
}

func (h *WebHandler) PlansFragment(w http.ResponseWriter, r *http.Request) {
	plans, err := h.planRepo.ListAll(r.Context())
	if err != nil {
		http.Error(w, "<div>Ошибка тарифов</div>", http.StatusInternalServerError)
		return
	}
	_ = h.templates.ExecuteTemplate(w, "plans_fragment.html", map[string]any{"Plans": plans})
}

func (h *WebHandler) PlanToggle(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	p, err := h.planRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	p.Active = !p.Active
	if err := h.planRepo.Update(r.Context(), p); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}
	h.PlansFragment(w, r)
}

func (h *WebHandler) PlanPrice(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	kopeks, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("price_kopeks")), 10, 64)
	if err != nil || kopeks <= 0 || kopeks > 99_999_999 {
		http.Error(w, "invalid price_kopeks", http.StatusBadRequest)
		return
	}
	p, err := h.planRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	p.PriceKopeks = kopeks
	if err := h.planRepo.Update(r.Context(), p); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}
	h.PlansFragment(w, r)
}

func (h *WebHandler) SettingsSave(w http.ResponseWriter, r *http.Request) {
	s, err := h.settingsRepo.Get(r.Context())
	if err != nil {
		http.Error(w, "load settings failed", http.StatusInternalServerError)
		return
	}
	s.TrialDays = parseIntDefault(r.FormValue("trial_days"), s.TrialDays)
	s.ReferralBonusDays = parseIntDefault(r.FormValue("referral_bonus_days"), s.ReferralBonusDays)
	s.SupportURL = strings.TrimSpace(r.FormValue("support_url"))
	if s.TrialDays < 0 || s.TrialDays > 365 {
		http.Error(w, "invalid trial_days", http.StatusBadRequest)
		return
	}
	if s.ReferralBonusDays < 0 || s.ReferralBonusDays > 365 {
		http.Error(w, "invalid referral_bonus_days", http.StatusBadRequest)
		return
	}
	if err := h.settingsRepo.Upsert(r.Context(), s); err != nil {
		http.Error(w, "save failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<div class="card ok">Сохранено</div>`))
}

func (h *WebHandler) ServersFragment(w http.ResponseWriter, r *http.Request) {
	list, err := h.serverRepo.ListAll(r.Context())
	if err != nil {
		http.Error(w, "<div>Ошибка серверов</div>", http.StatusInternalServerError)
		return
	}
	_ = h.templates.ExecuteTemplate(w, "servers_fragment.html", map[string]any{"Servers": list})
}

func (h *WebHandler) ServerAdd(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	inbound := parseIntDefault(r.FormValue("inbound_id"), 0)
	s := &domain.VPNServer{
		ID:         uuid.New(),
		Name:       name,
		Region:     strings.TrimSpace(r.FormValue("region")),
		XUIBaseURL: strings.TrimSpace(r.FormValue("xui_base_url")),
		InboundID:  inbound,
		Active:     true,
		Notes:      strings.TrimSpace(r.FormValue("notes")),
	}
	if err := h.serverRepo.Create(r.Context(), s); err != nil {
		http.Error(w, "create failed", http.StatusInternalServerError)
		return
	}
	h.ServersFragment(w, r)
}

func (h *WebHandler) ServerToggle(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	s, err := h.serverRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	s.Active = !s.Active
	now := time.Now()
	s.LastCheckedAt = &now
	if err := h.serverRepo.Update(r.Context(), s); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}
	h.ServersFragment(w, r)
}

func (h *WebHandler) userRow(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	uid := id.String()
	users, _, _ := h.userUC.List(r.Context(), "", 1, 5000)
	for _, u := range users {
		if u.ID.String() == uid {
			_ = h.templates.ExecuteTemplate(w, "user_row.html", u)
			return
		}
	}
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
