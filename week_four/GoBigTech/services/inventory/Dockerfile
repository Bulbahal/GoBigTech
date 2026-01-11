FROM golang:1.24.4 AS builder

WORKDIR /app
COPY inventory ./inventory
WORKDIR /app/inventory

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o inventory-service ./cmd/inventory

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && update-ca-certificates
RUN adduser -D appuser

WORKDIR /app
COPY --from=builder /app/inventory/inventory-service .

EXPOSE 50051
USER appuser
CMD ["./inventory-service"]