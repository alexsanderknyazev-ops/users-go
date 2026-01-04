# Build stage
FROM golang:1.24.0 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o user-service .

# Final stage
FROM alpine:latest

# Устанавливаем необходимые пакеты
RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Копируем бинарник
COPY --from=builder /app/user-service .

# Даем права на выполнение
RUN chmod +x user-service

EXPOSE 8072

# Health check (если есть эндпоинт /health)
HEALTHCHECK --interval=30s --timeout=3s --start-period=15s --retries=3 \
  CMD curl -f http://localhost:8072/health || exit 1

CMD ["./user-service"]