package di

import (
	"context"
	"strings"

	platformevents "github.com/bulbahal/GoBigTech/platform/events"
	platformkafka "github.com/bulbahal/GoBigTech/platform/kafka"
	platformlogger "github.com/bulbahal/GoBigTech/platform/logger"
	"github.com/bulbahal/GoBigTech/services/assembly/internal/config"
	"github.com/bulbahal/GoBigTech/services/assembly/internal/service"

	"go.uber.org/zap"
)

type Container struct {
	cfg *config.Config

	log *zap.Logger

	consumer *platformkafka.Consumer
	producer *platformkafka.Producer

	svc service.AssemblyService
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

func (c *Container) KafkaConsumer() *platformkafka.Consumer {
	if c.consumer == nil {
		brokers := strings.Split(c.cfg.KafkaBrokers, ",")
		c.consumer = platformkafka.NewConsumer(
			brokers,
			"assembly-payment",
			[]string{platformevents.TopicOrderPaymentCompleted},
			c.Logger(),
		)
	}
	return c.consumer
}

func (c *Container) KafkaProducer() *platformkafka.Producer {
	if c.producer == nil {
		brokers := strings.Split(c.cfg.KafkaBrokers, ",")
		c.producer = platformkafka.NewProducer(brokers, c.Logger())
	}
	return c.producer
}

func (c *Container) AssemblyService() service.AssemblyService {
	if c.svc == nil {
		c.svc = service.NewAssemblyService(c.KafkaProducer())
	}
	return c.svc
}

func (c *Container) Close(ctx context.Context) error {
	if c.consumer != nil {
		_ = c.consumer.Close()
	}
	if c.producer != nil {
		_ = c.producer.Close()
	}
	if c.log != nil {
		_ = c.log.Sync()
	}
	return nil
}
