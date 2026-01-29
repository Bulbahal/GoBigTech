package main

import (
	"context"
	"time"

	platformhealth "github.com/bulbahal/GoBigTech/platform/health"
	"github.com/bulbahal/GoBigTech/platform/shutdown"
	"github.com/bulbahal/GoBigTech/services/payment/internal/config"
	"github.com/bulbahal/GoBigTech/services/payment/internal/di"
	paymentpb "github.com/bulbahal/GoBigTech/services/payment/v1"

	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	shutdownMgr := shutdown.New()

	c := di.New(cfg)
	log := c.Logger()

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
