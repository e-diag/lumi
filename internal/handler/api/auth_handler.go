package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/freeway-vpn/backend/internal/usecase"
	"github.com/golang-jwt/jwt/v5"
)

// AuthHandler — авторизация через Telegram WebApp initData.
type AuthHandler struct {
	userUC    usecase.UserUseCase
	jwtSecret string
	botToken  string
}

func NewAuthHandler(userUC usecase.UserUseCase, jwtSecret, botToken string) *AuthHandler {
	return &AuthHandler{
		userUC:    userUC,
		jwtSecret: jwtSecret,
		botToken:  botToken,
	}
}

type telegramAuthRequest struct {
	InitData string `json:"init_data"`
}

type telegramUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// TelegramAuth: POST /api/v1/auth/tg
func (h *AuthHandler) TelegramAuth(w http.ResponseWriter, r *http.Request) {
	var req telegramAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.InitData == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "init_data required"})
		return
	}

	values, err := url.ParseQuery(req.InitData)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid init_data"})
		return
	}

	if !validateTelegramInitData(values, h.botToken) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid init_data"})
		return
	}

	userJSON := values.Get("user")
	if userJSON == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user missing"})
		return
	}
	var tgUser telegramUser
	if err := json.Unmarshal([]byte(userJSON), &tgUser); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user"})
		return
	}

	username := tgUser.Username
	if username == "" {
		username = strings.TrimSpace(tgUser.FirstName + " " + tgUser.LastName)
	}

	u, err := h.userUC.Register(r.Context(), tgUser.ID, username)
	if err != nil {
		slog.Error("telegram auth: register user failed", "error", err, "telegram_id", tgUser.ID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	exp := time.Now().Add(30 * 24 * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":         u.ID.String(),
		"telegram_id": u.TelegramID,
		"exp":         exp.Unix(),
	})
	accessToken, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		slog.Error("telegram auth: sign jwt failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": accessToken,
		"sub_token":    u.SubToken,
		"user": map[string]any{
			"id":           u.ID.String(),
			"telegram_id":  u.TelegramID,
			"username":     u.Username,
			"device_limit": u.DeviceLimit,
		},
	})
}

func validateTelegramInitData(values url.Values, botToken string) bool {
	receivedHash := values.Get("hash")
	if receivedHash == "" {
		return false
	}

	// data_check_string: все поля кроме hash, сортировка по ключу, строки key=value, join \n.
	keys := make([]string, 0, len(values))
	for k := range values {
		if k == "hash" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, values.Get(k)))
	}
	dataCheckString := strings.Join(parts, "\n")

	// secret_key = HMAC-SHA256(bot_token, "WebAppData")
	secret := hmacSHA256([]byte(botToken), []byte("WebAppData"))
	calculated := hmacSHA256(secret, []byte(dataCheckString))

	want, err := hex.DecodeString(receivedHash)
	if err != nil {
		return false
	}
	if len(want) != sha256.Size {
		return false
	}
	return subtle.ConstantTimeCompare(want, calculated) == 1
}

func hmacSHA256(key, msg []byte) []byte {
	m := hmac.New(sha256.New, key)
	_, _ = m.Write(msg)
	return m.Sum(nil)
}
