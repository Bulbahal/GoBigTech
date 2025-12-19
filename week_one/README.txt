oapi-codegen -config tools/openapi.oapi.yaml api/openapi/order.yaml
buf generate api/proto --template tools/buf.gen.yaml

buf breaking api/proto --against '.git#ref=HEAD'

cd services/inventory && go run ./cmd/inventory
cd services/payment   && go run ./cmd/payment
cd services/order     && go run ./cmd/order

curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id":"u1","items":[{"product_id":"p1","quantity":2}]}'