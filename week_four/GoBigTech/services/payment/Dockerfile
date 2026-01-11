FROM golang:1.24.4 AS builder

WORKDIR /app
COPY payment ./payment
WORKDIR /app/payment

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o payment-service ./cmd/payment

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && update-ca-certificates
RUN adduser -D appuser

WORKDIR /app
COPY --from=builder /app/payment/payment-service .

EXPOSE 50052
USER appuser
CMD ["./payment-service"]