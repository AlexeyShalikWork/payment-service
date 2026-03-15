# Payment Service

Микросервис для обработки платежей на Go. Принимает команды через Kafka, хранит данные в PostgreSQL, отдаёт по HTTP.

## Стек

- **Go** + Fiber (HTTP)
- **PostgreSQL** + sqlx (хранение)
- **Kafka** + franz-go (события)
- **OpenTelemetry** (трейсинг)

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
  kafka/
    consumer.go            — чтение событий (retry + DLQ)
    producer.go            — публикация событий
    outbox_publisher.go    — transactional outbox → Kafka
    propagation.go         — trace context через Kafka headers
    events/                — структуры событий
  ports/                   — интерфейсы
  tracing/                 — инициализация OpenTelemetry
migrations/                — SQL-миграции (применяются при старте)
```

## Переменные окружения

| Переменная       | По умолчанию                         | Описание                       |
|------------------|--------------------------------------|--------------------------------|
| `DB_URL`         | `postgres://localhost:5432/payments` | PostgreSQL connection          |
| `KAFKA_BROKERS`  | `localhost:9092`                     | Kafka брокеры (через запятую)  |
| `HTTP_PORT`      | `3005`                               | Порт HTTP-сервера              |
| `LOG_FORMAT`     | `text`                               | Формат логов (`text` / `json`) |
| `MIGRATIONS_DIR` | `migrations`                         | Путь к миграциям               |

## Запуск

```bash
DB_URL=postgres://user:pass@localhost:5432/payments go run ./cmd/api/
```

Миграции применяются автоматически при старте.

## API

```
GET /payments/:user_id   — список платежей пользователя
GET /payment/:id         — платёж по ID
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

## Архитектурные решения

- **Transactional outbox** — платёж и событие записываются в одной транзакции, outbox publisher публикует в Kafka отдельно
- **Idempotency** — `ON CONFLICT (id) DO NOTHING` на уровне БД
- **Retry + DLQ** — 3 попытки с нарастающей паузой, потом dead letter queue
- **Graceful shutdown** — корректное завершение HTTP, Kafka consumer, outbox publisher с таймаутом 10 сек
- **OpenTelemetry** — сквозной трейсинг HTTP → Service → Kafka с propagation через headers

## Тесты

```bash
go test ./...
```
