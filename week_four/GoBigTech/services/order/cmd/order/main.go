package main

import (
	"context"
	"github.com/bulbahal/GoBigTech/platform/shutdown"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	inventorypb "github.com/bulbahal/GoBigTech/services/inventory/v1"
	orderapi "github.com/bulbahal/GoBigTech/services/order/api"
	paymentpb "github.com/bulbahal/GoBigTech/services/payment/v1"

	platformlogger "github.com/bulbahal/GoBigTech/platform/logger"
	"github.com/bulbahal/GoBigTech/services/order/internal/config"
	"github.com/bulbahal/GoBigTech/services/order/internal/repository"
	"github.com/bulbahal/GoBigTech/services/order/internal/service"
	orderhttp "github.com/bulbahal/GoBigTech/services/order/internal/transport/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type InventoryClientAdapter struct {
	client inventorypb.InventoryServiceClient
}

func (i *InventoryClientAdapter) ReserveStock(ctx context.Context, productID string, qty int32) error {
	_, err := i.client.ReserveStock(ctx, &inventorypb.ReserveStockRequest{
		ProductId: productID,
		Quantity:  qty,
	})
	return err
}

type PaymentClientAdapter struct {
	client paymentpb.PaymentServiceClient
}

func (p *PaymentClientAdapter) ProcessPayment(ctx context.Context, orderID, userID string, amount float64, method string) error {
	_, err := p.client.ProcessPayment(ctx, &paymentpb.ProcessPaymentRequest{
		OrderId: orderID,
		UserId:  userID,
		Amount:  amount,
		Method:  method,
	})
	return err
}

func main() {
	ctx := context.Background()
	cfg := config.Load()

	shutdownMgr := shutdown.New()

	log := platformlogger.New(cfg.Env)
	defer func() { _ = log.Sync() }()

	pool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatal("pgxpool new", zap.Error(err))
	}
	shutdownMgr.Add(func(ctx context.Context) error {
		log.Info("closing pg pool")
		pool.Close()
		return nil
	})

	connInv, err := grpc.Dial(cfg.InventoryAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal("connect inventory", zap.Error(err))
	}
	shutdownMgr.Add(func(ctx context.Context) error {
		log.Info("closing inventory grpc connection")
		return connInv.Close()
	})

	connPay, err := grpc.Dial(cfg.PaymentAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal("connect payment", zap.Error(err))
	}
	shutdownMgr.Add(func(ctx context.Context) error {
		log.Info("closing payment grpc connection")
		return connPay.Close()
	})

	invClient := &InventoryClientAdapter{client: inventorypb.NewInventoryServiceClient(connInv)}
	payClient := &PaymentClientAdapter{client: paymentpb.NewPaymentServiceClient(connPay)}

	repo := repository.NewPostgresRepository(pool)
	svc := service.NewOrderService(invClient, payClient, repo)

	h := &orderhttp.Handler{Service: svc}
	r := chi.NewRouter()
	orderapi.HandlerFromMux(h, r)

	addr := ":" + cfg.HttpPort
	srv := &http.Server{Addr: addr, Handler: r}

	shutdownMgr.Add(func(ctx context.Context) error {
		log.Info("stopping http server")
		return srv.Shutdown(ctx)
	})
	go func() {
		log.Info("order listening", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("http server error", zap.Error(err))
		}
	}()

	sig := shutdown.WaitSignal()
	log.Info("shutdown signal received", zap.Any("signal", sig))

	if err := shutdownMgr.Shutdown(10 * time.Second); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}

	log.Info("order stopped gracefully")
}
