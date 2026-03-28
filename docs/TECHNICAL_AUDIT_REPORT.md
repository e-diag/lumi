# Технический и продуктовый отчёт: FreeWay VPN (backend)

**Дата подготовки:** 2026-03-28  
**Охват:** фактический код репозитория (`cmd/*`, `internal/*`), включая файлы, которые могут не попадать в индексацию IDE.

---

## 1. Архитектура

**Clean Architecture в целом соблюдена на уровне usecase ↔ repository:** бизнес-логика опирается на интерфейсы из `internal/usecase/interfaces.go` и `internal/repository/interfaces.go`, GORM спрятан в репозиториях.

### Нарушения и жёсткие связи

1. **`SubHandler` обходит usecase и тянет репозиторий напрямую** — для `GET /sub/{token}` используется `repository.UserRepository` + `usecase.ConfigUseCase`. По заявленным правилам handler не должен знать о repository; поиск пользователя по токену логичнее вынести в `UserUseCase` (или отдельный порт).

2. **Инфраструктура через адаптеры к usecase — хорошо:** `yookassa.GatewayAdapter`, `remnawave.UsecaseAdapter` реализуют `PaymentGateway` и `RemnawaveClient` — зависимость инвертирована корректно.

3. **Дублирование логики URI:** генерация VLESS есть в `internal/usecase/config_usecase.go` и отдельно в `internal/infrastructure/xray/config_gen.go` — риск расхождения при правках (сейчас по сути два пути).

4. **Разные «сборки» процессов:** в `cmd/api/main.go` подключены Remnawave и YooKassa; в `cmd/web/main.go` и `cmd/bot/main.go` для `SubscriptionUseCase` и `PaymentUseCase` передаётся **`nil` вместо gateway/remnawave** — панель и бот **не могут** создавать реальные платежи и не синхронизируют срок с панелью так же, как API.

### Итог по архитектуре

Ядро слоёв в порядке, но есть явные отклонения от заявленных правил (sub handler) и **несогласованность DI между процессами** (API vs web/bot).

---

## 2. Реализованные фичи (по коду)

| Фича | Статус | Где | Как работает / замечания |
|------|--------|-----|---------------------------|
| **Telegram WebApp → JWT** | Почти complete | `internal/handler/api/auth_handler.go` | Разбор `init_data`, HMAC по доке Telegram, `UserUseCase.Register`, JWT HS256 ~30 дней. |
| **JWT для API** | Partial | `internal/handler/api/middleware/jwt.go` | Bearer, парсинг `sub`. Нет явной валидации `exp` через registered claims (стоит уточнить/усилить). |
| **Платежи YooKassa (create/status)** | Complete в API | `internal/usecase/payment_usecase.go`, `internal/infrastructure/yookassa/*`, `internal/handler/api/payment_handler.go` | Создание платежа, сохранение в БД, polling через worker + ленивая проверка в `GetPaymentStatus`. |
| **Webhook YooKassa** | Partial | `internal/handler/api/webhook_handler.go` | **Только allowlist IP** ЮKassa, разбор JSON → `HandleWebhook`. **Нет проверки подписи тела** (у ЮKassa есть модель с секретом — здесь опора только на сеть). |
| **Подписки** | Partial | `internal/usecase/subscription_usecase.go` | Активация/продление, `ExpireOld`, даунгрейд в Free, `DeviceLimit` на `User`. **Remnawave: вызывается только `UpdateUserExpiry`**, не `CreateUser` при регистрации/первой активации — риск «пользователя нет на панели». |
| **GET /sub/{token}** | Complete (функционально) | `internal/handler/api/sub_handler.go`, `internal/usecase/config_usecase.go` | Токен → пользователь → тариф по подписке → ноды по регионам → VLESS Reality/gRPC/WS, CDN в конце (сортировка по подстроке `"cdn"` в строке — хрупко). Заголовки под Hiddify. |
| **Routing lists + antifilter** | Complete | `internal/usecase/routing_usecase.go`, `internal/repository/routing_repository.go`, `internal/worker/routing_update_worker.go`, `internal/handler/api/routing_handler.go` | Загрузка списков, сохранение в БД, публичный JSON + кэш 1 ч, `direct_strict_mode` в ответе. **Версия в ответе — дата `time.Now()`, а не версия из БД** — для клиента вводит в заблуждение. |
| **Workers** | Complete как код | `internal/worker/*` | **Payment:** pending >5 мин → опрос API; >24 ч → cancel. **Subscription:** `ExpireOld` + лог «истекает через 3 дня» (без рассылки). **Node health:** TCP dial, 2 фейла → `Active=false`. **Routing:** стартовое обновление если пусто, затем ежедневно ~03:00 UTC. |
| **Ноды** | Partial | `internal/usecase/node_usecase.go`, `internal/repository/node_repository.go` | CRUD-уровень минимальный; выдача в конфиге только **active** нод по региону. Нет отдельного «операторского» API в REST (только веб/бот). |
| **REST User endpoints** | Stub | `internal/handler/api/user_handler.go` | `POST /register`, `GET /me`, `GET /me/subscription` возвращают **`{"status":"TODO"}`** — для клиента это мёртвые маршруты. |
| **Веб-админка** | Complete (htmx) | `internal/handler/web/web_handler.go` | Логин по токену, сессия + CSRF на POST, дашборд, пользователи, ноды, платежи, routing (update/add/delete), grant/revoke. |
| **Telegram-бот** | Complete (менеджерский) | `internal/handler/bot/*.go`, `cmd/bot/main.go` | Статистика, пользователи, ноды, финансы, рассылка (по файлам), routing — на usecase. |
| **Migrator** | Stub | `cmd/migrator/main.go` | Только лог «not implemented». |

---

## 3. API

### Зарегистрированные маршруты (`cmd/api/main.go`)

| Метод | Путь | Защита | Примечание |
|-------|------|--------|------------|
| GET | `/sub/{token}` | Нет | Критичный публичный канал |
| POST | `/api/v1/auth/tg` | Нет | |
| POST | `/api/v1/payments/webhook` | IP ЮKassa | |
| GET | `/api/v1/routing/lists` | Нет | `Throttle(20)` |
| POST | `/api/v1/users/register` | JWT | **Заглушка TODO** |
| GET | `/api/v1/users/me` | JWT | **TODO** |
| GET | `/api/v1/users/me/subscription` | JWT | **TODO** |
| POST | `/api/v1/payments` | JWT | |
| GET | `/api/v1/payments/{id}/status` | JWT | |
| GET | `/health` | Нет | |

### Production-ready

`/health`, `/sub/{token}` (при валидных данных в БД), `/api/v1/auth/tg`, платежи + webhook (с оговорками ниже), `/api/v1/routing/lists`.

### Не production-ready

Все три `/api/v1/users/*` — явные заглушки.

### Валидация и безопасность

- Платёж: `tier` и `days` из JSON **без жёсткой валидации** на handler-уровне (частично в usecase).
- **Rate limit:** в `.cursorrules` заявлено «на все публичные» — **в коде throttle только на routing**. **`/sub/{token}` и `/auth/tg` без лимита** — уязвимы к перебору токенов и нагрузке.
- Webhook: при ошибке бизнес-логики всё равно **200 OK** — правильно для ЮKassa, но скрывает сбои; нужен алертинг по логам.

---

## 4. БД и домен

**Есть:** `User`, `Subscription`, `Node`, `Payment`, `RoutingRule`, `MigrationRecord`, лимиты тарифов в `TierLimitsMap`.

### Для зрелого SaaS не хватает (в коде нет)

- **Устройства / сессии** — `DeviceLimit` есть, но нет таблицы устройств, нет привязки ключа к клиенту, нет отзыва по устройству.
- **Аудит / abuse** — нет учёта запросов на `/sub`, нет банов, нет fraud-паттернов.
- **События биллинга** — одна запись `Payment`; нет явного журнала идемпотентности webhook (кроме статуса платежа).
- **Скорость тарифа (`SpeedMbps`)** — в домене объявлена, **нигде не применяется** в генерации конфига и не передаётся в заголовки (`subscription-userinfo` захардкожен нулями).

---

## 5. Xray / генерация конфигов

- URI собираются в **usecase** из `User.ID` (UUID) как **VLESS UUID** — типичный подход, но **идентичность пользователя = публичный идентификатор в ссылке**; смена UUID без миграции на нодах = операционная боль.
- **Масштабирование:** генерация O(число нод на тариф), без кэша — для тысяч RPS на один инстанс узкое место; для типичного VPN-SaaS обычно достаточно, если не DDoS на `/sub`.
- **Безопасность токена:** `SubToken` — длинный случайный UUID в URL; **нет ротации**, нет «одноразовых» ссылок. Утечка ссылки = полный доступ к конфигам до смены токена (механизма смены в API не видно).
- Сортировка CDN «по подстроке `cdn`» в имени — **ломается**, если переименовать ноду без «cdn» в названии.

Пакет `internal/infrastructure/xray` **не используется** из основного потока генерации подписки — дублирование логики.

---

## 6. Платежи

### Плюсы

- HTTP-клиент ЮKassa с **Idempotence-Key** на create.
- Повторная обработка webhook для уже `succeeded`/`canceled` отсекается по статусу в БД.
- Worker догоняет pending.

### Минусы (реальные)

1. **Гонка webhook vs polling:** два параллельных запроса могут оба прочитать `pending` и оба вызвать `ActivateSubscription` до того, как статус сохранится — **двойное продление**. Нужна транзакция / `SELECT FOR UPDATE` / атомарное «только первый успешный апдейт».
2. **Webhook:** только IP; за прокси с некорректным `X-Forwarded-For` теоретически опасно (зависит от деплоя).
3. **Metadata из webhook** не сверяется с записью платежа (сумма/tier/days) — доверие только к своей БД и предыдущему create; при компрометации потока событий защита слабая.
4. В коде есть комментарий про **AmountRub** «временно» — сумма в рублях из копеек округлением; для нестандартных периодов важно следить за консистентностью с ЮKassa.

---

## 7. Security audit (кратко)

| Риск | Детали |
|------|--------|
| Перебор `/sub/{token}` | Нет rate limit; UUIDv4 ≈ 122 бит энтропии — грубый перебор нереалистичен, **утечки логов/рефереров** опаснее. |
| Нет лимита на `/auth/tg` | Brute/DoS по телеграм-логике и нагрузка на БД. |
| JWT в ответе + `sub_token` | Долгоживущий токен; нет refresh/revoke в коде. |
| Админ веб | Один общий `admin_token`; утечка = полный доступ. CSRF для cookie-сессии учтён на POST. |
| Remnawave client | Пути `/users`, `/users/{id}/expiry` **могут не совпадать** с реальным API панели — в комментарии в коде это указано; **в проде интеграция может быть нерабочей без сверки**. |

---

## 8. Production readiness

### Готов ли проект к запуску?

**Как публичный коммерческий SaaS — нет.**  
**Как закрытый MVP для ограниченной аудитории с ручным контролем — близко**, при условии настройки нод, Remnawave, ЮKassa и прокси.

### Блокеры (critical)

1. **REST `/api/v1/users/*` — заглушки**; мобильный клиент не может нормально жить только на них (кроме auth + payments + sub URL из auth).
2. **Идемпотентность активации подписки** при конкурирующих webhook/poll.
3. **Remnawave:** нет гарантированного **создания пользователя** на панели; только обновление срока — нужно сверить с реальным API и дописать flow.
4. **Процессы web/bot без платёжного gateway** — не единая правда с API (операционный долг).

### Риски (medium)

- Версия routing не отражает данные БД.
- `ExpireOld` прерывается на первой ошибке в цикле — частичный даунгрейд.
- Нет observability (метрики, трассировки, алерты на webhook errors).

### Улучшения (low)

- Убрать дублирование xray/config_usecase.
- Жёсткая сортировка CDN по `RegionCDN` / флагу, а не по имени.
- Заполнить `subscription-userinfo` реальными лимитами/expire.

---

## 9. Чего не хватает до «настоящего» продукта

- **Мониторинг:** Prometheus/OpenTelemetry, SLO на `/sub`, webhook, workers.
- **Централизованные логи + алерты** (сейчас JSON slog в stdout — база, но без политики retention/поиска).
- **Горизонтальное масштабирование API:** in-memory кэш routing **не shared** между инстансами (устаревание списков); нужен Redis или короткий TTL + ETag.
- **Защита от злоупотреблений:** WAF, rate limits, капча на чувствительных эндпоинтах при необходимости.
- **Управление устройствами и отзыв доступа.**
- **Ротация sub_token, аудит действий админа.**
- **E2E-тесты** хотя бы на критические пути (sub, webhook happy path).

---

## 10. Итоговый вердикт

- **Стадия:** **pre-MVP / ранний MVP** с сильным ядром usecase и рабочими кусками API, но **дырявым пользовательским REST-слоем** и **операционными рисками** (платежи, панель).

### Сильные стороны

Чёткое разделение интерфейсов usecase/repository; осмысленная модель подписок и платежей; webhook с IP-filter; Telegram auth по канону; админка и бот на реальных usecase; routing + worker под antifilter; генерация подписки с CDN в конце.

### Главные слабости

Заглушки user API; гонки при активации оплаты; неполная/непроверенная связка Remnawave; расхождение конфигурации между `cmd/api` и `cmd/web`/`cmd/bot`; слабая защита публичных эндпоинтов (лимиты); «мёртвые» поля домена (скорость); дублирование генерации URI.

### Топ-5 действий дальше

1. **Реализовать** `UserHandler` (me, subscription) или убрать маршруты до готовности — не вести в прод заглушки.
2. **Сделать атомарную идемпотентную активацию** после оплаты (транзакция БД + уникальный constraint / conditional update).
3. **Сверить Remnawave client с реальным API** и добавить `CreateUser` в цепочку регистрации/первой оплаты.
4. **Унифицировать DI** в web/bot с api (те же адаптеры YooKassa/Remnawave или общий bootstrap).
5. **Rate limiting** на `GET /sub/{token}` и `POST /api/v1/auth/tg` + мониторинг ошибок webhook/workers.

---

*Документ сформирован по состоянию кодовой базы на момент аудита.*
