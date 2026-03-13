package config

import "os"

// Config описывает настройки IAM Service.
// В реальном проекте такие параметры обычно приходят из переменных окружения или файла конфигурации.
type Config struct {
	// Env — окружение (local/dev/stage/prod) для настройки логгера и других компонентов.
	Env string

	// GRPCAddr — адрес, на котором будет слушать gRPC-сервер IAM.
	GRPCAddr string

	// PostgresDSN — строка подключения к PostgreSQL, где хранятся пользователи.
	PostgresDSN string

	// RedisAddr — адрес Redis, где будут храниться сессии (обычно host:port, например redis:6379).
	RedisAddr string
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Load загружает конфигурацию из переменных окружения с безопасными значениями по умолчанию.
func Load() Config {
	return Config{
		Env:        getEnv("ENV", "local"),
		GRPCAddr:   getEnv("GRPC_ADDR", "0.0.0.0:50060"),
		PostgresDSN: getEnv("POSTGRES_DSN", "postgres://postgres:postgres@db:5432/iamdb?sslmode=disable"),
		RedisAddr:  getEnv("REDIS_ADDR", "redis:6379"),
	}
}

