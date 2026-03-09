package config

import "os"

// Config — настройки сервиса уведомлений (Kafka, Telegram).
type Config struct {
	Env string

	// KafkaBrokers — адреса брокеров для консьюмеров (например "kafka:9092" или "host1:9092,host2:9092").
	KafkaBrokers string

	// TELEGRAM_BOT_TOKEN и TELEGRAM_CHAT_ID задаются через env; без них отправка в Telegram не работает.
	TelegramBotToken string
	TelegramChatID   string
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func Load() Config {
	return Config{
		Env:              getEnv("ENV", "local"),
		KafkaBrokers:     getEnv("KAFKA_BROKERS", "kafka:9092"),
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", "8792715861:AAEY23RWOU43ABfn0V_RHNP9FyQJLNfo1mg"),
		TelegramChatID:   getEnv("TELEGRAM_CHAT_ID", "945576559"),
	}
}
