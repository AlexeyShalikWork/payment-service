# Payment Service

Микросервис для обработки платежей на Go. Принимает команды через Kafka, хранит данные в PostgreSQL, отдаёт по HTTP.

## Стек

- **Go** + Fiber (HTTP)
- **PostgreSQL** + sqlx (хранение)
- **Kafka** + franz-go (события)
- **OpenTelemetry** (трейсинг)
- **Prometheus** + Grafana (метрики)
- **Elasticsearch** + Kibana (логирование)
- **Docker Compose** (локальная инфраструктура)
- **Kubernetes** (оркестрация)

## Структура

```
cmd/api/main.go            — точка входа
internal/
  config/                  — конфигурация через env
  model/                   — модели (Payment, Outbox)
  repository/              — слой БД (payment, outbox)
  service/                 — бизнес-логика + валидация
  http/
    payment.go             — HTTP-хендлеры
    middleware.go           — RequestID, Logger, Recover, Tracing
    metrics.go             — Prometheus метрики
  kafka/
    consumer.go            — чтение событий (retry + DLQ)
    producer.go            — публикация событий
    outbox_publisher.go    — transactional outbox → Kafka
    propagation.go         — trace context через Kafka headers
    events/                — структуры событий
  logging/
    elastic.go             — slog handler для Elasticsearch
    multi.go               — multi-handler (консоль + ES)
  ports/                   — интерфейсы
  tracing/                 — инициализация OpenTelemetry
migrations/                — SQL-миграции (применяются при старте)
k8s/                       — Kubernetes манифесты
```

## Переменные окружения

| Переменная       | По умолчанию                         | Описание                       |
|------------------|--------------------------------------|--------------------------------|
| `DB_URL`         | `postgres://localhost:5432/payments` | PostgreSQL connection          |
| `KAFKA_BROKERS`  | `localhost:9092`                     | Kafka брокеры (через запятую)  |
| `HTTP_PORT`      | `3005`                               | Порт HTTP-сервера              |
| `LOG_FORMAT`     | `text`                               | Формат логов (`text` / `json`) |
| `MIGRATIONS_DIR` | `migrations`                         | Путь к миграциям               |
| `ELASTIC_URL`    |                                      | Elasticsearch URL (опционально)|

## Запуск

### Локальная разработка

```bash
# Поднять инфраструктуру
docker compose up -d postgres kafka prometheus grafana elasticsearch kibana

# Запустить приложение
go run ./cmd/api
```

### Полностью в Docker

```bash
docker compose up -d --build
```

### Kubernetes (minikube)

```bash
minikube start
eval $(minikube docker-env)
docker build -t payment:latest .
kubectl apply -f k8s/
```

## API

```
GET /payment/health      — health check
GET /payments/:user_id   — список платежей пользователя
GET /payment/:id         — платёж по ID
GET /metrics             — Prometheus метрики
```

## Kafka

Топик: `PaymentService`

Входящее событие (consumer):
```json
{
  "eventType": "payment.create",
  "payload": {
    "id": "uuid",
    "userId": "uuid",
    "type": "charge",
    "amount": 1000,
    "currency": "USD"
  }
}
```

Типы платежей: `charge`, `refund`, `payout`

DLQ топик: `PaymentService.dlq` — сообщения после 3 неудачных попыток обработки.

## Мониторинг

| Сервис         | URL                    | Назначение          |
|----------------|------------------------|---------------------|
| Prometheus     | http://localhost:9090  | Сбор метрик         |
| Grafana        | http://localhost:3000  | Дашборды            |
| Kibana         | http://localhost:5601  | Поиск по логам      |
| Elasticsearch  | http://localhost:9200  | Хранение логов      |

## Архитектурные решения

- **Transactional outbox** — платёж и событие записываются в одной транзакции, outbox publisher публикует в Kafka отдельно
- **Idempotency** — `ON CONFLICT (id) DO NOTHING` на уровне БД
- **Retry + DLQ** — 3 попытки с нарастающей паузой, потом dead letter queue
- **Graceful shutdown** — корректное завершение HTTP, Kafka consumer, outbox publisher через WaitGroup с таймаутом 10 сек
- **OpenTelemetry** — сквозной трейсинг HTTP → Service → Kafka с propagation через headers
- **Prometheus метрики** — http_requests_total, http_request_duration_seconds
- **Централизованное логирование** — slog → Elasticsearch через кастомный handler

## Тесты

```bash
go test ./...
```
