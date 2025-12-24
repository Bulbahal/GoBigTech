package config

import "os"

type Config struct {
	HttpPort string

	PostgresDSN   string
	InventoryAddr string
	PaymentAddr   string
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func Load() Config {
	return Config{
		HttpPort:      getEnv("HTTP_PORT", "8080"),
		PostgresDSN:   getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/appdb?sslmode=disable"),
		InventoryAddr: getEnv("INVENTORY_GRPC_ADDR", "localhost:50051"),
		PaymentAddr:   getEnv("PAYMENT_GRPC_ADDR", "localhost:50052"),
	}
}
