    Order Service
Сервис отвечает за создание заказов. 
Работает поверх Inventory Service и Payment Service.

    Архитектура
Сервис построен по трёхслойной структуре:
transport (HTTP) — принимает запросы от клиента
service (бизнес-логика) — проверка данных, вызовы зависимостей
clients — gRPC-адаптеры для Inventory и Payment

    Как запустить
Перед запуском заказа нужно стартовать зависимости:
cd services/inventory && go run ./cmd/inventory
cd services/payment   && go run ./cmd/payment
Теперь запускаем Order Service:
cd services/order && go run ./cmd/order
Сервис будет доступен на:
http://localhost:8080

    Пример запроса
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id":"u1","items":[{"product_id":"p1","quantity":2}]}'

    Тесты
Запуск тестов:
go test ./internal/service -v
Покрытие:
go test ./internal/service -cover
Текущее покрытие: 66.7%