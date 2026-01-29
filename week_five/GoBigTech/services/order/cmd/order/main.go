package main

import (
	"context"
	"net/http"
	"time"

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

	if err := shutdownMgr.Shutdown(10 * time.Second); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}

	log.Info("order stopped gracefully")
}
