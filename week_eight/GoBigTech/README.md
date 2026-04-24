# GoBigTech

Production-подобная микросервисная платформа обработки заказов с API Gateway (Envoy), event-driven архитектурой (Kafka) и полной observability (metrics, logs, traces).

## Архитектура (кратко)

Client → Envoy → Services → Kafka → Consumers

- Envoy — API Gateway + auth (Lua)
- Kafka — асинхронный pipeline
- gRPC — межсервисное взаимодействие

## Состав системы

- `order` — HTTP API заказов, оркестрация бизнес-процесса.
- `inventory` — gRPC-сервис склада (MongoDB).
- `payment` — gRPC-сервис оплаты.
- `assembly` — фоновая обработка событий оплаты и публикация события сборки.
- `notification` — отправка уведомлений в Telegram по событиям.
- `iam` — gRPC-сервис аутентификации/сессий (PostgreSQL + Redis) и внутренний HTTP endpoint для проверки сессии.
- `platform` — общие библиотеки (Kafka, логирование, health, shutdown, события).

## Технологии

- Go `1.24.4` (см. `go.work`).
- gRPC + Protobuf.
- HTTP + OpenAPI (`api/openapi/order.yaml`).
- PostgreSQL, MongoDB, Redis.
- Kafka (KRaft).
- Envoy.
- OpenTelemetry Collector, Prometheus, Alertmanager, Grafana, Jaeger, Elasticsearch, Kibana.

## Структура репозитория

```text
.
├── api/
│   ├── openapi/
│   └── proto/
├── platform/
├── services/
│   ├── order/
│   ├── inventory/
│   ├── payment/
│   ├── assembly/
│   ├── notification/
│   └── iam/
├── docker-compose.yml
├── envoy.yaml
└── go.work
```

## Схема взаимодействия

1. Клиент отправляет HTTP-запрос в `order` (`POST /orders`).
2. `order` вызывает `inventory` (резерв стока) и `payment` (оплата) по gRPC.
3. `order` сохраняет заказ в PostgreSQL и публикует событие `order.payment.completed.v1` в Kafka.
4. `assembly` читает событие оплаты, имитирует сборку, публикует `order.assembly.completed.v1`.
5. `order` обновляет статус заказа по событию сборки.
6. `notification` слушает события и отправляет сообщения в Telegram (контакты запрашивает в `iam`).

## Требования

- Docker + Docker Compose v2.
- Go `1.24+`.
- (Опционально) `goose` для ручного запуска миграций:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

## Быстрый старт (рекомендуется)

Все команды выполняются из директории, где находится `docker-compose.yml`.

1) Поднять все сервисы:

```bash
docker compose up -d --build
docker compose ps
```

2) Применить миграции для `order` и `iam`:

```bash
# Если база iamdb не создана:
docker compose exec db psql -U postgres -d postgres -c "CREATE DATABASE iamdb;"

goose -dir ./services/order/migrations postgres "postgres://postgres:postgres@localhost:5432/appdb?sslmode=disable" up
goose -dir ./services/iam/migrations postgres "postgres://postgres:postgres@localhost:5432/iamdb?sslmode=disable" up
```

3) Проверить доступность основных точек:

- Order API: `http://localhost:8080`
- Envoy: `http://localhost:8081`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`
- Jaeger: `http://localhost:16686`
- Kibana: `http://localhost:5601`

Остановка:

```bash
docker compose down
```

С удалением volumes:

```bash
docker compose down -v
```

## Локальный запуск сервисов без Docker Compose

Сначала поднимите инфраструктуру:

```bash
docker compose up -d db mongo redis kafka
```

Затем в отдельных терминалах:

```bash
go run ./services/inventory/cmd/inventory
go run ./services/payment/cmd/payment
go run ./services/order/cmd/order
go run ./services/assembly/cmd/assembly
go run ./services/notification/cmd/notification
go run ./services/iam/cmd/iam
```

## Миграции БД

- `order`: `services/order/migrations`
- `iam`: `services/iam/migrations`

Примеры команд:

```bash
goose -dir ./services/order/migrations postgres "postgres://postgres:postgres@localhost:5432/appdb?sslmode=disable" status
goose -dir ./services/order/migrations postgres "postgres://postgres:postgres@localhost:5432/appdb?sslmode=disable" up

goose -dir ./services/iam/migrations postgres "postgres://postgres:postgres@localhost:5432/iamdb?sslmode=disable" status
goose -dir ./services/iam/migrations postgres "postgres://postgres:postgres@localhost:5432/iamdb?sslmode=disable" up
```

## API и контракты

### HTTP (Order)

OpenAPI: `api/openapi/order.yaml`

Создать заказ:

```bash
curl -X POST "http://localhost:8080/orders" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"u1","items":[{"product_id":"p1","quantity":2}]}'
```

Получить заказ:

```bash
curl "http://localhost:8080/orders/<order_id>"
```

### gRPC (Proto)

- `api/proto/inventory/v1/inventory.proto`
- `api/proto/payment/v1/payment.proto`
- `api/proto/iam/v1/iam.proto`

Генерация Go-кода выполняется через `buf` с конфигом `tools/buf.gen.yaml`.

## Переменные окружения (основные)

### `order`

- `HTTP_PORT` (по умолчанию `8080`)
- `POSTGRES_DSN`
- `INVENTORY_GRPC_ADDR`
- `PAYMENT_GRPC_ADDR`
- `KAFKA_BROKERS`

### `inventory`

- `GRPC_ADDR` (по умолчанию `127.0.0.1:50051`)
- `MONGO_URI`
- `MONGO_DB`
- `IAM_GRPC_ADDR`

### `payment`

- `GRPC_ADDR` (по умолчанию `127.0.0.1:50052`)

### `iam`

- `GRPC_ADDR` (по умолчанию `0.0.0.0:50060`)
- `HTTP_ADDR` (по умолчанию `0.0.0.0:50061`)
- `POSTGRES_DSN`
- `REDIS_ADDR`

### `notification`

- `KAFKA_BROKERS`
- `TELEGRAM_BOT_TOKEN`
- `TELEGRAM_CHAT_ID`
- `IAM_GRPC_ADDR`

## Тесты

Из корня проекта:

```bash
go test ./... 
```

Примеры точечного запуска:

```bash
go test ./services/order/internal/service -v
go test ./services/order/internal/repository -v
go test ./services/inventory/e2e -v
```

> Для интеграционных/E2E тестов требуется Docker (используются testcontainers).

## Observability

- Метрики и алерты: `prometheus.yml`, `alerts.yml`, `alertmanager.yml`
- Трейсинг и сбор телеметрии: `otel-collector-config.yaml`
- Маршрутизация/гейтвей: `envoy.yaml`

## Полезные команды

Логи всех сервисов:

```bash
docker compose logs -f
```

Логи конкретного сервиса:

```bash
docker compose logs -f order
```

Пересборка одного сервиса:

```bash
docker compose up -d --build order
```

## Архитектурные решения

- Использован Envoy как единая точка входа и слой авторизации
- Kafka выбран для асинхронной обработки и слабой связанности сервисов
- gRPC применяется для эффективной межсервисной коммуникации
- Polyglot persistence: PostgreSQL + MongoDB + Redis

## Безопасность

- Не храните реальные токены/ключи в `docker-compose.yml`.
- Для локальной разработки передавайте секреты через `.env`/переменные окружения.
- Перед публикацией проекта обязательно проверьте, что чувствительные данные удалены из истории Git.