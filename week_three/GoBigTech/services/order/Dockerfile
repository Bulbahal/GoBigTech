FROM golang:1.24.4 AS builder

WORKDIR /app

COPY order ./order
COPY inventory ./inventory
COPY payment ./payment

WORKDIR /app/order

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o order-service ./cmd/order

FROM alpine:3.20

RUN apk add --no-cache ca-certificates && update-ca-certificates

RUN adduser -D appuser

WORKDIR /app

COPY --from=builder /app/order/order-service .

EXPOSE 8080

USER appuser

CMD ["./order-service"]