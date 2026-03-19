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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	shutdownMgr := shutdown.New()

	log := platformlogger.New(cfg.Env)
	defer func() { _ = log.Sync() }()

	// Инициализация OpenTelemetry: OTLP gRPC в collector, resource service.name=order-service.
	res, err := resource.New(ctx, resource.WithAttributes(attribute.String("service.name", "order-service")))
	if err != nil {
		log.Fatal("resource.New", zap.Error(err))
	}
	// Трейсинг.
	traceExp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("otel-collector:4317"),
		otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal("otlptracegrpc.New", zap.Error(err))
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	shutdownMgr.Add(func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	})

	// Метрики.
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
	// Минимальный HTTP-трейсинг: каждый входящий запрос создаёт span и уходит в Jaeger.
	httpHandler := otelhttp.NewHandler(r, "order-http")

	addr := ":" + cfg.HttpPort
	srv := &http.Server{Addr: addr, Handler: httpHandler}

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
