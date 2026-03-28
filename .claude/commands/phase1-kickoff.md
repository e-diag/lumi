---
name: phase1-kickoff
description: "Запускает реализацию Фазы 1: бэкенд ядро FreeWay VPN"
---

# Запуск Фазы 1 — Бэкенд ядро

Выполни следующие шаги строго по порядку. После каждого шага сообщай что создано и что дальше.

## Шаг 1: Инициализация проекта

Создай структуру директорий и go.mod:

```bash
mkdir -p cmd/{api,bot,web,migrator}
mkdir -p internal/{domain,usecase,repository,handler/{api/middleware,bot,web/templates,web/static},worker,infrastructure/{config,database,remnawave,yookassa,xray}}
mkdir -p migrations
```

go.mod с зависимостями:
- github.com/go-chi/chi/v5 v5.1.0
- github.com/go-telegram/bot v1.18.0
- github.com/golang-jwt/jwt/v5 v5.2.1
- github.com/google/uuid v1.6.3
- gorm.io/gorm v1.25.5
- gorm.io/driver/postgres v1.5.4
- gopkg.in/yaml.v3 v3.0.1
- golang.org/x/crypto v0.14.0
- github.com/stretchr/testify v1.8.4
- github.com/stretchr/mock v1.6.0

## Шаг 2: Domain-модели

Создай все файлы в internal/domain/:
- user.go (User struct + NewUser constructor + sentinel errors)
- subscription.go (Subscription + SubscriptionTier + TierLimitsMap + IsActive() + DaysLeft())
- node.go (Node + NodeRegion + NodeTransport)
- payment.go (Payment struct)
- routing_rule.go (RoutingRule struct)
- migration.go (MigrationRecord struct — заглушка для Фазы 6)
- errors.go (все sentinel errors: ErrUserNotFound, ErrSubscriptionNotFound, etc.)

## Шаг 3: Infrastructure — Config и Database

Создай:
- internal/infrastructure/config/config.go — загрузка из config.yaml + env подстановка
- internal/infrastructure/database/database.go — GORM PostgreSQL + AutoMigrate + seed нод
- config.yaml — шаблон с ${ENV_VAR} плейсхолдерами
- .env.example — все переменные окружения с описаниями

## Шаг 4: Repository интерфейсы и реализации

Создай internal/repository/interfaces.go с интерфейсами:
- UserRepository
- SubscriptionRepository  
- NodeRepository
- PaymentRepository
- RoutingRepository

Реализуй каждый в отдельном файле через GORM.

## Шаг 5: UseCase интерфейсы и реализации

Создай internal/usecase/interfaces.go с интерфейсами:
- UserUseCase
- SubscriptionUseCase
- NodeUseCase
- ConfigUseCase (генерация subscription URL)
- StatsUseCase

Реализуй ConfigUseCase полностью — это приоритет #1:
- GenerateSubscription(userUUID, nodes, tier) string
- generateVLESSReality(uuid, node) string
- generateVLESSWebSocket(uuid, node) string

## Шаг 6: HTTP API сервер

Создай cmd/api/main.go:
- chi router с middleware (JWT auth, rate limit, CORS, request logging через slog)
- Graceful shutdown через context + os.Signal
- Все endpoints как заглушки (200 + {"status": "TODO"})

Реализуй полностью sub_handler.go:
- GET /sub/{token}
- Поиск user по sub_token
- Определение тарифа → allowed nodes
- Генерация конфигов через ConfigUseCase
- Возврат base64 с правильными заголовками (profile-title, profile-update-interval: 24)

## Шаг 7: Docker

Создай:
- Dockerfile (multi-stage: builder + alpine, компилирует все 4 бинаря)
- docker-compose.yml (postgres + api + bot + web + pgadmin для разработки)
- .gitignore (включая .env, *.key, binary files)

## Шаг 8: Первичная проверка

Выполни:
```bash
go build ./...
go vet ./...
go test ./...
docker-compose up -d postgres
go run ./cmd/api
```

Проверь что GET /sub/{test-token} возвращает валидный base64 с VLESS конфигами.
Импортируй subscription URL в Hiddify и убедись что конфиги видны.

---

После завершения всех шагов выведи:
1. Список созданных файлов
2. Результаты go build и go test
3. Пример subscription URL (без реальных секретов)
4. Что нужно для перехода к Фазе 2
