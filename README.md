# FreeWay VPN Backend

Бэкенд **Telegram-first** VPN SaaS: основной продукт — Telegram-бот; оплата — **ЮKassa**; провижининг и источник подписки — **3x-ui**; бизнес-состояние — **PostgreSQL**. Клиенты: **Happ** (iOS) и **v2RayTun** (Android) — см. [`docs/SUPPORTED_CLIENTS.md`](docs/SUPPORTED_CLIENTS.md).

## Сборка и тесты

```bash
go build ./...
go test ./...
```

Docker-образ собирает бинарники `api`, `bot`, `web`, `migrator` ([`Dockerfile`](Dockerfile)); `config.yaml` копируется в образ, переменные — из `.env` (см. [`docker-compose.yml`](docker-compose.yml)).

## Архитектура

| Слой | Назначение |
|------|------------|
| `internal/handler` | HTTP (chi), Telegram-бот, веб-панель менеджера |
| `internal/usecase` | Подписки, платежи, пользователи, статистика, выдача подписки |
| `internal/repository` | GORM ↔ PostgreSQL |
| `internal/infrastructure` | **xui** (3x-ui), yookassa, telegramnotify, vlessconfig, … |

| Команда | Роль |
|---------|------|
| [`cmd/api`](cmd/api) | REST, `GET /sub/{token}`, JWT, webhook ЮKassa, **встроенные воркеры** (платежи, подписки, ноды, routing) |
| [`cmd/bot`](cmd/bot) | Пользовательский сценарий + `/manager` для `TELEGRAM_ADMIN_IDS` |
| [`cmd/web`](cmd/web) | HTML/htmx панель (cookie + CSRF), порт из `config.yaml` |

Стек: **Go 1.24**, PostgreSQL, GORM, chi, go-telegram/bot, 3x-ui (HTTP), YooKassa.

## Быстрый старт

1. `cp .env.example .env` — заполните `DATABASE_DSN`, `JWT_SECRET`, `TELEGRAM_BOT_TOKEN`, `BASE_URL`, `YOOKASSA_*`, при необходимости **`XUI_*`** и `ADMIN_WEB_TOKEN`.
2. `docker compose up --build`
3. `curl http://localhost:8080/health` и `curl http://localhost:8080/health/ready`
4. Веб-панель: `http://localhost:3000/admin/login` (токен = `ADMIN_WEB_TOKEN`)

В Docker для API/бота укажите **`DATABASE_DSN`** с хостом **`postgres`**, не `localhost`.

## Документация

| Файл | Содержание |
|------|------------|
| [`docs/SETUP.md`](docs/SETUP.md) | Требования, **полный список env**, Docker Compose, 3x-ui, типовые ошибки, **ограничения текущей версии** |
| [`docs/SUPPORTED_CLIENTS.md`](docs/SUPPORTED_CLIENTS.md) | Happ и v2RayTun, формат подписки |
| [`docs/MIGRATION_TO_3XUI_REPORT.md`](docs/MIGRATION_TO_3XUI_REPORT.md) | Миграция с Remnawave, **финальный аудит** |

## Готовность и пробелы (кратко)

- **Готово**: компиляция, интеграция 3x-ui (`VPNPanelClient`), выдача `/sub/{token}` с fallback на VLESS из БД, бот (меню Happ/v2RayTun, ключи, поддержка, оплата), веб-панель (дашборд, пользователи, ноды, платежи, routing), ЮKassa, рефералы.
- **Не готово / частично**: тарифы и цены только в коде (`payment_usecase`); веб-«Настройки» продукта нет; в боте `/manager` пункт «Настройки» — заглушка; health 3x-ui не отображается как отдельный виджет; диплинк `v2raytun://` нужно проверить на реальном приложении.
- **Проверить в первую очередь**: логин в 3x-ui и `addClient`/`updateClient` на стенде; совпадение `XUI_PUBLIC_SUBSCRIPTION_BASE_URL` + `XUI_SUBSCRIPTION_PATH` с настройками subscription в панели; полный сценарий оплаты ЮKassa + webhook; импорт подписки в Happ и v2RayTun по URL с вашего `BASE_URL`.

Подробнее — раздел **«Финальный аудит репозитория»** в [`docs/MIGRATION_TO_3XUI_REPORT.md`](docs/MIGRATION_TO_3XUI_REPORT.md).
