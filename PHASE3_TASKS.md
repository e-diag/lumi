# Фаза 3 — Панели управления менеджера
# Задачи для Cursor (выполнять строго по порядку)

## Что уже готово
- ✅ Фаза 1: domain, usecase, repository, GET /sub/{token}
- ✅ Фаза 2: ЮKassa, webhook, auth/tg, workers

## Цель Фазы 3
Два интерфейса для менеджера:
1. Telegram-бот — быстрые действия с телефона
2. Веб-панель — детальная статистика с компьютера (Go templates + htmx)

---

## Задача 3.1 — StatsUseCase реализация

Файл: `internal/usecase/stats_usecase.go`
Делай первым — нужен всем остальным задачам.

```go
type DashboardStats struct {
    TotalUsers          int
    FreeUsers           int
    BasicUsers          int
    PremiumUsers        int
    ActiveSubscriptions int
    RevenueToday        float64
    RevenueMonth        float64
    PaymentsToday       int
    Nodes               []NodeStatus
    RecentPayments      []PaymentSummary // последние 10
    NewUsersPerDay      []DailyCount     // последние 30 дней
}

type NodeStatus struct {
    Name      string
    Region    string
    IsOnline  bool
    LatencyMs int
    Online    int
}

func (uc *statsUseCase) GetDashboardStats(ctx context.Context) (*DashboardStats, error)
func (uc *statsUseCase) GetFinanceStats(ctx context.Context, period string) (*FinanceStats, error)
```

---

## Задача 3.2 — Telegram-бот: инфраструктура + главное меню

Файлы: `cmd/bot/main.go`, `internal/handler/bot/bot_handler.go`

Используй `github.com/go-telegram/bot`. Все команды проверяют adminIDs из конфига.

Команда `/start` → inline-кнопки:
```
[📊 Статистика] [🌐 Ноды]
[👥 Пользователи] [💰 Финансы]
[📢 Рассылка]    [⚙️ Настройки]
```
Каждая кнопка — callback. При нажатии редактировать сообщение, не слать новое.
Кнопка `← Назад` везде возвращает в главное меню.

---

## Задача 3.3 — Telegram-бот: статистика и ноды

Файлы: `internal/handler/bot/stats_handler.go`, `internal/handler/bot/nodes_handler.go`

**Статистика** (callback `stats`):
```
📊 FreeWay — статистика

👥 Всего: 2 847
├ 🆓 Free: 2 421
├ 🔵 Basic: 312
└ 💎 Premium: 114

💰 Сегодня: 8 940 ₽ (29 платежей)
💰 Месяц: 234 180 ₽

🌐 EU-NL: 🟢 45мс / 187 онлайн
🌐 USA:   🟢 118мс / 94 онлайн
🌐 CDN:   🟢 резерв

[🔄 Обновить] [← Назад]
```

**Ноды** (callback `nodes`) — карточки нод + кнопка принудительного health check.

---

## Задача 3.4 — Telegram-бот: пользователи

Файл: `internal/handler/bot/users_handler.go`

Команда `/user <telegram_id>`:
```
👤 @username (#1247)
Тариф: 💎 Premium
Активна до: 15 апреля | Осталось: 18 дней
Платежей: 5 / Потрачено: 1 495 ₽

[✅ Выдать подписку] [❌ Отозвать] [← Назад]
```

`Выдать подписку` → выбор тарифа (кнопки) → ввод дней текстом → confirm → `subscriptionUC.Activate()`
`Отозвать` → confirm → `subscriptionUC.Deactivate()`

---

## Задача 3.5 — Telegram-бот: финансы и рассылка

Файлы: `internal/handler/bot/finance_handler.go`, `internal/handler/bot/broadcast_handler.go`

**Финансы** (callback `finance`): статистика платежей по периодам + последние 5 платежей.

**Рассылка** (callback `broadcast`):
1. Попросить текст
2. Превью + `[✅ Отправить] [❌ Отмена]`
3. Батчи по 30 msg/sec
4. Отчёт: "Отправлено: 2847, Ошибок: 12"

---

## Задача 3.6 — Веб-панель: инфраструктура и base template

Файлы: `cmd/web/main.go`, `internal/handler/web/web_handler.go`

**Технология: Go templates + htmx. НИКАКОГО React/Vue/Angular.**
Подключать только: htmx через CDN.
CSS — чистый, без Bootstrap/Tailwind. Тёмная тема.

```go
type WebHandler struct {
    // зависимости через usecase интерфейсы
    adminToken string  // Bearer токен из конфига
    templates  *template.Template
}
```

Auth middleware: проверка `Authorization: Bearer {token}` или session cookie.
Если не авторизован → редирект на `/admin/login`.

`internal/handler/web/templates/base.html`:
- Тёмный фон: `#0f0f0f`, карточки: `#1a1a1a`
- Боковая навигация: Dashboard / Пользователи / Ноды / Платежи
- Подключение htmx: `<script src="https://unpkg.com/htmx.org@1.9.10"></script>`

---

## Задача 3.7 — Веб-панель: Dashboard

Роут: `GET /admin/`
Файл: `internal/handler/web/templates/dashboard.html`

**4 карточки** в CSS Grid:
```
[👥 2,847 пользователей] [💎 426 платных] [💰 234,180 ₽] [🌐 3/3 нод]
```

**SVG-график** новых пользователей за 30 дней (чистый SVG, без Chart.js).

**Таблица** последних 10 платежей.

**Статус нод** (список с цветовыми индикаторами).

htmx автообновление карточек каждые 30 сек:
```html
<div hx-get="/admin/api/stats" hx-trigger="every 30s" hx-swap="innerHTML">
```

---

## Задача 3.8 — Веб-панель: пользователи

Роут: `GET /admin/users`
Файл: `internal/handler/web/templates/users.html`

Поиск с htmx (без перезагрузки страницы):
```html
<input hx-get="/admin/api/users" hx-trigger="keyup changed delay:300ms"
       hx-target="#users-table" name="q" placeholder="Поиск...">
```

Таблица: TG ID | Username | Тариф | Активна до | Дней | [Выдать] [Отозвать]

Действия через htmx POST, обновляют только строку таблицы.
Пагинация: 50 пользователей на страницу.

---

## Задача 3.9 — Веб-панель: ноды и платежи

Роуты: `GET /admin/nodes`, `GET /admin/payments`

**Ноды**: карточки с индикатором онлайн/офлайн, пинг (htmx 30 сек), кнопка проверки.

**Платежи**: фильтры по статусу и периоду через htmx.
Таблица: ЮKassa ID | Пользователь | Тариф | Сумма | Статус | Дата.
Итоговая строка: сумма за период.

---

## Задача 3.10 — Веб-панель: htmx API

Роуты возвращают HTML-фрагменты (не JSON):
```
GET  /admin/api/stats              → HTML карточки статистики
GET  /admin/api/users?q=&page=     → HTML таблица пользователей
POST /admin/api/users/{id}/grant   → HTML обновлённая строка
POST /admin/api/users/{id}/revoke  → HTML обновлённая строка
GET  /admin/api/nodes              → HTML список нод
POST /admin/api/nodes/{id}/check   → HTML карточка ноды
GET  /admin/api/payments?status=&period= → HTML таблица платежей
```

---

## Проверка Фазы 3

```bash
go build ./...   # 0 ошибок
go test ./...    # PASS
```

1. `go run ./cmd/bot` → /start → видишь меню с кнопками
2. `go run ./cmd/web` → http://localhost:3000/admin/ → dashboard открывается
3. Поиск пользователя в веб-панели работает без перезагрузки
4. Кнопки "Выдать/Отозвать" в боте вызывают usecase корректно

---

## Порядок: 3.1 → 3.2 → 3.3 → 3.4 → 3.5 → 3.6 → 3.7 → 3.8 → 3.9 → 3.10
