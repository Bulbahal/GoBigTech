package main

import (
	"context"
	"time"

	platformhealth "github.com/bulbahal/GoBigTech/platform/health"
	"github.com/bulbahal/GoBigTech/platform/shutdown"
	"github.com/bulbahal/GoBigTech/services/payment/internal/config"
	"github.com/bulbahal/GoBigTech/services/payment/internal/di"
	paymentpb "github.com/bulbahal/GoBigTech/services/payment/v1"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	shutdownMgr := shutdown.New()

	c := di.New(cfg)
	log := c.Logger()

	// OTEL: service.name=payment-service, OTLP -> otel-collector:4317.
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", "payment-service"),
		),
	)
	if err != nil {
		log.Fatal("resource.New", zap.Error(err))
	}
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("otel-collector:4317"),
		otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal("otlptracegrpc.New", zap.Error(err))
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	shutdownMgr.Add(func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	})

	l := c.Listener()
	g := c.GRPCServer()

	paymentpb.RegisterPaymentServiceServer(g, c.Handler(ctx))
	platformhealth.RegisterGRPC(g)

	shutdownMgr.Add(c.Close)

	go func() {
		log.Info("payment grpc server listening", zap.String("address", cfg.GRPCAddr))
		if err := g.Serve(l); err != nil {
			log.Error("grpc serve error", zap.Error(err))
		}
	}()

	sig := shutdown.WaitSignal()
	log.Info("shutdown signal received", zap.String("signal", sig.String()))

	if err := shutdownMgr.Shutdown(10 * time.Second); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}

	log.Info("payment stopped gracefully")
}
