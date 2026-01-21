package main

import (
	"context"
	"time"

	platformhealth "github.com/bulbahal/GoBigTech/platform/health"
	"github.com/bulbahal/GoBigTech/platform/shutdown"
	"github.com/bulbahal/GoBigTech/services/inventory/internal/config"
	"github.com/bulbahal/GoBigTech/services/inventory/internal/di"
	inventorypb "github.com/bulbahal/GoBigTech/services/inventory/v1"

	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	shutdownMgr := shutdown.New()

	c := di.New(cfg)
	log := c.Logger()

	repo := c.Repository(ctx)
	_ = repo.SetStock(ctx, "p1", 10)
	_ = repo.SetStock(ctx, "p2", 10)

	l := c.GRPCListener()
	g := c.GRPCServer()

	inventorypb.RegisterInventoryServiceServer(g, c.Handler(ctx))
	platformhealth.RegisterGRPC(g)

	shutdownMgr.Add(c.Close)

	go func() {
		log.Info("inventory grpc server listening", zap.String("address", cfg.GRPCAddr))
		if err := g.Serve(l); err != nil {
			log.Error("grpc serve error", zap.Error(err))
		}
	}()

	sig := shutdown.WaitSignal()
	log.Info("shutdown signal received", zap.String("signal", sig.String()))

	if err := shutdownMgr.Shutdown(10 * time.Second); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}

	log.Info("inventory stopped gracefully")
}
