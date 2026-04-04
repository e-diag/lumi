package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/google/uuid"
)

type fakeUserUC struct {
	user *domain.User
}

func (f *fakeUserUC) Register(_ context.Context, telegramID int64, username string) (*domain.User, error) {
	if f.user != nil {
		return f.user, nil
	}
	u := domain.NewUser(telegramID, username)
	u.DeviceLimit = 1
	f.user = u
	return u, nil
}
func (f *fakeUserUC) GetByTelegramID(_ context.Context, _ int64) (*domain.User, error) {
	return f.user, nil
}
func (f *fakeUserUC) GetByID(_ context.Context, _ uuid.UUID) (*domain.User, error) {
	return f.user, nil
}
func (f *fakeUserUC) GetBySubToken(_ context.Context, _ string) (*domain.User, error) {
	return f.user, domain.ErrUserNotFound
}
func (f *fakeUserUC) List(_ context.Context, _ string, _ int, _ int) ([]*domain.User, int64, error) {
	if f.user == nil {
		return nil, 0, nil
	}
	return []*domain.User{f.user}, 1, nil
}

func TestAuthHandler_TelegramAuth_ValidInitData_ReturnsJWT(t *testing.T) {
	botToken := "123:TEST_BOT_TOKEN"
	jwtSecret := "secret"

	initData := buildInitData(botToken, map[string]string{
		"query_id":  "AAE",
		"auth_date": strconv.FormatInt(time.Now().Unix(), 10),
		"user":      `{"id":42,"username":"tester","first_name":"Test"}`,
	})

	h := NewAuthHandler(&fakeUserUC{}, jwtSecret, botToken)

	body, _ := json.Marshal(map[string]string{"init_data": initData})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/tg", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.TelegramAuth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["access_token"] == "" {
		t.Fatalf("expected access_token, got %v", resp)
	}
	if resp["sub_token"] == "" {
		t.Fatalf("expected sub_token, got %v", resp)
	}
}

func TestAuthHandler_TelegramAuth_InvalidHash_Unauthorized(t *testing.T) {
	botToken := "123:TEST_BOT_TOKEN"
	jwtSecret := "secret"

	values := url.Values{}
	values.Set("query_id", "AAE")
	values.Set("auth_date", strconv.FormatInt(time.Now().Unix(), 10))
	values.Set("user", `{"id":42,"username":"tester"}`)
	values.Set("hash", strings.Repeat("0", 64)) // явно неправильный

	h := NewAuthHandler(&fakeUserUC{}, jwtSecret, botToken)

	body, _ := json.Marshal(map[string]string{"init_data": values.Encode()})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/tg", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.TelegramAuth(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
}

func buildInitData(botToken string, fields map[string]string) string {
	values := url.Values{}
	for k, v := range fields {
		values.Set(k, v)
	}
	values.Set("hash", calcInitDataHash(values, botToken))
	return values.Encode()
}

func calcInitDataHash(values url.Values, botToken string) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		if k == "hash" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("%s=%s", k, values.Get(k)))
	}
	dataCheckString := strings.Join(lines, "\n")

	secret := hmacSHA256Test([]byte(botToken), []byte("WebAppData"))
	sum := hmacSHA256Test(secret, []byte(dataCheckString))
	return hex.EncodeToString(sum)
}

func hmacSHA256Test(key, msg []byte) []byte {
	m := hmac.New(sha256.New, key)
	_, _ = m.Write(msg)
	return m.Sum(nil)
}
