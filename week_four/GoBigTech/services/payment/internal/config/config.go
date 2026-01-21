package config

import "os"

type Config struct {
	Env      string
	GRPCAddr string
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func Load() Config {
	return Config{
		Env:      getEnv("ENV", "dev"),
		GRPCAddr: getEnv("GRPC_ADDR", "127.0.0.1:50052"),
	}
}
