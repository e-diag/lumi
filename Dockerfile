# ── Builder ────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Контекст должен быть корнем репозитория (есть cmd/api, internal/). Иначе на сервере часто только compose+config (~9 KiB) — сборка падает непонятно.
RUN test -d ./cmd/api && test -d ./cmd/worker && test -d ./cmd/bot && test -d ./cmd/web && test -d ./cmd/migrator || ( \
  echo "ERROR: неполный контекст Docker: нет ./cmd/*. Клонируйте весь репозиторий и запускайте docker compose из его корня (не из каталога только с compose-файлом)." >&2; \
  echo "Содержимое контекста:" >&2; ls -la >&2; \
  exit 1)

# Компилируем бинари
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api      ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/bot      ./cmd/bot
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/web      ./cmd/web
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/migrator ./cmd/migrator

# ── Runtime ────────────────────────────────────────────────
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata wget
WORKDIR /app

COPY --from=builder /bin/api      /app/api
COPY --from=builder /bin/worker   /app/worker
COPY --from=builder /bin/bot      /app/bot
COPY --from=builder /bin/web      /app/web
COPY --from=builder /bin/migrator /app/migrator
COPY --from=builder /app/config.yaml /app/config.yaml
# Из builder: на части серверов контекст сборки без этого каталога (sparse/rsync) — шаблоны уже в образе после COPY . .
COPY --from=builder /app/internal/handler/web/templates /app/internal/handler/web/templates

EXPOSE 8080 3000

# По умолчанию запускаем API (переопределяется в docker-compose)
CMD ["/app/api"]
