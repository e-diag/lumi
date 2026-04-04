// Package xui — HTTP-клиент панели 3x-ui (сессия cookie, inbounds API).
package xui

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Client выполняет запросы к API 3x-ui после логина.
type Client struct {
	baseURL    string // нормализованный корень веб-панели, напр. https://host:2053/panel
	httpClient *http.Client
	username   string
	password   string

	mu        sync.Mutex
	loggedIn  bool
	loginFail time.Time
}

// NewClient создаёт клиент. baseURL — публичный URL панели с учётом webBasePath (см. настройки 3x-ui).
func NewClient(baseURL, username, password string) (*Client, error) {
	root, err := normalizePanelRootURL(baseURL)
	if err != nil {
		return nil, err
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("xui: cookie jar: %w", err)
	}
	return &Client{
		baseURL:  root,
		username: strings.TrimSpace(username),
		password: password,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
			Jar:     jar,
		},
	}, nil
}

func normalizePanelRootURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("xui: invalid base url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("xui: base url must be http(s)")
	}
	if u.Host == "" {
		return "", fmt.Errorf("xui: base url host required")
	}
	u.Fragment = ""
	u.RawQuery = ""
	path := strings.TrimRight(u.Path, "/")
	u.Path = path
	return u.String(), nil
}

type panelMsg struct {
	Success bool            `json:"success"`
	Msg     string          `json:"msg"`
	Obj     json.RawMessage `json:"obj"`
}

func (c *Client) login(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.loggedIn {
		return nil
	}
	if time.Since(c.loginFail) < 2*time.Second {
		return fmt.Errorf("xui: login backoff")
	}

	form := url.Values{}
	form.Set("username", c.username)
	form.Set("password", c.password)

	loginURL := c.baseURL + "/login"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("xui: login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	res, err := c.httpClient.Do(req)
	if err != nil {
		c.loginFail = time.Now()
		return fmt.Errorf("xui: login http: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK {
		c.loginFail = time.Now()
		return fmt.Errorf("xui: login http %d: %s", res.StatusCode, truncateForErr(body))
	}

	var msg panelMsg
	if err := json.Unmarshal(body, &msg); err != nil {
		c.loginFail = time.Now()
		return fmt.Errorf("xui: login decode: %w", err)
	}
	if !msg.Success {
		c.loginFail = time.Now()
		return fmt.Errorf("xui: login failed: %s", msg.Msg)
	}
	c.loggedIn = true
	return nil
}

func truncateForErr(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 400 {
		return s[:400] + "…"
	}
	return s
}

func (c *Client) postJSON(ctx context.Context, path string, payload any) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("xui: marshal: %w", err)
	}
	full := strings.TrimRight(c.baseURL, "/") + path

	var lastBody []byte
	var lastStatus int
	for attempt := 0; attempt < 2; attempt++ {
		if err := c.login(ctx); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, full, bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("xui: new request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")

		res, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("xui: post %s: %w", path, err)
		}
		out, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		lastBody = out
		lastStatus = res.StatusCode

		if res.StatusCode == http.StatusNotFound && attempt == 0 {
			c.mu.Lock()
			c.loggedIn = false
			c.mu.Unlock()
			continue
		}
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return nil, fmt.Errorf("xui: %s http %d: %s", path, res.StatusCode, truncateForErr(out))
		}
		var msg panelMsg
		if err := json.Unmarshal(out, &msg); err != nil {
			return out, nil
		}
		if !msg.Success {
			return nil, fmt.Errorf("xui: api %s: %s", path, msg.Msg)
		}
		return out, nil
	}
	return nil, fmt.Errorf("xui: %s http %d: %s", path, lastStatus, truncateForErr(lastBody))
}

// AddInboundClient добавляет клиента в inbound (id — числовой ID inbound в панели).
func (c *Client) AddInboundClient(ctx context.Context, inboundID int, settingsJSON string) error {
	_, err := c.postJSON(ctx, "/panel/api/inbounds/addClient", map[string]any{
		"id":       inboundID,
		"settings": settingsJSON,
	})
	return err
}

// UpdateInboundClient обновляет клиента (clientID — UUID клиента Xray для VLESS/VMess).
func (c *Client) UpdateInboundClient(ctx context.Context, inboundID int, clientID string, settingsJSON string) error {
	path := "/panel/api/inbounds/updateClient/" + url.PathEscape(clientID)
	_, err := c.postJSON(ctx, path, map[string]any{
		"id":       inboundID,
		"settings": settingsJSON,
	})
	return err
}

// InboundListItem — элемент списка inbounds (фрагмент ответа list).
type InboundListItem struct {
	ID     int    `json:"id"`
	Remark string `json:"remark"`
	Enable bool   `json:"enable"`
}

type inboundListResp struct {
	Obj []InboundListItem `json:"obj"`
}

// PingInbounds проверяет доступность API (после логина).
func (c *Client) PingInbounds(ctx context.Context) error {
	if err := c.login(ctx); err != nil {
		return err
	}
	full := strings.TrimRight(c.baseURL, "/") + "/panel/api/inbounds/list"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, full, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("xui: list inbounds http %d: %s", res.StatusCode, truncateForErr(body))
	}
	return nil
}

// BuildClientSettingsJSON формирует JSON-строку settings.clients[0] для VLESS/универсального inbound.
func BuildClientSettingsJSON(clientUUID, email string, limitIP int, expiryMs int64, enable bool, subID string) (string, error) {
	client := map[string]any{
		"id":         clientUUID,
		"email":      email,
		"limitIp":    limitIP,
		"totalGB":    0,
		"expiryTime": expiryMs,
		"enable":     enable,
		"subId":      subID,
		"tgId":       "",
		"flow":       "",
	}
	wrap := map[string]any{"clients": []any{client}}
	b, err := json.Marshal(wrap)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func newSubID() (string, error) {
	var buf [10]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func ensureClientUUID(existing string) string {
	if strings.TrimSpace(existing) != "" {
		return existing
	}
	return uuid.New().String()
}
