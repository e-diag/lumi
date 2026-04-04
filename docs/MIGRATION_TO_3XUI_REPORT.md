# Отчёт: переход на 3x-ui и Telegram-first модель

## Что удалено

- Пакет **`internal/infrastructure/remnawave`** (HTTP-клиент Remnawave и адаптер под старый интерфейс).
- Конфигурация **`remnawave`** в `config.yaml` и переменные **`REMNAWAVE_*`** в документации и `.env.example`.

## Что переписано

- **Интерфейс панели**: вместо `RemnawaveClient` введён **`VPNPanelClient`** с методом `SyncUserAccess` (`internal/usecase/interfaces.go`).
- **`SubscriptionUseCase`**: при изменении подписки вызывается синхронизация с 3x-ui; в `user` сохраняются поля **`PanelClientUUID`** и **`PanelSubID`**.
- **`ConfigUseCase`**: при наличии `XUI_PUBLIC_SUBSCRIPTION_BASE_URL` и `user.PanelSubID` подписка для `GET /sub/{token}` **загружается с публичного subscription-сервера 3x-ui**; иначе — прежняя генерация VLESS из таблицы `nodes`.
- **Bootstrap**: `newVPNPanelAdapter` поднимает **`internal/infrastructure/xui`**; при пустом `XUI_BASE_URL` адаптер не создаётся (только БД).
- **Telegram-бот**: главное меню и сценарий «Подключить VPN» переведены на клиенты **Happ** и **v2RayTun**; добавлены пункты «Мои ключи / подписка», «Поддержка».

## Что сохранено

- Модель **users / subscriptions / payments / referrals / nodes / routing** и воркеры (платежи, истечение подписок, здоровье нод, routing).
- **ЮKassa** (создание платежа, webhook, идемпотентная активация через `PaymentActivation`).
- **JWT + Telegram auth** для API.
- **Реферальная логика** и **антиабьюз триалов** по IP.
- Веб-панель менеджера (дашборд, пользователи, ноды, платежи, routing) — расширен счётчик **истёкших подписок** в статистике.

## Как интегрирована 3x-ui

1. **HTTP-клиент** (`internal/infrastructure/xui/client.go`): логин в панель, cookie-сессия, `POST /panel/api/inbounds/addClient` и `POST /panel/api/inbounds/updateClient/{clientId}`.
2. **Адаптер** (`internal/infrastructure/xui/adapter.go`): для каждого пользователя создаётся клиент с `email` вида `fw-{uuid}@svc`, **`limitIp: 3`**, **`subId`** для subscription URL на стороне панели.
3. **Публичная подписка**: backend проксирует ответ `GET {XUI_PUBLIC_SUBSCRIPTION_BASE_URL}/{XUI_SUBSCRIPTION_PATH}/{PanelSubID}` в `GenerateSubscription`, чтобы **Happ и v2RayTun** получали тот же формат, что отдаёт 3x-ui.

## Поддержка Happ

- Пользовательская ссылка: **`BASE_URL/sub/{sub_token}`** (как и раньше).
- При настроенной 3x-ui тело ответа совпадает с subscription-контентом панели.
- В боте: отдельная ветка Happ, ссылка на App Store из `TELEGRAM_APP_URL_IOS`, кнопка открытия HTTPS subscription URL.

## Поддержка v2RayTun

- Тот же **subscription URL**; в боте — диплинк `v2raytun://install-config?url=` + base64 (как у ряда Android-клиентов); при необходимости — ручное копирование URL.

## Риски и ограничения

- **Схема диплинка v2RayTun** может отличаться на конкретной сборке; при сбое — копирование URL подписки.
- **Путь subscription** в 3x-ui настраивается в панели; переменная **`XUI_SUBSCRIPTION_PATH`** должна совпадать с ним (по умолчанию `sub`).
- **Один inbound** в конфиге: `inbound_id` в `config.yaml` / `XUI_INBOUND_ID`; для нескольких inbound потребуется расширение модели (таблица серверов/связей).
- **Нет 3x-ui** в конфиге: подписка только из БД-нод; новые пользователи не получат `PanelSubID` до включения панели.

## Рекомендуемые следующие шаги

1. Тарифы и цены из **БД** (таблица `plans`) вместо хардкода в `calcAmountKopeks`.
2. Отдельная таблица **VPN-серверов** с привязкой к инстансам 3x-ui и health-check через API панели.
3. Автотесты с **httptest** для клиента 3x-ui (мок ответов `success`).

---

## Финальный аудит репозитория (после миграции)

Проверено: **`go build ./...`**, **`go test ./...`**, поиск по коду на остатки Remnawave, согласованность `.env.example` / `config.yaml` / `docs`, `docker-compose.yml` и `Dockerfile`.

| Критерий | Результат |
|----------|-----------|
| Сборка | Успешна для всех пакетов под модулем. |
| Импорты | Битых импортов на удалённый `remnawave` нет; пакет удалён из дерева исходников. |
| Устаревшие ссылки | `REMNAWAVE_*` / `remnawave:` в коде и актуальной документации не используются (упоминания только в этом отчёте как история). |
| Мёртвый код | Отдельного «хвоста» Remnawave не осталось; routing/workers/nodes — осознанно сохранены. |
| Документация | README / SETUP синхронизированы с 3x-ui; добавлены ограничения веб-панели и бота; убрана ложная отсылка к каталогу `migrations/` (схема через GORM AutoMigrate). |
| Docker Compose | Сервисы `postgres`, `api`, `bot`, `web`; общий `.env`; `CONFIG_PATH` в образе не требуется — `config.yaml` в `/app`. Нужно вручную задать корректный `DATABASE_DSN` для сети compose. |
| Переменные окружения | В `.env.example` присутствуют `XUI_*`, включая **`XUI_INBOUND_ID`** (перекрывает `inbound_id` в YAML при задании). |
| Панель менеджера (web) | Реализованы дашборд, пользователи, ноды, платежи, routing. Нет UI ценообразования и глобальных настроек продукта — см. SETUP §10. |
| Пользовательский бот | Сценарий Happ / v2RayTun, ключи, поддержка, оплата, рефералы; менеджерский `/manager` с известными заглушками. |

**Итог:** репозиторий собирается, тесты проходят, документация и env приведены к модели 3x-ui; до «полного» продукта по ТЗ не хватает в основном **управления тарифами в БД/UI** и **глубокой интеграции health 3x-ui** в панели.
