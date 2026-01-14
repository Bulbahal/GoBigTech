package main

import (
	"context"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"

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

	log := platformlogger.New(cfg.Env)
	defer func() { _ = log.Sync() }()

	pool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatal("pgxpool new", zap.Error(err))
	}
	defer pool.Close()

	connInv, err := grpc.Dial(cfg.InventoryAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal("connect inventory", zap.Error(err))
	}
	defer connInv.Close()

	connPay, err := grpc.Dial(cfg.PaymentAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal("connect payment", zap.Error(err))
	}
	defer connPay.Close()

	invClient := &InventoryClientAdapter{client: inventorypb.NewInventoryServiceClient(connInv)}
	payClient := &PaymentClientAdapter{client: paymentpb.NewPaymentServiceClient(connPay)}

	repo := repository.NewPostgresRepository(pool)
	svc := service.NewOrderService(invClient, payClient, repo)

	h := &orderhttp.Handler{Service: svc}
	r := chi.NewRouter()
	orderapi.HandlerFromMux(h, r)

	addr := ":" + cfg.HttpPort
	log.Info("order listening", zap.String("addr", addr))

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal("http server error", zap.Error(err))
	}
}
