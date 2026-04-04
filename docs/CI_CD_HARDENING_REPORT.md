# Отчёт: production hardening CI/CD

Дата: апрель 2026 (итерация hardening).

## 1. Найденные проблемы

| Проблема | Риск |
|----------|------|
| Job `deploy` зависел только от `docker`; явная связь с `go` была только через косвенную цепочку | Путаница при будущих изменениях графа jobs |
| Не было шага миграции БД перед поднятием нового кода | `cmd/migrator` был заглушкой; схема применялась только при старте процессов (порядок и предсказуемость хуже, чем явный шаг) |
| Не было внешней проверки health после выката | Битый деплой мог остаться незамеченным до пользователей |
| Документация и compose в репозитории ориентировались на `build: .`, тогда как CI публикует GHCR | Расхождение «как в CI» vs «как на сервере» |
| Не было сканирования уязвимостей зависимостей в CI | Меньше сигналов до продакшена |
| Жёсткая привязка версий actions только по major | Теоретический supply-chain риск при теге, перезаписанном злоумышленником (низкая вероятность для доверенных org) |

## 2. Внесённые исправления

### CI workflow (`.github/workflows/ci.yml`)

- **`deploy.needs: [go, docker]`** — деплой явно ждёт и тесты, и образ.
- **`golang/govulncheck-action@v1`** после `go mod verify` — проверка известных уязвимостей в зависимостях (падает job при находках по политике govulncheck).
- **SSH-скрипт**: логи `[deploy]`, `docker compose pull` → **`docker compose run --rm api /app/migrator`** → **`docker compose up -d`**, затем `docker compose ps`; таймаут SSH `command_timeout: 30m`.
- **Пост-деплой**: шаг на runner с `curl` на публичные URL (см. secrets ниже); при отсутствии `DEPLOY_HEALTH_API_READY_URL` — **warning** и пропуск (не ломает пайплайн для старых конфигов).
- Комментарий в начале файла про **pinning actions по SHA** как следующий шаг.

### Миграции (`cmd/migrator`)

- Реализован рабочий **migrator**: загрузка `config.yaml`, проверка `DATABASE_DSN`, вызов **`database.Connect`** (GORM AutoMigrate, индексы, идемпотентный seed/catalog/topology — как при старте API).
- В **`config`**: **`ValidateMigrator()`** — только DSN.

### Docker Compose

- Сервис **`migrator`** с **`profiles: [tools]`** для локального одноразового запуска:  
  `docker compose --profile tools run --rm migrator`
- Файл **`docker-compose.prod.example.yml`**: пример стека с **`image: ghcr.io/OWNER/REPO:latest`** и комментариями про порядок pull → migrator → up.

### Документация

- **`docs/SETUP.md`**: обновлены CI/CD, миграции, health, secrets, модель деплоя, откат; уточнено описание migrator.
- **`README.md`**: блок «готовность» приведён в соответствие с тарифами в БД и CI.

## 3. Модель деплоя (как задумано сейчас)

1. **GitHub Actions** собирает и пушит образ в **GHCR** (`main` / тег `v*`).
2. На сервере в **`DEPLOY_PATH`** лежит `docker-compose.yml` (или свой файл), где сервисы **`api` / `bot` / `web`** указывают на **тот же образ** из GHCR (не `build: .`).
3. Скрипт по SSH:
   - `docker compose pull`
   - `docker compose run --rm api /app/migrator` — схема БД до перезапуска приложений
   - `docker compose up -d --remove-orphans`
4. С **runner** GitHub (не с сервера) вызываются публичные **`GET /health/ready`** (API) и опционально **`GET /health`** (web), если заданы secrets.

**Важно:** имя сервиса в compose для переопределения команды должно быть **`api`** (как в примерах). Если переименовано — поправьте workflow или скрипт.

## 4. Secrets (деплой + health)

| Secret | Назначение |
|--------|------------|
| Уже были | `DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_SSH_KEY`, `DEPLOY_PATH`, опционально `DEPLOY_GHCR_*` |
| **Рекомендуется** | `DEPLOY_HEALTH_API_READY_URL` — полный URL, например `https://api.example.com/health/ready` |
| Опционально | `DEPLOY_HEALTH_WEB_URL` — например `https://panel.example.com/health` |

## 5. Оставшиеся риски

- **AutoMigrate** не заменяет версионируемые SQL-миграции: откат схемы «назад» не автоматизирован; опасные изменения моделей требуют ручного плана.
- **govulncheck** может давать ложные срабатывания или шум; при необходимости сузьте scope или временно зафиксируйте версию action.
- **Health с runner** проверяет только публичную доступность; внутренняя сеть / firewall может отличаться от реальности клиентов.
- **Пиннинг по SHA** для third-party actions не внедрён — остаётся рекомендацией.

## 6. Зависимости и govulncheck

Для «зелёного» **govulncheck** в CI обновлены: **Go 1.25.8** (`go.mod`, образ `golang:1.25-alpine`), **github.com/golang-jwt/jwt/v5 v5.2.2**, **github.com/jackc/pgx/v5 v5.5.4** и транзитивные `golang.org/x/*`. Локальный запуск `govulncheck` требует установленного toolchain не старше `go.mod`.

## 7. Рекомендуемые следующие шаги

1. Задать **`DEPLOY_HEALTH_API_READY_URL`** (и при необходимости web) в production.
2. Включить **protection rules** для environment **`production`** (required reviewers).
3. Добавить отдельный workflow для ветки **`develop`** / **`staging`** (только `go` + сборка Docker без push или в отдельный registry).
4. Рассмотреть **Trivy** / **Grype** для образа (отдельный job, при желании `continue-on-error` на переходный период).
5. Зафиксировать **digest** образа в compose при релизе (immutable deploy).

## 8. Откат (кратко)

См. раздел в **`docs/SETUP.md`**. Суть: откатить **тег образа** в compose на предыдущий, выполнить `pull` и `up`; миграции схемы «назад» при необходимости — вручную в БД (AutoMigrate вперёд-совместим в типичных случаях, но не гарантирует откат).
