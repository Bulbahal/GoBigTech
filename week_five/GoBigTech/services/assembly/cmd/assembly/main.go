package main

import (
	"context"
	"encoding/json"
	"time"

	platformevents "github.com/bulbahal/GoBigTech/platform/events"
	"github.com/bulbahal/GoBigTech/platform/kafka"
	"github.com/bulbahal/GoBigTech/platform/shutdown"
	"github.com/bulbahal/GoBigTech/services/assembly/internal/config"
	"github.com/bulbahal/GoBigTech/services/assembly/internal/di"

	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	shutdownMgr := shutdown.New()

	c := di.New(cfg)
	log := c.Logger()

	consumer := c.KafkaConsumer()
	svc := c.AssemblyService()

	shutdownMgr.Add(c.Close)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := consumer.Run(ctx, func(ctx context.Context, msg kafka.Message) error {
			var event platformevents.OrderPaymentCompleted
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Error("unmarshal payment event", zap.Error(err), zap.ByteString("value", msg.Value))
				return err
			}
			return svc.HandlePaymentCompleted(ctx, event)
		})
		if err != nil && ctx.Err() == nil {
			log.Error("consumer stopped", zap.Error(err))
		}
	}()

	log.Info("assembly consumer started", zap.String("topic", platformevents.TopicOrderPaymentCompleted))

	sig := shutdown.WaitSignal()
	log.Info("shutdown signal received", zap.String("signal", sig.String()))

	cancel()
	if err := shutdownMgr.Shutdown(10 * time.Second); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}

	log.Info("assembly stopped gracefully")
}
