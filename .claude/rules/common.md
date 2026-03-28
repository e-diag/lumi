---
description: "Universal development principles for FreeWay VPN"
alwaysApply: true
---

# Общие правила разработки — FreeWay VPN

## Git workflow

```
feat(api): add subscription endpoint
fix(bot): correct node health check interval
refactor(usecase): extract payment validation logic
docs(readme): update deployment instructions
test(worker): add routing update worker tests
chore(deps): update go dependencies
```

Коммиты на английском. Формат: `type(scope): description`.

## Порядок работы над задачей

1. Прочитай CLAUDE.md и соответствующий rule-файл
2. Создай интерфейс перед реализацией
3. Напиши тест перед реализацией (TDD)
4. Реализуй минимально рабочий код
5. Рефакторинг
6. Проверь: `go build ./...` и `go test ./...`

## Безопасность (обязательно)

- Никаких секретов в коде, конфигах, логах
- JWT токены не логировать
- SQL: только параметризованные запросы через GORM (не raw SQL со строковой конкатенацией)
- Входные данные API: всегда валидировать перед передачей в usecase
- Rate limiting на все публичные endpoints
- Проверка IP для вебхука ЮKassa (список доверенных IP в конфиге)

## Структура файлов

```
Один файл = одна ответственность.
Максимум 300 строк на файл.
Если больше — декомпозируй.
```

## Документация

```go
// Все публичные типы и функции — комментарий перед объявлением.
// Комментарии на русском языке.
// Формат: "ИмяФункции делает X."

// GetBySubToken возвращает пользователя по subscription токену.
// Используется endpoint'ом GET /sub/{token}.
func (uc *userUseCase) GetBySubToken(ctx context.Context, token string) (*domain.User, error) {
```

## Docker

- Все сервисы запускаются через docker-compose
- Prod и dev конфиги разделены: `docker-compose.yml` и `docker-compose.prod.yml`
- Health checks обязательны для всех сервисов
- Никаких root-пользователей в контейнерах

## Переменные окружения

Всегда документировать в `.env.example`:
```
# Telegram
TELEGRAM_BOT_TOKEN=          # Токен бота от @BotFather
TELEGRAM_ADMIN_ID=           # Ваш Telegram ID (числовой)

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=                 # Обязательно заполнить
DB_NAME=freeway

# JWT
JWT_SECRET=                  # Минимум 32 символа, случайная строка

# ЮKassa
YOOKASSA_SHOP_ID=            # ID магазина из личного кабинета
YOOKASSA_SECRET_KEY=         # Секретный ключ из личного кабинета

# Remnawave
REMNAWAVE_HOST=              # URL панели Remnawave
REMNAWAVE_API_TOKEN=         # API токен Remnawave

# Ноды
NODE_EU_IP=                  # IP EU-ноды
NODE_EU_PUBLIC_KEY=          # Reality public key EU-ноды
NODE_EU_SHORT_ID=            # Reality short ID EU-ноды
NODE_USA_IP=                 # IP USA-ноды
NODE_USA_PUBLIC_KEY=         # Reality public key USA-ноды
NODE_USA_SHORT_ID=           # Reality short ID USA-ноды
```

## Проверки перед коммитом

```bash
go build ./...          # должно компилироваться
go test ./...           # все тесты зелёные
go vet ./...            # нет предупреждений
gofmt -l .              # нет неформатированных файлов
```
