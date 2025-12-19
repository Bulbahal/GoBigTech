# Order Service
Сервис отвечает за создание заказов.
Работает поверх Inventory Service и Payment Service.
Сервис реализован как HTTP API и является частью микросервисной системы.

# Архитектура
Сервис построен по слоистой архитектуре:
•	transport (HTTP) — принимает и валидирует входящие запросы
•	service (business logic) — бизнес-правила, orchestration, транзакционный поток
•	repository — работа с PostgreSQL
•	clients — gRPC-адаптеры для Inventory и Payment сервисов
Поток запроса:
```
HTTP → Handler → Service → Repository (Postgres)
                    ↓
              Inventory (MongoDB)
                    ↓
               Payment (gRPC)
```
# Хранилища данных
•PostgreSQL — хранение заказов и позиций заказа (реляционные данные, транзакции)
•MongoDB — хранение и резервирование складских остатков (частые обновления, простая структура)
Миграции PostgreSQL выполняются с помощью goose.

# Запуск проекта
1. Поднять инфраструктуру (Postgres + MongoDB):
```bash
docker-compose up -d
```
1. Применить миграции PostgreSQL:
```bash
goose -dir ./migrations postgres "postgres://postgres:postgres@localhost:5432/appdb?sslmode=disable" up
```
1. Запустить сервисы:
```bash
cd services/inventory && go run ./cmd/inventory
cd services/payment   && go run ./cmd/payment
cd services/order     && go run ./cmd/order
```

Order Service будет доступен по адресу:
``http://localhost:8080

# Пример запроса
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id":"u1","items":[{"product_id":"p1","quantity":2}]}'
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