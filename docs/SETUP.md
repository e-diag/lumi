# FreeWay VPN — установка и запуск

Документ описывает развёртывание бэкенда (REST API, Telegram-бот, веб-панель), переменные окружения и типовые проблемы.

---

## 1. Обзор проекта

Монорепозиторий на **Go 1.24**: Clean Architecture (`handler` → `usecase` → `repository`), PostgreSQL (GORM), ЮKassa, Remnawave/Xray, фоновые воркеры (платежи, подписки, здоровье нод, routing).

| Компонент | Пакет | Порт | Назначение |
|-----------|--------|------|------------|
| API | `cmd/api` | 8080 | REST, `GET /sub/{token}`, JWT, платежи, webhook |
| Бот | `cmd/bot` | — | Telegram (go-telegram/bot) |
| Веб-панель | `cmd/web` | 3000 | HTML/htmx для менеджеров |
| Мигратор | `cmd/migrator` | — | Фаза 6 (заглушка) |

Конфигурация: **`config.yaml`** в корне с плейсхолдерами `${ENV_VAR}`. Путь переопределяется **`CONFIG_PATH`**.

---

## 2. Требования

- **Go** 1.24+
- **Docker** и Docker Compose v2 (для полного стенда)
- **PostgreSQL** 16 (локально или только через Compose)

---

## 3. Переменные окружения

Скопируйте `.env.example` → `.env` и заполните значения.

### База данных

| Переменная | Описание |
|------------|----------|
| `DATABASE_DSN` | DSN PostgreSQL, например `postgres://freeway:password@localhost:5432/freeway?sslmode=disable`. В **docker-compose** хост БД — **`postgres`**, не `localhost`. |

### Сервер и JWT

| Переменная | Описание |
|------------|----------|
| `BASE_URL` | Публичный URL API (ссылки возврата ЮKassa, тексты бота). Пример: `https://api.example.com` |
| `JWT_SECRET` | Секрет HMAC для JWT (минимум **16** символов; в продакшене — длинная случайная строка). |

### Telegram

| Переменная | Описание |
|------------|----------|
| `TELEGRAM_BOT_TOKEN` | Токен от @BotFather |
| `TELEGRAM_BOT_USERNAME` | Имя бота без `@` |
| `TELEGRAM_ADMIN_IDS` | ID менеджеров через запятую (доступ к /manager в боте) |
| `TELEGRAM_APP_URL_IOS` | Ссылка на App Store (можно пусто) |
| `TELEGRAM_APP_URL_ANDROID` | Ссылка на RuStore/Google Play (можно пусто) |

В `config.yaml` также задаются `max_trials_per_ip`, `referral_bonus_max_per_month`, `payment_default_days` (подставляются из YAML; при необходимости вынесите в ENV отдельным расширением конфига).

### Remnawave

| Переменная | Описание |
|------------|----------|
| `REMNAWAVE_BASE_URL` | Базовый URL API панели |
| `REMNAWAVE_API_KEY` | API-ключ |

Для **cmd/web** Remnawave не подключается (только списки/статистика в БД); для API и бота клиент используется в `SubscriptionUseCase`.

### ЮKassa

| Переменная | Описание |
|------------|----------|
| `YOOKASSA_SHOP_ID` | Идентификатор магазина |
| `YOOKASSA_SECRET_KEY` | Секретный ключ |

Обязательны для **cmd/api** и **cmd/bot** (валидация при старте). Для тестов ЮKassa используйте [тестовый магазин](https://yookassa.ru/developers/payment-acceptance/testing-and-going-live/testing) и те же переменные.

### Ноды (плейсхолдеры в seed БД)

После первого запуска миграции в таблицу нод попадают строки с `${NODE_*}` — замените через SQL/панель или обновите записи:

| Переменная | Назначение |
|------------|------------|
| `NODE_EU_HOST`, `NODE_EU_PUBLIC_KEY`, `NODE_EU_SHORT_ID` | EU Reality |
| `NODE_USA_HOST`, `NODE_USA_PUBLIC_KEY`, `NODE_USA_SHORT_ID` | USA Reality |
| `NODE_CDN_HOST`, `NODE_CDN_SNI` | CDN gRPC |

### Веб-панель

| Переменная | Описание |
|------------|----------|
| `ADMIN_WEB_TOKEN` | Секрет для входа в панель (минимум **12** символов). |

### Docker Compose (опционально)

| Переменная | Описание |
|------------|----------|
| `POSTGRES_PASSWORD` | Пароль пользователя `freeway` (по умолчанию в compose: `password`) |
| `PGADMIN_PASSWORD` | Пароль pgAdmin (профиль `dev`) |

### Запуск веб-панели в Docker

| Переменная | Описание |
|------------|----------|
| `WEB_TEMPLATE_DIR` | Каталог шаблонов `*.html`. В образе: `/app/internal/handler/web/templates` (уже задано в compose). Локально можно не задавать — используется `internal/handler/web/templates` относительно cwd. |

---

## 4. Локальный запуск (без Docker)

1. Клонируйте репозиторий.
2. Установите PostgreSQL, создайте БД и пользователя `freeway` (или свои — пропишите в DSN).
3. `cp .env.example .env` и заполните переменные.
4. Убедитесь, что `config.yaml` читается (или `export CONFIG_PATH=...`).
5. Запуск сервисов в отдельных терминалах:

```bash
go run ./cmd/api
go run ./cmd/bot
go run ./cmd/web
```

API слушает `server.host`:`server.port` из конфига (по умолчанию `0.0.0.0:8080`), веб — `web.host`:`web.port` (например `:3000`).

---

## 5. Docker Compose (всё сразу)

Из корня репозитория (рядом `.env` и `docker-compose.yml`):

```bash
docker compose up --build
```

Поднимутся: **postgres**, **api**, **bot**, **web**. Порты: `5432`, `8080`, `3000`.

Профиль **dev** (pgAdmin): `docker compose --profile dev up -d`.

Проверка API:

```bash
curl -s http://localhost:8080/health
```

Проверка веб-панели:

```bash
curl -s http://localhost:3000/health
```

**Важно:** в `.env` для compose укажите `DATABASE_DSN` с хостом **`postgres`**, например:

`postgres://freeway:password@postgres:5432/freeway?sslmode=disable`

(пароль должен совпадать с `POSTGRES_PASSWORD` в compose, если вы его меняли).

---

## 6. Как тестировать

### Юнит-тесты

```bash
go test ./internal/... ./cmd/...
```

### API (примеры)

- `GET /health` — без авторизации.
- `GET /sub/{sub_token}` — выдача подписки (rate limit по IP).
- `POST /api/v1/auth/tg` — тело `{"init_data":"..."}` (Telegram WebApp), rate limit по IP.
- `GET /api/v1/users/me` — заголовок `Authorization: Bearer <jwt>`.
- `GET /api/v1/users/me/subscription` — то же.
- `POST /api/v1/payments/webhook` — тело события ЮKassa (в продакшене только с валидной подписью/сетевой политикой).

### Бот

Напишите боту в Telegram после запуска `cmd/bot` с валидным `TELEGRAM_BOT_TOKEN`.

### Платежи (песочница)

Используйте тестовые ключи ЮKassa, создайте платёж через API или бота, подтвердите тестовой картой из документации ЮKassa. Webhook должен достучаться до `POST .../payments/webhook` (ngrok/публичный URL).

---

## 7. Типовые ошибки

| Симптом | Что проверить |
|---------|----------------|
| `bootstrap failed` / `DATABASE_DSN is required` | `.env`, подстановка в `config.yaml` |
| API не коннектится к БД в Docker | В DSN хост `postgres`, не `localhost`; пароль совпадает с `POSTGRES_PASSWORD` |
| `JWT_SECRET must be at least 16 characters` | Удлините секрет |
| ЮKassa / платежи | `YOOKASSA_*`, `BASE_URL`, доступность webhook с интернета |
| Пустая или битая подписка `/sub/...` | В БД у пользователя `sub_token`, у нод не остались `${...}` в host/key |
| Веб-панель 500 при старте в контейнере | Образ должен содержать шаблоны; `WEB_TEMPLATE_DIR` указывает на каталог с `*.html` |
| Remnawave ошибки при продлении | `REMNAWAVE_*`, пользователь создан в панели |
| Платёж `succeeded` в БД, подписка не продлилась (редкий сбой между резервом и активацией) | В таблице `payment_activations` удалите строку с `payment_id` этого платежа и дождитесь повторной доставки webhook или проверьте воркером |

---

## 8. Продакшен

- Все секреты только в **переменных окружения** или секрет-хранилище; не коммитьте `.env`.
- `JWT_SECRET`, `ADMIN_WEB_TOKEN`, ключи ЮKassa и Remnawave — уникальные длинные значения.
- Включите TLS на reverse proxy (nginx/caddy) перед API и веб-панелью.
- Ограничьте доступ к `POST /api/v1/payments/webhook` по IP ЮKassa или проверке подписи (если добавлена).
- Масштабирование: API stateless; одна БД; воркеры встроены в процесс `cmd/api` — при горизонтальном масштабировании API вынесите воркеры в отдельный `cmd/worker` или используйте leader election (отдельная задача).
- Логи: структурированный **slog** в JSON на stdout — собирайте агентом (Vector, Promtail и т.д.).
- Резервное копирование PostgreSQL по расписанию.

---

## 9. Структура каталогов (ориентир)

```
cmd/api          # HTTP API + воркеры
cmd/bot          # Telegram
cmd/web          # Панель
cmd/migrator     # Фаза 6
internal/bootstrap   # Общая сборка DI
internal/domain
internal/handler
internal/infrastructure
internal/repository
internal/usecase
internal/worker
config.yaml      # Шаблон конфигурации
docs/            # Документация
```
