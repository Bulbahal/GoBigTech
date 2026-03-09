package di

import (
	"context"
	"strings"

	platformevents "github.com/bulbahal/GoBigTech/platform/events"
	platformkafka "github.com/bulbahal/GoBigTech/platform/kafka"
	platformlogger "github.com/bulbahal/GoBigTech/platform/logger"
	"github.com/bulbahal/GoBigTech/services/notification/internal/config"
	"github.com/bulbahal/GoBigTech/services/notification/internal/service"
	"github.com/bulbahal/GoBigTech/services/notification/internal/telegram"

	"go.uber.org/zap"
)

// Container — DI-контейнер: создаёт логгер, Telegram-отправителя, консьюмеров Kafka и сервис уведомлений.
type Container struct {
	cfg *config.Config

	log *zap.Logger

	telegramSender telegram.Sender

	consumerPayment *platformkafka.Consumer
	consumerAssembly *platformkafka.Consumer

	svc service.NotificationService
}

func New(cfg config.Config) *Container {
	return &Container{cfg: &cfg}
}

func (c *Container) Logger() *zap.Logger {
	if c.log == nil {
		c.log = platformlogger.New(c.cfg.Env)
	}
	return c.log
}

// TelegramSender возвращает отправителя сообщений в Telegram (один на сервис, без состояния для закрытия).
func (c *Container) TelegramSender() telegram.Sender {
	if c.telegramSender == nil {
		c.telegramSender = telegram.NewSender(c.cfg.TelegramBotToken, c.cfg.TelegramChatID)
	}
	return c.telegramSender
}

// ConsumerPayment — консьюмер топика «оплата завершена», отдельная consumer group для notification.
func (c *Container) ConsumerPayment() *platformkafka.Consumer {
	if c.consumerPayment == nil {
		brokers := strings.Split(c.cfg.KafkaBrokers, ",")
		c.consumerPayment = platformkafka.NewConsumer(
			brokers,
			"notification-payment",
			[]string{platformevents.TopicOrderPaymentCompleted},
			c.Logger(),
		)
	}
	return c.consumerPayment
}

// ConsumerAssembly — консьюмер топика «сборка завершена».
func (c *Container) ConsumerAssembly() *platformkafka.Consumer {
	if c.consumerAssembly == nil {
		brokers := strings.Split(c.cfg.KafkaBrokers, ",")
		c.consumerAssembly = platformkafka.NewConsumer(
			brokers,
			"notification-assembly",
			[]string{platformevents.TopicOrderAssemblyCompleted},
			c.Logger(),
		)
	}
	return c.consumerAssembly
}

// NotificationService возвращает бизнес-сервис уведомлений (шаблоны + Telegram).
func (c *Container) NotificationService() service.NotificationService {
	if c.svc == nil {
		c.svc = service.NewNotificationService(c.TelegramSender())
	}
	return c.svc
}

// Close закрывает консьюмеры Kafka и синхронизирует лог (Telegram sender не держит соединений).
func (c *Container) Close(ctx context.Context) error {
	if c.consumerPayment != nil {
		_ = c.consumerPayment.Close()
	}
	if c.consumerAssembly != nil {
		_ = c.consumerAssembly.Close()
	}
	if c.log != nil {
		_ = c.log.Sync()
	}
	return nil
}
