# FreeWay VPN — Claude Code Project Context

## Проект

**FreeWay** — VPN-сервис для России. Один клик — всё работает.
Бэкенд на Go (Clean Architecture). Отдельный репозиторий от MTProto-бота.

## Стек

- **Language**: Go 1.24
- **ORM**: GORM + PostgreSQL
- **Router**: chi v5
- **Telegram-бот**: go-telegram/bot
- **JWT**: golang-jwt/jwt v5
- **VPN-ядро**: Xray-core (управление через Remnawave API)
- **CDN**: Яндекс Cloud CDN (WebSocket fallback при белых списках)
- **Оплата**: ЮKassa
- **Клиент**: Hiddify Next (форк, Flutter)
- **Магазин**: RuStore (старт)

## Архитектура

```
cmd/api/       → REST API :8080
cmd/bot/       → Telegram-бот менеджера
cmd/web/       → Веб-панель :3000 (Go templates + htmx)
cmd/migrator/  → ФАЗА 6: миграция с MTProto-бота (реализуется ПОСЛЕДНЕЙ)

internal/domain/      → сущности
internal/usecase/     → бизнес-логика (интерфейсы + реализации)
internal/repository/  → слой БД
internal/handler/     → HTTP + bot + web handlers
internal/worker/      → фоновые задачи
internal/infrastructure/ → config, db, remnawave, yookassa, xray
```

## Правила Clean Architecture (СТРОГО)

- handler → usecase → repository. Никаких прямых вызовов DB из handler.
- Все usecase определяются через интерфейсы в `usecase/interfaces.go`
- Все repository определяются через интерфейсы в `repository/interfaces.go`
- Зависимости инжектируются через конструкторы, не глобальные переменные

## Ноды VPN

- **EU-NL** (Hetzner, Нидерланды): основная, VLESS+Reality, порт 443
- **USA** (Vultr/DO): AI-сервисы, VLESS+Reality, порт 443
- **CDN-Yandex**: Яндекс Cloud CDN, VLESS+WebSocket+TLS — резерв при белых списках

## Тарифы

```
Free    → 1 Мбит/с, 1 устройство, только EU
Basic   → 149 ₽/мес, 10 Мбит/с, 2 устройства, EU+USA
Premium → 299 ₽/мес, без лимитов, 5 устройств, EU+USA+CDN
```

## Порядок реализации фаз

```
Фаза 1: Бэкенд ядро (domain, usecase, repo, API, /sub/{token})
Фаза 2: Платежи ЮKassa + подписки + workers
Фаза 3: Telegram-бот + веб-панель менеджера
✅ Фаза 4 — Маршрутизация и CDN (ЗАВЕРШЕНА)
Фаза 5: Мобильное приложение (Hiddify форк)
Фаза 6: Миграция MTProto-бота (ТОЛЬКО ПОСЛЕ ГОТОВОГО ПРОДУКТА)
```

**ВАЖНО**: Никогда не переходи к следующей фазе, пока текущая не протестирована.

## Приоритет #1 — GET /sub/{token}

Этот endpoint — сердце системы. Без него ничего не работает.
Возвращает base64(VLESS конфиги через \n). Hiddify импортирует автоматически.

## Миграция MTProto-бота (Фаза 6 — детали)

**Маппинг тарифов:**
- MTProto Free → FreeWay Free
- MTProto Pro → FreeWay Basic (остаток дней сохраняется 1:1)
- MTProto Premium → FreeWay Premium (остаток дней × 2)

**Механизм:**
1. cmd/migrator читает БД старого бота (read-only подключение)
2. Создаёт пользователей в FreeWay с сохранением TelegramID
3. Активирует подписки с расчётом дней (Pro: daysLeft, Premium: daysLeft×2)
4. Добавляет пользователей в Remnawave
5. Создаёт MigrationRecord{Status: "pending"}
6. Отдельно: батчевая рассылка уведомлений через старый MTProto-бот (30 msg/sec)
7. Старый прокси-бот работает параллельно ещё 30 дней

## Секреты

Все через переменные окружения. Никаких секретов в коде.
Файл `.env` в .gitignore. В коде только config.yaml с ${ENV_VAR} плейсхолдерами.

## Команды разработки

```bash
# Запуск всего локально
docker-compose up -d

# Только API
go run ./cmd/api

# Только бот
go run ./cmd/bot

# Только веб-панель
go run ./cmd/web

# Тесты
go test ./...

# Линтер
golangci-lint run
```

## Стиль кода

- Все публичные функции и типы — с комментариями на русском
- Ошибки: всегда оборачивай через fmt.Errorf("usecase: %w", err)
- Логирование: log/slog (стандартная библиотека Go 1.21+)
- Тесты: table-driven tests, покрытие usecase-слоя обязательно
- Никаких panic() в продакшен-коде
