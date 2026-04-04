# FreeWay VPN Backend

Бэкенд **Telegram-first** VPN SaaS: основной продукт — Telegram-бот; оплата в боте/API — **ЮKassa** (для бота ключи и триал работают и без ЮKassa); провижининг и источник подписки — **3x-ui**; бизнес-состояние — **PostgreSQL**. Клиенты: **Happ** (iOS) и **v2RayTun** (Android) — см. [`docs/SUPPORTED_CLIENTS.md`](docs/SUPPORTED_CLIENTS.md).

## Сборка и тесты

```bash
go build ./...
go test ./...
```

Docker-образ собирает бинарники `api`, `bot`, `web`, `migrator` ([`Dockerfile`](Dockerfile)); `config.yaml` копируется в образ, переменные — из `.env` (см. [`docker-compose.yml`](docker-compose.yml)). Для продакшена без публикации Postgres наружу используйте [`docker-compose.prod.example.yml`](docker-compose.prod.example.yml).

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

Стек: **Go 1.25.8** (см. `go.mod`), PostgreSQL, GORM, chi, go-telegram/bot, 3x-ui (HTTP), YooKassa.

## Быстрый старт

1. `cp .env.example .env` — заполните `DATABASE_DSN`, `JWT_SECRET`, `TELEGRAM_BOT_TOKEN`, `BASE_URL`; для API и оплаты нужны **`YOOKASSA_*`** (для одного только бота без оплаты в чате их можно не задавать); при необходимости **`XUI_*`** и `ADMIN_WEB_TOKEN`.
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
| [`docs/CI_CD_HARDENING_REPORT.md`](docs/CI_CD_HARDENING_REPORT.md) | CI/CD: govulncheck, мигратор, health после деплоя |
| [`docs/PRODUCTION_REVIEW_FIX_REPORT.md`](docs/PRODUCTION_REVIEW_FIX_REPORT.md) | Production hardening: безопасность логов, compose, лимиты списков, честные пробелы |

## Готовность и пробелы (кратко)

- **Готово**: компиляция, интеграция 3x-ui (`VPNPanelClient`), выдача `/sub/{token}` с fallback на VLESS из БД, бот (меню Happ/v2RayTun, ключи, поддержка, оплата по тарифам из БД), веб-панель (дашборд, пользователи, ноды, платежи, routing, тарифы/настройки/каталог серверов), ЮKassa, рефералы, **CI/CD** (тесты, образ GHCR, SSH-деплой, опционально внешний health).
- **Не готово / частично**: в боте `/manager` пункт «Настройки» — заглушка; health 3x-ui не отображается отдельным виджетом; диплинк `v2raytun://` проверить на реальном клиенте; схема БД — **GORM AutoMigrate**, не SQL-миграции с откатом.
- **Проверить в первую очередь**: логин в 3x-ui и `addClient`/`updateClient` на стенде; совпадение `XUI_PUBLIC_SUBSCRIPTION_BASE_URL` + `XUI_SUBSCRIPTION_PATH` с настройками subscription в панели; полный сценарий оплаты ЮKassa + webhook; импорт подписки в Happ и v2RayTun по URL с вашего `BASE_URL`.

Подробнее — раздел **«Финальный аудит репозитория»** в [`docs/MIGRATION_TO_3XUI_REPORT.md`](docs/MIGRATION_TO_3XUI_REPORT.md).
