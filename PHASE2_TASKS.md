# Фаза 2 — Платежи и подписки
# Задачи для Cursor (по порядку)

## Статус Фазы 1 (уже готово)
- ✅ domain модели (User, Subscription, Node, Payment, RoutingRule)
- ✅ repository interfaces + GORM реализации
- ✅ usecase interfaces + реализации
- ✅ GET /sub/{token} — subscription URL генерация
- ✅ chi роутер + middleware (JWT, logger, rate limit)
- ✅ docker-compose.yml + Dockerfile
- ✅ config.yaml + .env.example

---

## Задача 2.1 — ЮKassa клиент

Создай `internal/infrastructure/yookassa/client.go`:

```go
package yookassa

type Client struct {
    shopID     string
    secretKey  string
    httpClient *http.Client
}

type CreatePaymentRequest struct {
    Amount       Amount            `json:"amount"`
    Confirmation Confirmation      `json:"confirmation"`
    Description  string            `json:"description"`
    Metadata     map[string]string `json:"metadata"` // user_id, tier, days
    Capture      bool              `json:"capture"`
}

type Amount struct {
    Value    string `json:"value"`    // "299.00"
    Currency string `json:"currency"` // "RUB"
}

type Confirmation struct {
    Type      string `json:"type"`       // "redirect"
    ReturnURL string `json:"return_url"`
}

type PaymentResponse struct {
    ID           string            `json:"id"`
    Status       string            `json:"status"` // pending/succeeded/cancelled
    Confirmation struct {
        ConfirmationURL string `json:"confirmation_url"`
    } `json:"confirmation"`
}

// CreatePayment создаёт платёж в ЮKassa
// POST https://api.yookassa.ru/v3/payments
// Auth: Basic base64(shopID:secretKey)
// Idempotence-Key: uuid (важно для защиты от двойной оплаты)
func (c *Client) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*PaymentResponse, error)

// GetPayment возвращает статус платежа
// GET https://api.yookassa.ru/v3/payments/{id}
func (c *Client) GetPayment(ctx context.Context, id string) (*PaymentResponse, error)
```

---

## Задача 2.2 — Payment UseCase

Реализуй `internal/usecase/payment_usecase.go` (заглушка уже есть):

```go
// CreatePayment создаёт платёж в ЮKassa и сохраняет в БД
// Возвращает payment_url для редиректа пользователя
func (uc *paymentUseCase) CreatePayment(ctx context.Context, userID uint, tier domain.SubscriptionTier, days int) (*domain.Payment, string, error)

// HandleWebhook обрабатывает вебхук от ЮKassa
// При status=succeeded → активирует подписку
func (uc *paymentUseCase) HandleWebhook(ctx context.Context, event WebhookEvent) error

// GetPendingPayments возвращает платежи в статусе pending (для воркера)
func (uc *paymentUseCase) GetPendingPayments(ctx context.Context) ([]domain.Payment, error)

// CheckAndUpdatePayment проверяет статус pending платежа в ЮKassa
func (uc *paymentUseCase) CheckAndUpdatePayment(ctx context.Context, paymentID uint) error
```

---

## Задача 2.3 — Subscription UseCase (дополнить)

Дополни `internal/usecase/subscription_usecase.go`:

```go
// Activate активирует подписку пользователя
// Если уже есть активная подписка — продлевает (добавляет дни)
func (uc *subscriptionUseCase) Activate(ctx context.Context, userID uint, tier domain.SubscriptionTier, days int) error

// Expire деактивирует истёкшие подписки → даунгрейд до Free
// Вызывается воркером каждую минуту
func (uc *subscriptionUseCase) ExpireOld(ctx context.Context) error

// GetActiveByUserID возвращает активную подписку пользователя
func (uc *subscriptionUseCase) GetActiveByUserID(ctx context.Context, userID uint) (*domain.Subscription, error)

// GetExpiringIn3Days возвращает подписки истекающие через 3 дня (для уведомлений)
func (uc *subscriptionUseCase) GetExpiringIn3Days(ctx context.Context) ([]domain.Subscription, error)
```

При активации подписки:
1. Создать/обновить запись в subscription таблице
2. Обновить DeviceLimit пользователя согласно тарифу
3. Обновить пользователя в Remnawave (через RemnawaveClient)

---

## Задача 2.4 — Webhook Handler

Создай `internal/handler/api/webhook_handler.go`:

```go
// POST /webhook/yookassa
// 1. Проверить IP источника (список доверенных IP ЮKassa)
// 2. Прочитать body
// 3. Распарсить событие
// 4. Если type=payment.succeeded → вызвать paymentUC.HandleWebhook
// 5. Вернуть 200 OK

// Доверенные IP ЮKassa:
// 185.71.76.0/27, 185.71.77.0/27, 77.75.153.0/25, 77.75.154.128/25

func (h *WebhookHandler) YookassaWebhook(w http.ResponseWriter, r *http.Request)
```

---

## Задача 2.5 — Payment API Handler

Создай/дополни `internal/handler/api/payment_handler.go`:

```go
// POST /api/v1/payment/create
// Auth: Bearer JWT
// Body: { "tier": "basic"|"premium", "days": 30 }
// Return: { "payment_url": "...", "payment_id": 123 }
func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request)

// GET /api/v1/payment/{id}/status
// Auth: Bearer JWT
// Return: { "status": "pending|succeeded|cancelled", "active_until": "..." }
func (h *PaymentHandler) GetPaymentStatus(w http.ResponseWriter, r *http.Request)
```

---

## Задача 2.6 — Авторизация через Telegram

Создай/дополни `internal/handler/api/auth_handler.go`:

```go
// POST /api/v1/auth/tg
// Body: { "init_data": "..." }  ← Telegram WebApp initData строка
// 1. Валидируем init_data через HMAC-SHA256 (ключ = HMAC-SHA256(bot_token, "WebAppData"))
// 2. Парсим user из init_data
// 3. Создаём или находим пользователя по TelegramID
// 4. Генерируем JWT (payload: user_id, telegram_id, exp)
// 5. Return: { "access_token": "...", "sub_token": "...", "user": {...} }
func (h *AuthHandler) TelegramAuth(w http.ResponseWriter, r *http.Request)
```

Валидация Telegram initData:
```go
// data_check_string = все поля из init_data (кроме hash), отсортированные по ключу,
//                     соединённые через \n в формате key=value
// secret_key = HMAC-SHA256(bot_token, "WebAppData")
// hash должен совпадать с HMAC-SHA256(data_check_string, secret_key)
```

---

## Задача 2.7 — Workers

### Worker 1: payment_worker.go
```
Интервал: каждые 30 секунд
Логика:
  1. GetPendingPayments() — платежи старше 5 минут в статусе pending
  2. Для каждого: CheckAndUpdatePayment()
  3. Если не оплачен > 24 часов — отменить (status=cancelled)
```

### Worker 2: subscription_worker.go
```
Интервал: каждую минуту
Логика:
  1. ExpireOld() — найти подписки где ActiveUntil < now, перевести в Free
  2. GetExpiringIn3Days() — отправить уведомление (пока просто slog.Info)
```

### Worker 3: node_health_worker.go
```
Интервал: каждые 5 минут
Логика:
  1. Для каждой активной ноды: TCP ping на host:port
  2. Если нода недоступна 2+ раза подряд: IsActive=false, slog.Error
  3. При восстановлении: IsActive=true, slog.Info
  4. Обновить LatencyMs
```

Все workers запускаются в cmd/api/main.go через goroutine + context.

---

## Задача 2.8 — Remnawave клиент

Создай `internal/infrastructure/remnawave/client.go`:

```go
type Client struct {
    baseURL    string
    apiToken   string
    httpClient *http.Client
}

// CreateUser добавляет пользователя на все ноды через Remnawave
func (c *Client) CreateUser(ctx context.Context, uuid, username string, tier domain.SubscriptionTier) error

// DeleteUser удаляет пользователя (при удалении аккаунта)
func (c *Client) DeleteUser(ctx context.Context, uuid string) error

// UpdateUserExpiry обновляет дату истечения доступа пользователя
func (c *Client) UpdateUserExpiry(ctx context.Context, uuid string, expiresAt *time.Time) error

// GetNodeStats возвращает статистику ноды (онлайн пользователей, трафик)
func (c *Client) GetNodeStats(ctx context.Context, nodeID string) (*NodeStats, error)
```

---

## Задача 2.9 — Тесты

Напиши table-driven тесты для:
- `payment_usecase_test.go` — TestCreatePayment, TestHandleWebhook
- `subscription_usecase_test.go` — TestActivate, TestExpireOld
- `auth_handler_test.go` — TestTelegramAuth (валидная/невалидная initData)

Используй `github.com/stretchr/testify/mock` для мокирования зависимостей.

---

## Проверка завершения Фазы 2

```bash
go build ./...    # 0 ошибок
go vet ./...      # 0 предупреждений
go test ./...     # все PASS
```

Функциональная проверка:
1. `docker-compose up -d postgres`
2. `go run ./cmd/api`
3. `POST /api/v1/auth/tg` с валидным Telegram initData → получить JWT
4. `POST /api/v1/payment/create` с JWT → получить payment_url
5. Симуляция вебхука ЮKassa → подписка активируется

---

## Порядок выполнения в Cursor

1. Задача 2.1 (ЮKassa клиент) — независимая, начни с неё
2. Задача 2.8 (Remnawave клиент) — независимая, можно параллельно
3. Задача 2.2 (Payment UseCase) — зависит от 2.1
4. Задача 2.3 (Subscription UseCase) — зависит от 2.8
5. Задача 2.4 (Webhook Handler) — зависит от 2.2
6. Задача 2.5 (Payment Handler) — зависит от 2.2
7. Задача 2.6 (Auth Handler) — независимая
8. Задача 2.7 (Workers) — зависит от 2.2, 2.3
9. Задача 2.9 (Тесты) — пишем по ходу каждой задачи
