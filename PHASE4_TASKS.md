# Фаза 4 — Маршрутизация и CDN
# Задачи для Cursor (выполнять строго по порядку)

## Что уже готово
- ✅ Фазы 1-3: весь бэкенд, панели управления

## Цель Фазы 4
1. Умная маршрутизация: через VPN идёт только заблокированное, остальное напрямую
2. AI-сервисы → только через USA-ноду
3. CDN через Яндекс Cloud CDN — резерв при белых списках операторов
4. Автообновление списков заблокированных доменов (antifilter.download)

---

## Задача 4.1 — Routing domain-модель

Файл: `internal/domain/routing_rule.go` (расширить существующий)

```go
type RouteAction string
const (
    ActionProxyEU  RouteAction = "proxy_eu"   // через EU-ноду
    ActionProxyUSA RouteAction = "proxy_usa"  // через USA-ноду (AI-сервисы)
    ActionDirect   RouteAction = "direct"     // напрямую без VPN
    ActionBlock    RouteAction = "block"       // заблокировать
)

type RoutingList struct {
    Version   string      // дата обновления "2025-03-27"
    UpdatedAt time.Time
    ProxyEU   []string    // домены через EU (заблокированные в РФ)
    ProxyUSA  []string    // AI-сервисы (всегда через USA)
    Direct    []string    // российские сервисы (напрямую)
}

// Фиксированный список AI-сервисов для USA-ноды (обновляется вручную)
var AIServiceDomains = []string{
    "openai.com", "chat.openai.com", "api.openai.com",
    "anthropic.com", "claude.ai",
    "gemini.google.com", "bard.google.com",
    "midjourney.com", "discord.com",  // Discord нужен для Midjourney
    "stability.ai", "perplexity.ai",
    "huggingface.co", "replicate.com",
    "together.ai", "groq.com",
}

// Фиксированный список российских сервисов (всегда напрямую)
var DirectDomains = []string{
    "vk.com", "vk.ru", "ok.ru",
    "yandex.ru", "ya.ru", "yandex.com",
    "sber.ru", "sberbank.ru",
    "gosuslugi.ru", "mos.ru",
    "nalog.ru", "pfr.gov.ru",
    "tinkoff.ru", "alfabank.ru", "vtb.ru",
    "mail.ru", "rambler.ru",
    "avito.ru", "wildberries.ru", "ozon.ru",
    "hh.ru", "kinopoisk.ru",
}
```

---

## Задача 4.2 — Routing Repository

Файл: `internal/repository/routing_repository.go` (расширить)

```go
// SaveDomains сохраняет список доменов одного источника (batch upsert)
func (r *routingRepository) SaveDomains(ctx context.Context, source string, action RouteAction, domains []string) error

// GetRoutingList собирает актуальный RoutingList из всех источников
func (r *routingRepository) GetRoutingList(ctx context.Context) (*domain.RoutingList, error)

// GetVersion возвращает дату последнего обновления
func (r *routingRepository) GetVersion(ctx context.Context) (string, error)

// AddManualDomain добавляет домен вручную (из Telegram-бота менеджера)
func (r *routingRepository) AddManualDomain(ctx context.Context, domain string, action RouteAction) error

// DeleteManualDomain удаляет вручную добавленный домен
func (r *routingRepository) DeleteManualDomain(ctx context.Context, domain string) error
```

---

## Задача 4.3 — Routing UseCase

Файл: `internal/usecase/routing_usecase.go` (новый)

```go
type RoutingUseCase interface {
    // GetLists возвращает актуальные списки для мобильного клиента
    GetLists(ctx context.Context) (*domain.RoutingList, error)

    // UpdateFromAntifilter скачивает и сохраняет список с antifilter.download
    UpdateFromAntifilter(ctx context.Context) error

    // AddDomain добавляет домен вручную (из панели управления)
    AddDomain(ctx context.Context, domain string, action domain.RouteAction) error

    // RemoveDomain удаляет вручную добавленный домен
    RemoveDomain(ctx context.Context, domain string) error
}
```

`UpdateFromAntifilter`:
1. GET `https://antifilter.download/list/domains.lst` — список заблокированных доменов
2. GET `https://antifilter.download/list/allyouneed.lst` — список заблокированных IP
3. Парсим построчно, фильтруем мусор (комментарии, пустые строки)
4. Batch upsert в routing_rules таблицу (source="antifilter", action="proxy_eu")
5. Обновляем версию (дата)

Таймаут HTTP запроса: 60 секунд (файл большой — ~500K строк).
При ошибке загрузки — не затирать старые данные, логировать и вернуть ошибку.

---

## Задача 4.4 — Routing API Endpoint

Файл: `internal/handler/api/routing_handler.go`

```go
// GET /api/v1/routing/lists
// Публичный endpoint (без авторизации) — мобильный клиент скачивает при запуске
// Отдаёт JSON с тремя списками + версией
// Кешировать в памяти на 1 час (sync.RWMutex + время последней генерации)
func (h *RoutingHandler) GetLists(w http.ResponseWriter, r *http.Request)
```

Ответ:
```json
{
  "version": "2025-03-27",
  "updated_at": "2025-03-27T00:00:00Z",
  "proxy_eu": ["instagram.com", "facebook.com", "..."],
  "proxy_usa": ["openai.com", "claude.ai", "..."],
  "direct": ["vk.com", "yandex.ru", "..."]
}
```

Ограничение: proxy_eu может быть очень большим (100K+ доменов).
Отдавать сжатым (gzip middleware уже должен быть в chi).

---

## Задача 4.5 — Routing Update Worker

Файл: `internal/worker/routing_update_worker.go`

```go
// Интервал: раз в 24 часа (в 03:00 по UTC)
// При запуске сервиса: если данные старше 24 часов — обновить сразу

type RoutingUpdateWorker struct {
    routingUC usecase.RoutingUseCase
    interval  time.Duration
}

func (w *RoutingUpdateWorker) Start(ctx context.Context) {
    // Проверить при старте — если список пустой или старый, обновить сразу
    // Затем по расписанию раз в 24 часа
}
```

При успешном обновлении логировать количество доменов:
```go
slog.Info("routing lists updated",
    "proxy_eu_count", len(euDomains),
    "source", "antifilter.download",
)
```

---

## Задача 4.6 — Routing в панелях управления

### Telegram-бот (добавить в tasks 3.x)

Добавить в главное меню кнопку `[🗺 Роутинг]`.
Callback `routing_main`:
```
🗺 Маршрутизация

📊 Доменов в базе: 127 483
📅 Обновлено: 27 марта 2025

🇪🇺 proxy_eu: 127 341 (antifilter)
🇺🇸 proxy_usa: 16 (AI-сервисы, ручные)
↩️ direct: 23 (российские, ручные)
✋ manual: 4 (добавлены вручную)

[➕ Добавить домен] [🔄 Обновить сейчас] [← Назад]
```

`Добавить домен` → ввести домен → выбрать действие (кнопки) → подтвердить.
`Обновить сейчас` → запустить `routingUC.UpdateFromAntifilter()` → отчёт.

### Веб-панель (новая страница)

Роут: `GET /admin/routing`
Файл: `internal/handler/web/templates/routing.html`

- Счётчики доменов по категориям
- Дата последнего обновления
- Кнопка "Обновить сейчас" (htmx POST)
- Форма добавления домена вручную
- Таблица вручную добавленных доменов с кнопкой удаления

---

## Задача 4.7 — Яндекс CDN: серверная часть

Это настройка инфраструктуры + изменения в config_usecase.

### Что нужно настроить на EU-ноде (документация для DevOps)

Создай файл `docs/cdn-setup.md`:

```markdown
# Настройка Яндекс CDN для резервного подключения

## Зачем
Когда операторы РФ включают "белые списки", прямое подключение к EU/USA нодам блокируется.
Яндекс Cloud CDN находится в белом списке у всех операторов — трафик идёт через него.

## Настройка Яндекс Cloud CDN
1. Яндекс Cloud Console → CDN → Создать ресурс
2. Источник: https://{EU_NODE_IP}:8443
3. Домен: vpn.freeway.app (CNAME → *.edgecdn.ru)
4. SSL: выпустить через Яндекс (бесплатно)
5. Настройки: передавать все заголовки, не кешировать (Cache-Control: no-store)

## WebSocket inbound на EU-ноде (добавить в Xray config)
{
  "tag": "ws-cdn",
  "port": 8443,
  "protocol": "vless",
  "streamSettings": {
    "network": "ws",
    "security": "tls",
    "wsSettings": { "path": "/vless-yndx" },
    "tlsSettings": {
      "certificates": [{
        "certificateFile": "/etc/ssl/freeway.crt",
        "keyFile": "/etc/ssl/freeway.key"
      }]
    }
  }
}
```

### Изменения в config_usecase.go

CDN-нода уже есть в конфиге. Убедись что:
1. В subscription URL CDN-нода всегда идёт ПОСЛЕДНЕЙ
2. Используется `generateVLESSWebSocket()` для CDN-ноды
3. CDN доступна только тарифу Premium (согласно TierLimitsMap)

---

## Задача 4.8 — Интеграционное тестирование маршрутизации

Файл: `internal/usecase/routing_usecase_test.go`

```go
func TestRoutingUseCase_GetLists(t *testing.T) {
    // Тест: возвращает корректную структуру с правильными списками
    // Тест: AI-сервисы всегда в proxy_usa
    // Тест: российские домены всегда в direct
}

func TestRoutingUseCase_UpdateFromAntifilter(t *testing.T) {
    // Mock HTTP сервер с тестовым списком доменов
    // Тест: домены сохраняются корректно
    // Тест: при ошибке HTTP — старые данные сохраняются
    // Тест: дублирующиеся домены не создают проблем (upsert)
}
```

---

## Задача 4.9 — Обновить .cursorrules и CLAUDE.md

После завершения Фазы 4 обнови статус в обоих файлах:
```
✅ Фаза 4 — Маршрутизация и CDN (ЗАВЕРШЕНА)
```

И добавь в .cursorrules информацию о новых endpoints:
```
GET /api/v1/routing/lists — публичный, без авторизации, отдаёт gzip JSON
```

---

## Проверка Фазы 4

```bash
go build ./...   # 0 ошибок
go test ./...    # PASS
```

Функциональная проверка:
1. `GET /api/v1/routing/lists` → JSON с тремя списками и версией
2. Руками вызвать `routingUC.UpdateFromAntifilter()` → логи с количеством доменов
3. В Telegram-боте: кнопка "Обновить сейчас" → отчёт
4. Добавить домен через бота → появляется в `/api/v1/routing/lists`
5. Subscription URL для Premium пользователя содержит CDN-конфиг последним

---

## Порядок: 4.1 → 4.2 → 4.3 → 4.4 → 4.5 → 4.6 → 4.7 → 4.8 → 4.9
