# Сборка IAM Service из корня монорепозитория (context: .): go.work + platform + services/*
FROM golang:1.24.4 AS builder

WORKDIR /app
COPY . .

WORKDIR /app/services/iam
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o iam-service ./cmd/iam

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && update-ca-certificates
RUN adduser -D appuser

WORKDIR /app
COPY --from=builder /app/services/iam/iam-service .

EXPOSE 50060
USER appuser
CMD ["./iam-service"]

