# Payment Service

Микросервис для обработки платежей на Go. Принимает команды через Kafka, хранит данные в PostgreSQL, отдаёт по HTTP.

## Стек

- **Go** + Fiber (HTTP)
- **PostgreSQL** + sqlx (хранение)
- **Kafka** + franz-go (события)

## Структура

```
cmd/api/main.go          — точка входа
internal/
  config/                — конфигурация через env
  model/                 — модели (Payment, статусы, типы, валюты)
  repository/            — слой БД
  service/               — бизнес-логика
  http/                  — HTTP-хендлеры
  kafka/
    consumer.go          — чтение событий из Kafka
    producer.go          — публикация событий в Kafka
    events/              — структуры событий
  ports/                 — интерфейсы
migrations/              — SQL-миграции
```

## Переменные окружения

| Переменная      | По умолчанию                          | Описание              |
|-----------------|---------------------------------------|-----------------------|
| `DB_URL`        | `postgres://localhost:5432/payments`  | PostgreSQL connection |
| `KAFKA_BROKERS` | `localhost:9092`                      | Kafka брокеры (через запятую) |
| `HTTP_PORT`     | `3005`                                | Порт HTTP-сервера     |
| `LOG_FORMAT`    | `text`                                | Формат логов (`text` / `json`) |

## Запуск

```bash
# миграции (golang-migrate)
migrate -path migrations -database "$DB_URL" up

# запуск
DB_URL=postgres://user:pass@localhost:5432/payments go run ./cmd/api/
```

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
