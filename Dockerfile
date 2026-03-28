# ── Builder ────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Компилируем все 4 бинаря
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api      ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/bot      ./cmd/bot
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/web      ./cmd/web
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/migrator ./cmd/migrator

# ── Runtime ────────────────────────────────────────────────
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /bin/api      /app/api
COPY --from=builder /bin/bot      /app/bot
COPY --from=builder /bin/web      /app/web
COPY --from=builder /bin/migrator /app/migrator
COPY config.yaml /app/config.yaml

EXPOSE 8080 3000

# По умолчанию запускаем API (переопределяется в docker-compose)
CMD ["/app/api"]
