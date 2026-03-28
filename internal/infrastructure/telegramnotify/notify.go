// Пакет telegramnotify отправляет сервисные сообщения пользователям через Bot API.
package telegramnotify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/freeway-vpn/backend/internal/repository"
	"github.com/google/uuid"
)

// PaymentSuccessNotifier уведомляет пользователя об успешной оплате (inline «Подключиться»).
type PaymentSuccessNotifier struct {
	token    string
	client   *http.Client
	userRepo repository.UserRepository
}

// NewPaymentSuccessNotifier создаёт нотификатор; token — токен того же бота, что и у пользователей.
func NewPaymentSuccessNotifier(botToken string, userRepo repository.UserRepository) *PaymentSuccessNotifier {
	return &PaymentSuccessNotifier{
		token:    botToken,
		userRepo: userRepo,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

type sendMessagePayload struct {
	ChatID      int64          `json:"chat_id"`
	Text        string         `json:"text"`
	ReplyMarkup map[string]any `json:"reply_markup,omitempty"`
}

// NotifySubscriptionPaid отправляет сообщение после успешной активации подписки по платежу.
func (n *PaymentSuccessNotifier) NotifySubscriptionPaid(ctx context.Context, userID uuid.UUID) error {
	if n == nil || n.token == "" || n.userRepo == nil {
		return nil
	}
	u, err := n.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("telegram notify: get user: %w", err)
	}
	payload := sendMessagePayload{
		ChatID: u.TelegramID,
		Text:   "Оплата прошла успешно.\n\nНажмите «Подключиться», чтобы продолжить.",
		ReplyMarkup: map[string]any{
			"inline_keyboard": [][]map[string]string{
				{{"text": "🔌 Подключиться", "callback_data": "u:c"}},
				{{"text": "🏠 Меню", "callback_data": "u:m"}},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram notify: marshal: %w", err)
	}
	url := "https://api.telegram.org/bot" + n.token + "/sendMessage"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram notify: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram notify: do: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram notify: status %s", resp.Status)
	}
	return nil
}
