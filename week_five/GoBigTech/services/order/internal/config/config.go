package config

import "os"

type Config struct {
	HttpPort string
	Env      string

	PostgresDSN   string
	InventoryAddr string
	PaymentAddr   string

	KafkaBrokers string
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func Load() Config {
	return Config{
		Env:           getEnv("ENV", "local"),
		HttpPort:      getEnv("HTTP_PORT", "8080"),
		PostgresDSN:   getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/appdb?sslmode=disable"),
		InventoryAddr: getEnv("INVENTORY_GRPC_ADDR", "localhost:50051"),
		PaymentAddr:   getEnv("PAYMENT_GRPC_ADDR", "localhost:50052"),
		KafkaBrokers:  getEnv("KAFKA_BROKERS", "kafka:9092"),
	}
}
