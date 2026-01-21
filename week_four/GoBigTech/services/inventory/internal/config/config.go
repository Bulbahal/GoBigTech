package config

import "os"

type Config struct {
	Env      string
	GRPCAddr string
	MongoURI string
	MongoDB  string
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
		GRPCAddr: getEnv("GRPC_ADDR", "127.0.0.1:50051"),
		MongoURI: getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:  getEnv("MONGO_DB", "appdb"),
	}
}
