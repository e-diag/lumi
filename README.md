# FreeWay VPN Backend

Бэкенд для VPN/proxy SaaS: API, Telegram-бот, веб-панель менеджера, подписки и платежи.

## Что в репозитории

- `cmd/api` — REST API и фоновые воркеры.
- `cmd/bot` — Telegram-бот (пользовательский и менеджерский флоу).
- `cmd/web` — веб-панель менеджера.
- `internal/*` — домен, usecase, репозитории, хендлеры, инфраструктурные адаптеры.
- `migrations` — миграции БД.

Технологии: Go, PostgreSQL, GORM, chi, YooKassa, Remnawave/Xray.

## Быстрый старт

1. Скопируйте `.env.example` в `.env` и заполните значения.
2. Поднимите сервисы:

```bash
docker compose up --build
```

3. Проверьте API:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/health/ready
```

## Полная инструкция

Подробный запуск, переменные окружения, локальный режим, Docker и troubleshooting:

- [`docs/SETUP.md`](docs/SETUP.md)