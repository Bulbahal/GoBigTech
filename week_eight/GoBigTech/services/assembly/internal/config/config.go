package config

import "os"

type Config struct {
	Env          string
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
		Env:          getEnv("ENV", "local"),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "kafka:9092"),
	}
}
