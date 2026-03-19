package main

import (
	"context"
	"encoding/json"
	"time"

	platformevents "github.com/bulbahal/GoBigTech/platform/events"
	"github.com/bulbahal/GoBigTech/platform/kafka"
	"github.com/bulbahal/GoBigTech/platform/shutdown"
	"github.com/bulbahal/GoBigTech/services/notification/internal/config"
	"github.com/bulbahal/GoBigTech/services/notification/internal/di"

	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	shutdownMgr := shutdown.New()

	c := di.New(cfg)
	log := c.Logger()
	svc := c.NotificationService()

	// Регистрируем закрытие консьюмеров и логгера при shutdown.
	shutdownMgr.Add(c.Close)

	// Общий контекст для обоих консьюмеров: при cancel оба выходят из Run().
	consumerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Консьюмер событий «оплата завершена»: декодируем JSON и вызываем HandlePaymentCompleted.
	consumerPayment := c.ConsumerPayment()
	go func() {
		err := consumerPayment.Run(consumerCtx, func(ctx context.Context, msg kafka.Message) error {
			var event platformevents.OrderPaymentCompleted
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Error("unmarshal payment event", zap.Error(err), zap.ByteString("value", msg.Value))
				return err
			}
			return svc.HandlePaymentCompleted(ctx, event)
		})
		if err != nil && consumerCtx.Err() == nil {
			log.Error("payment consumer stopped", zap.Error(err))
		}
	}()
	log.Info("notification consumer started", zap.String("topic", platformevents.TopicOrderPaymentCompleted))

	// Консьюмер событий «сборка завершена»: декодируем JSON и вызываем HandleAssemblyCompleted.
	consumerAssembly := c.ConsumerAssembly()
	go func() {
		err := consumerAssembly.Run(consumerCtx, func(ctx context.Context, msg kafka.Message) error {
			var event platformevents.OrderAssemblyCompleted
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Error("unmarshal assembly event", zap.Error(err), zap.ByteString("value", msg.Value))
				return err
			}
			return svc.HandleAssemblyCompleted(ctx, event)
		})
		if err != nil && consumerCtx.Err() == nil {
			log.Error("assembly consumer stopped", zap.Error(err))
		}
	}()
	log.Info("notification consumer started", zap.String("topic", platformevents.TopicOrderAssemblyCompleted))

	// Ждём SIGINT/SIGTERM, затем отменяем контекст — консьюмеры завершат цикл.
	sig := shutdown.WaitSignal()
	log.Info("shutdown signal received", zap.String("signal", sig.String()))

	cancel()
	if err := shutdownMgr.Shutdown(10 * time.Second); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}

	log.Info("notification stopped gracefully")
}
