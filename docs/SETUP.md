# FreeWay VPN — установка и запуск

Документ описывает развёртывание бэкенда (REST API, Telegram-бот, веб-панель), переменные окружения и типовые проблемы.

---

## 1. Обзор проекта

Монорепозиторий на **Go 1.24**: Telegram-first VPN SaaS — Clean Architecture (`handler` → `usecase` → `repository`), PostgreSQL (GORM), **3x-ui** (провижининг и подписка), ЮKassa, фоновые воркеры (платежи, подписки, здоровье нод, routing).

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
| `TELEGRAM_APP_URL_IOS` | Ссылка на Happ в App Store (можно пусто) |
| `TELEGRAM_APP_URL_ANDROID` | Ссылка на v2RayTun в Google Play / RuStore (можно пусто) |
| `TELEGRAM_SUPPORT_URL` | Ссылка для раздела «Поддержка» в боте (можно пусто) |

В `config.yaml` также задаются `max_trials_per_ip`, `referral_bonus_max_per_month`, `payment_default_days` (подставляются из YAML; при необходимости вынесите в ENV отдельным расширением конфига).

### 3x-ui (панель и subscription)

| Переменная | Описание |
|------------|----------|
| `XUI_BASE_URL` | Корень веб-панели 3x-ui **вместе с path** из настроек «URI Path» (пример: `https://host:2053` или `https://host:2053/panel`). |
| `XUI_USERNAME` | Логин администратора панели |
| `XUI_PASSWORD` | Пароль |
| `XUI_INBOUND_ID` | Числовой ID inbound (список Inbounds в панели). Перекрывает `inbound_id` из `config.yaml`, если задан. |
| `XUI_PUBLIC_SUBSCRIPTION_BASE_URL` | Публичный URL **subscription-сервера** 3x-ui (часто отдельный порт), без завершающего `/`. Пример: `https://sub.example.com:2096`. |
| `XUI_SUBSCRIPTION_PATH` | Сегмент пути перед `subId` (в панели: Settings → Subscription), по умолчанию `sub`. |

Если **`XUI_BASE_URL` пустой**, API и бот **не** вызывают панель: подписка и `device_limit` обновляются только в PostgreSQL. В логах: `bootstrap: XUI base_url is empty`.

Для выдачи подписки пользователю в приложениях **Happ** и **v2RayTun** backend проксирует контент из 3x-ui: при наличии `PanelSubID` у пользователя и `XUI_PUBLIC_SUBSCRIPTION_BASE_URL` выполняется `GET {base}/{path}/{subId}`; иначе используется локальная генерация VLESS из таблицы `nodes` (fallback).

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

Поднимутся: **postgres**, **api**, **bot**, **web**. Порты: `5432`, `8080`, `3000`. Все три бинаря читают один **`config.yaml`** из образа и переменные из **`.env`** (`env_file` в compose); для API в Docker задайте **`DATABASE_DSN`** с хостом **`postgres`** (см. ниже).

Профиль **dev** (pgAdmin): `docker compose --profile dev up -d`.

Проверка API:

```bash
curl -s http://localhost:8080/health
curl -s http://localhost:8080/health/ready
```

`/health` — процесс жив. `/health/ready` — дополнительно **ping PostgreSQL** (для orchestrator/k8s).

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
go test ./...
```

### API (примеры)

- `GET /health` — без авторизации.
- `GET /health/ready` — готовность + БД.
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
| 3x-ui ошибки при продлении | `XUI_*`, inbound_id, логин в панель, клиент создан в inbound |
| Платёж `succeeded` в БД, подписка не продлилась (редкий сбой между резервом и активацией) | В таблице `payment_activations` удалите строку с `payment_id` этого платежа и дождитесь повторной доставки webhook или проверьте воркером |

---

## 8. Продакшен

- Все секреты только в **переменных окружения** или секрет-хранилище; не коммитьте `.env`.
- `JWT_SECRET`, `ADMIN_WEB_TOKEN`, ключи ЮKassa и пароль 3x-ui — уникальные длинные значения.
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
cmd/migrator     # заглушка / будущие миграции
internal/bootstrap   # Общая сборка DI
internal/domain
internal/handler
internal/infrastructure
internal/repository
internal/usecase
internal/worker
config.yaml      # Шаблон конфигурации (${ENV} подстановка)
docs/            # Документация
```

Схема БД создаётся через **GORM AutoMigrate** при старте (`internal/infrastructure/database`), отдельная папка `migrations/` в репозитории не используется.

## 10. Ограничения текущей версии (после 3x-ui)

| Область | Статус |
|---------|--------|
| Веб-панель менеджера | Есть: дашборд, пользователи (поиск, выдача/отзыв), ноды, платежи, routing, **тарифы** (`/admin/plans`), **настройки продукта** (`/admin/settings`), **каталог серверов** (`/admin/servers`). Нет отдельной кнопки «переотправить ключ» (ключ — тот же `GET /sub/{token}`). |
| Бот `/manager` | Статистика, ноды, финансы, рассылка, роутинг, выдача/отзыв по callback; «Настройки» — заглушка; пользователи — команда `/user` с Telegram ID. |
| 3x-ui | Один `inbound_id`; нет health-check панели в отчёте веб-UI (ноды по-прежнему из БД). |

---

## 11. GitHub Actions (CI/CD)

Файл [`.github/workflows/ci.yml`](../.github/workflows/ci.yml), workflow **CI / CD**.

### Цепочка (порядок жёсткий)

1. **Go** — `go mod verify`, `go vet`, `go test`, сборка всех `cmd/*`. При падении тестов дальше ничего не выполняется.
2. **Docker** — сборка образа; на PR **без** push в реестр; на `main` / тег `v*` — push в **GHCR**.
3. **Deploy** — только если задан secret `DEPLOY_HOST` и образ реально ушёл в реестр (не PR). Выполняется **после** успешных шагов 1–2.

### События

| Событие | Тесты + образ | Деплой на сервер |
|---------|---------------|------------------|
| **Pull request** в `main` | Да, без push в GHCR | Нет |
| **Push** в `main` | Push `latest` + SHA | Да, если настроены secrets |
| **Push** тега `v*` | Push semver-тегов | Да, если настроены secrets |
| **workflow_dispatch** | Push (если ветка не PR-сценарий) | Только если включён чекбокс **deploy** и ref = `main` или тег `v*` |

### Secrets репозитория (деплой)

Задаются в **Settings → Secrets and variables → Actions** (не в `.env` на ноутбуке):

| Secret | Обязательно | Описание |
|--------|-------------|----------|
| `DEPLOY_HOST` | Да, чтобы деплой работал | IP или hostname сервера |
| `DEPLOY_USER` | Да | SSH-пользователь |
| `DEPLOY_SSH_KEY` | Да | Приватный ключ (весь PEM, включая `BEGIN` / `END`) |
| `DEPLOY_PATH` | Да | Каталог на сервере с `docker-compose.yml` (например `/opt/freeway`) |
| `DEPLOY_GHCR_USER` | Рекомендуется | Логин GitHub для `docker login ghcr.io` на сервере |
| `DEPLOY_GHCR_TOKEN` | Рекомендуется | [PAT](https://github.com/settings/tokens) с правом **read:packages** (или классический `read:packages`) |

Если `DEPLOY_GHCR_*` не заданы, на сервере должен быть уже выполнен однократный `docker login ghcr.io`, иначе `docker compose pull` не подтянет приватный образ.

Окружение GitHub **production** создаётся при первом деплое; при желании включите в нём [required reviewers](https://docs.github.com/actions/deployment/targeting-different-environments/using-environments-for-deployment) для ручного подтверждения выката.

SSH по умолчанию **порт 22**. Нестандартный порт: добавьте в шаг `appleboy/ssh-action` в workflow параметр `port` (или настройте доступ с bastion).

### Образ на сервере

В `docker-compose` на сервере укажите образ из GHCR вместо `build: .`:

```yaml
image: ghcr.io/your-org/your-repo:latest
```

Имя: `ghcr.io/<владелец>/<репозиторий>` в нижнем регистре. Подробнее: [Container registry](https://docs.github.com/packages/working-with-a-github-packages-registry/working-with-the-container-registry).

### Ручной выкат без нового коммита

**Actions → CI / CD → Run workflow**: ветка `main` (или тег), включить **deploy** — после сборки и push образа выполнится тот же SSH-скрипт `compose pull && up`.
