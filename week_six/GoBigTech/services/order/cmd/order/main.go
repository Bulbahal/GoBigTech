package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	platformevents "github.com/bulbahal/GoBigTech/platform/events"
	"github.com/bulbahal/GoBigTech/platform/kafka"
	platformlogger "github.com/bulbahal/GoBigTech/platform/logger"
	"github.com/bulbahal/GoBigTech/platform/shutdown"
	"github.com/bulbahal/GoBigTech/services/order/internal/config"
	"github.com/bulbahal/GoBigTech/services/order/internal/di"
	orderhttp "github.com/bulbahal/GoBigTech/services/order/internal/transport/http"

	orderapi "github.com/bulbahal/GoBigTech/services/order/api"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	shutdownMgr := shutdown.New()

	log := platformlogger.New(cfg.Env)
	defer func() { _ = log.Sync() }()

	c := di.New(cfg)
	svc := c.OrderService(ctx)

	consumer := c.KafkaConsumerAssembly()
	consumerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		err := consumer.Run(consumerCtx, func(ctx context.Context, msg kafka.Message) error {
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
	log.Info("assembly consumer started", zap.String("topic", platformevents.TopicOrderAssemblyCompleted))

	h := &orderhttp.Handler{Service: svc}
	r := chi.NewRouter()
	orderapi.HandlerFromMux(h, r)

	addr := ":" + cfg.HttpPort
	srv := &http.Server{Addr: addr, Handler: r}

	shutdownMgr.Add(func(ctx context.Context) error {
		log.Info("stopping http server", zap.String("addr", addr))
		return srv.Shutdown(ctx)
	})

	shutdownMgr.Add(c.Close)

	go func() {
		log.Info("order listening", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen error", zap.Error(err))
		}
	}()

	sig := shutdown.WaitSignal()
	log.Info("shutdown signal received", zap.Any("signal", sig))

	cancel()
	if err := shutdownMgr.Shutdown(10 * time.Second); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}

	log.Info("order stopped gracefully")
}
