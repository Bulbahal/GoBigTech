# Order Service в составе системы
Небольшая микросервисная система:
- **order** — HTTP API для создания заказа (оркестрация)
- **inventory** — gRPC сервис склада (MongoDB)
- **payment** — gRPC сервис оплаты (stub/эмуляция)

# Архитектура
Слои в Order Service:
- `transport/http` — принимает запросы и маппит DTO ↔ доменные модели
- `service` — бизнес-логика и orchestration (Inventory + Payment + Repository)
- `repository` — PostgreSQL (orders, order_items)
- `clients` — gRPC-адаптеры Inventory/Payment  
  Поток запроса:
```
HTTP → Handler → Service
                    ├─→ Inventory (MongoDB)
                    ├─→ Payment (gRPC)
                    └─→ Repository (Postgres)
```

# Хранилища данных
•PostgreSQL — хранение заказов и позиций заказа (реляционные данные, транзакции)  
•MongoDB — хранение и резервирование складских остатков (частые обновления, простая структура)  
Миграции PostgreSQL выполняются с помощью goose.

# Быстрый старт

Все команды выполняются **из корня репозитория** (где лежит `docker-compose.yml`).

## 1. Поднять инфраструктуру (PostgreSQL + MongoDB)

```bash  
docker compose up -ddocker compose ps
```  

## 2. Применить миграции PostgreSQL

```bash  
cd services/ordergoose -dir ./migrations postgres "postgres://postgres:postgres@localhost:5432/appdb?sslmode=disable" upcd ../..
```  

## 2. Запустить сервисы
В разных терминалах:
```bash  
cd services/inventory && go run ./cmd/inventory

cd services/payment && go run ./cmd/payment

cd services/order && go run ./cmd/order
```  
Order Service будет доступен по адресу:
```bash  
http://localhost:8080  
```  


## API
OpenAPI спецификация:
- api/openapi/order.yaml
  **Создание заказа**
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id":"u1","items":[{"product_id":"p1","quantity":2}]}'
```
**Ошибка при недостатке товара**
```bash
curl -i -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id":"u1","items":[{"product_id":"p1","quantity":999}]}'
```

# Тесты
Запуск тестов:
```bash  
go test ./internal/service -v
```  
Покрытие:
```bash  
go test ./internal/service -cover
```  
`Текущее покрытие: 73.3%`

# Docker
Для Order Service реализован multi-stage Dockerfile, позволяющий собрать минимальный production-образ с бинарником сервиса.