---
description: "Go coding standards and patterns for FreeWay VPN project"
globs: ["**/*.go"]
alwaysApply: true
---

# Go Rules — FreeWay VPN

## Обязательные принципы

### Clean Architecture
- **СТРОГО**: handler → usecase → repository. Никаких исключений.
- Handler знает только об usecase интерфейсах, никогда о repository напрямую
- Usecase знает только об repository интерфейсах, никогда о конкретных реализациях
- Dependency Injection через конструкторы: `func NewUserUseCase(repo UserRepository) UserUseCase`

### Интерфейсы
- Все usecase и repository — интерфейсы, определённые в `interfaces.go`
- Конкретные реализации в отдельных файлах
- Принцип минимальных интерфейсов: только нужные методы

### Обработка ошибок
```go
// ПРАВИЛЬНО: оборачивать с контекстом
return fmt.Errorf("userUseCase.GetByID: %w", err)

// НЕПРАВИЛЬНО: голая ошибка без контекста
return err

// Проверка sentinel errors
if errors.Is(err, gorm.ErrRecordNotFound) {
    return nil, domain.ErrUserNotFound
}
```

### Логирование
```go
// Использовать только slog (стандартная библиотека)
slog.Info("subscription activated",
    "user_id", userID,
    "tier", tier,
    "days", days,
)
slog.Error("payment webhook failed",
    "error", err,
    "payment_id", paymentID,
)
```

### Именование
- Пакеты: одно слово, строчные (usecase, repository, handler)
- Интерфейсы: глагол или существительное без суффикса -er если не стандартно
  - UserUseCase (не UserUseCaser)
  - UserRepository (не UserRepositoryer)
- Структуры реализаций: `userUseCase`, `userRepository` (приватные)
- Конструкторы: `NewUserUseCase(...)`, `NewUserRepository(...)`

### Конфигурация
```go
// Все секреты только через env. В config.yaml только плейсхолдеры:
// jwt_secret: "${JWT_SECRET}"
// Загрузка:
os.Getenv("JWT_SECRET")
```

### HTTP Handlers
```go
// Структура handler'а
type AuthHandler struct {
    userUC usecase.UserUseCase
}

func NewAuthHandler(userUC usecase.UserUseCase) *AuthHandler {
    return &AuthHandler{userUC: userUC}
}

// Ответы всегда через helper
func respondJSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
    respondJSON(w, status, map[string]string{"error": message})
}
```

### Workers
```go
// Все workers реализуют интерфейс
type Worker interface {
    Start(ctx context.Context)
    Stop()
}

// Graceful shutdown через context
func (w *NodeHealthWorker) Start(ctx context.Context) {
    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            w.check(ctx)
        case <-ctx.Done():
            return
        }
    }
}
```

### Тесты
```go
// Table-driven tests обязательны для usecase-слоя
func TestUserUseCase_GetByID(t *testing.T) {
    tests := []struct {
        name    string
        userID  uint
        mockFn  func(*MockUserRepository)
        want    *domain.User
        wantErr error
    }{
        {
            name:   "успешное получение",
            userID: 1,
            mockFn: func(m *MockUserRepository) {
                m.On("GetByID", mock.Anything, uint(1)).
                    Return(&domain.User{ID: 1}, nil)
            },
            want: &domain.User{ID: 1},
        },
        {
            name:   "пользователь не найден",
            userID: 999,
            mockFn: func(m *MockUserRepository) {
                m.On("GetByID", mock.Anything, uint(999)).
                    Return(nil, domain.ErrUserNotFound)
            },
            wantErr: domain.ErrUserNotFound,
        },
    }
    // ...
}
```

### GORM паттерны
```go
// Всегда передавай context
db.WithContext(ctx).First(&user, id)
db.WithContext(ctx).Create(&user)

// Soft delete через DeletedAt (GORM встроенный)
type User struct {
    gorm.Model  // включает ID, CreatedAt, UpdatedAt, DeletedAt
    // ...
}

// Транзакции для связанных операций
err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(&user).Error; err != nil {
        return err
    }
    if err := tx.Create(&subscription).Error; err != nil {
        return err
    }
    return nil
})
```

## Запрещено

- `panic()` в любом продакшен-коде
- Глобальные переменные для зависимостей
- `init()` функции
- Прямые вызовы DB из handler или usecase слоёв
- Хардкод секретов, IP-адресов, URL в коде
- `fmt.Println` — только `slog`
- Игнорирование ошибок через `_` без комментария `//nolint`

## Обязательная структура нового файла

```go
// Package usecase содержит бизнес-логику FreeWay VPN.
package usecase

import (
    // стандартная библиотека
    "context"
    "fmt"
    
    // внешние зависимости
    "github.com/google/uuid"
    
    // внутренние пакеты
    "github.com/yourusername/freeway/internal/domain"
)

// UserUseCase описывает бизнес-операции над пользователями.
type UserUseCase interface {
    GetByID(ctx context.Context, id uint) (*domain.User, error)
}

type userUseCase struct {
    userRepo UserRepository
}

// NewUserUseCase создаёт новый экземпляр UserUseCase.
func NewUserUseCase(userRepo UserRepository) UserUseCase {
    return &userUseCase{userRepo: userRepo}
}
```
