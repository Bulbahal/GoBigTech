package main

import (
	"context"
	"time"

	platformhealth "github.com/bulbahal/GoBigTech/platform/health"
	"github.com/bulbahal/GoBigTech/platform/shutdown"
	"github.com/bulbahal/GoBigTech/services/iam/internal/config"
	"github.com/bulbahal/GoBigTech/services/iam/internal/di"
	iampb "github.com/bulbahal/GoBigTech/services/iam/v1"

	"go.uber.org/zap"
)

// main собирает все слои IAM Service: конфиг, DI-контейнер, gRPC-сервер и graceful shutdown.
func main() {
	ctx := context.Background()
	cfg := config.Load()

	shutdownMgr := shutdown.New()

	// Создаём DI-контейнер, из которого берём все зависимости.
	c := di.New(cfg)
	log := c.Logger()

	lis := c.Listener()
	srv := c.GRPCServer()

	// Регистрируем gRPC-сервер IAM через сгенерированный код.
	iampb.RegisterIAMServiceServer(srv, c.Handler(ctx))
	platformhealth.RegisterGRPC(srv)

	// При завершении сервиса корректно закрываем все ресурсы через контейнер.
	shutdownMgr.Add(c.Close)

	go func() {
		log.Info("iam grpc server listening", zap.String("address", cfg.GRPCAddr))
		if err := srv.Serve(lis); err != nil {
			log.Error("grpc serve error", zap.Error(err))
		}
	}()

	// Ждём SIGINT/SIGTERM и инициируем graceful shutdown.
	sig := shutdown.WaitSignal()
	log.Info("shutdown signal received", zap.String("signal", sig.String()))

	if err := shutdownMgr.Shutdown(10 * time.Second); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}

	log.Info("iam stopped gracefully")
}
