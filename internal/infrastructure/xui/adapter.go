package xui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/usecase"
)

// Config параметры синхронизации с одной панелью 3x-ui.
type Config struct {
	BaseURL   string // полный URL корня панели, например https://vpn.example.com:2053/panel
	Username  string
	Password  string
	InboundID int
	LimitIP   int // лимит устройств (по умолчанию 3)
}

// Adapter реализует usecase.VPNPanelClient для 3x-ui.
type Adapter struct {
	cfg    Config
	client *Client
}

// NewAdapter создаёт адаптер. При ошибке конфигурации HTTP-клиента — ошибка.
func NewAdapter(cfg Config) (*Adapter, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, fmt.Errorf("xui: empty base url")
	}
	if cfg.InboundID <= 0 {
		return nil, fmt.Errorf("xui: inbound id required")
	}
	if cfg.LimitIP <= 0 {
		cfg.LimitIP = 3
	}
	c, err := NewClient(cfg.BaseURL, cfg.Username, cfg.Password)
	if err != nil {
		return nil, err
	}
	return &Adapter{cfg: cfg, client: c}, nil
}

var _ usecase.VPNPanelClient = (*Adapter)(nil)

// SyncUserAccess создаёт или обновляет клиента в inbound, отключает при истёкшей free-подписке.
func (a *Adapter) SyncUserAccess(ctx context.Context, user *domain.User, tier domain.SubscriptionTier, expiresAt *time.Time) (*usecase.PanelSyncResult, error) {
	if user == nil {
		return nil, fmt.Errorf("xui: nil user")
	}

	clientUUID := ensureClientUUID(user.PanelClientUUID)
	email := panelEmail(user.ID)
	subID := strings.TrimSpace(user.PanelSubID)
	if subID == "" {
		sid, err := newSubID()
		if err != nil {
			return nil, err
		}
		subID = sid
	}

	var expiryMs int64
	enable := true
	if expiresAt != nil {
		expiryMs = expiresAt.UTC().UnixMilli()
	}
	// Истёкший доступ или явный free-tier с нулевым сроком — отключаем клиента в панели.
	if tier == domain.TierFree && expiresAt != nil && !expiresAt.After(time.Now()) {
		enable = false
		expiryMs = time.Now().UTC().UnixMilli()
	}

	settings, err := BuildClientSettingsJSON(clientUUID, email, a.cfg.LimitIP, expiryMs, enable, subID)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(user.PanelClientUUID) == "" {
		if err := a.client.AddInboundClient(ctx, a.cfg.InboundID, settings); err != nil {
			return nil, err
		}
		return &usecase.PanelSyncResult{ClientUUID: clientUUID, SubID: subID}, nil
	}

	if err := a.client.UpdateInboundClient(ctx, a.cfg.InboundID, clientUUID, settings); err != nil {
		return nil, err
	}
	return &usecase.PanelSyncResult{ClientUUID: clientUUID, SubID: subID}, nil
}

func panelEmail(userID interface{ String() string }) string {
	s := strings.ReplaceAll(userID.String(), "-", "")
	if len(s) > 24 {
		s = s[:24]
	}
	return "fw-" + s + "@svc"
}
