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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.Load()
	shutdownMgr := shutdown.New()

	c := di.New(cfg)
	log := c.Logger()

	// OTEL: метрики для сборки, отправляем в otel-collector по OTLP.
	ctx := context.Background()
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", "assembly-service"),
		),
	)
	if err != nil {
		log.Fatal("resource.New", zap.Error(err))
	}
	metricExp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint("otel-collector:4317"),
		otlpmetricgrpc.WithTLSCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal("otlpmetricgrpc.New", zap.Error(err))
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	shutdownMgr.Add(func(ctx context.Context) error {
		return mp.Shutdown(ctx)
	})

	consumer := c.KafkaConsumer()
	svc := c.AssemblyService()

	shutdownMgr.Add(c.Close)

	ctx, cancel := context.WithCancel(ctx)
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
