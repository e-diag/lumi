package remnawave

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
)

// Client — минимальный HTTP-клиент Remnawave API.
// Конкретные пути API могут отличаться в зависимости от инстанса панельки;
// клиент реализован максимально консервативно и покрывает базовые операции.
type Client struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

// NewClient создаёт Remnawave-клиент.
func NewClient(baseURL, apiToken string) *Client {
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

type NodeStats struct {
	OnlineUsers int64 `json:"online_users"`
	TrafficIn   int64 `json:"traffic_in"`
	TrafficOut  int64 `json:"traffic_out"`
}

// CreateUser добавляет пользователя на все ноды через Remnawave.
func (c *Client) CreateUser(ctx context.Context, userUUID, username string, tier domain.SubscriptionTier) error {
	req := map[string]any{
		"uuid":     userUUID,
		"username": username,
		"tier":     string(tier),
	}
	return c.postJSON(ctx, "/users", req, nil)
}

// DeleteUser удаляет пользователя (при удалении аккаунта).
func (c *Client) DeleteUser(ctx context.Context, userUUID string) error {
	return c.do(ctx, http.MethodDelete, "/users/"+userUUID, nil, nil)
}

// UpdateUserExpiry обновляет дату истечения доступа пользователя.
// expiresAt=nil означает “безлимит / не истекает” (если панель это поддерживает).
func (c *Client) UpdateUserExpiry(ctx context.Context, userUUID string, expiresAt *time.Time) error {
	var expires *string
	if expiresAt != nil {
		s := expiresAt.UTC().Format(time.RFC3339)
		expires = &s
	}
	req := map[string]any{
		"expires_at": expires,
	}
	return c.postJSON(ctx, "/users/"+userUUID+"/expiry", req, nil)
}

// GetNodeStats возвращает статистику ноды (онлайн пользователей, трафик).
func (c *Client) GetNodeStats(ctx context.Context, nodeID string) (*NodeStats, error) {
	var out NodeStats
	if err := c.do(ctx, http.MethodGet, "/nodes/"+nodeID+"/stats", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) postJSON(ctx context.Context, path string, in any, out any) error {
	b, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("remnawave: marshal request: %w", err)
	}
	return c.do(ctx, http.MethodPost, path, bytes.NewReader(b), out)
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, out any) error {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("remnawave: create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiToken)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("remnawave: request failed: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("remnawave: read response body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("remnawave: http %d: %s", res.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("remnawave: decode json: %w", err)
	}
	return nil
}

