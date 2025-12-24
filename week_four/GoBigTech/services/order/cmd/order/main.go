package main

import (
	"context"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
	"log"
	"net/http"

	inventorypb "github.com/bulbahal/GoBigTech/services/inventory/v1"
	orderapi "github.com/bulbahal/GoBigTech/services/order/api"
	paymentpb "github.com/bulbahal/GoBigTech/services/payment/v1"

	"github.com/bulbahal/GoBigTech/services/order/internal/config"
	"github.com/bulbahal/GoBigTech/services/order/internal/repository"
	"github.com/bulbahal/GoBigTech/services/order/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"

	orderhttp "github.com/bulbahal/GoBigTech/services/order/internal/transport/http"
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

	pool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	connInv, err := grpc.Dial(cfg.InventoryAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect inventory: %v", err)
	}
	defer connInv.Close()

	connPay, err := grpc.Dial(cfg.PaymentAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect payment: %v", err)
	}
	defer connPay.Close()

	invNative := inventorypb.NewInventoryServiceClient(connInv)
	payNative := paymentpb.NewPaymentServiceClient(connPay)

	invClient := &InventoryClientAdapter{client: invNative}
	payClient := &PaymentClientAdapter{client: payNative}

	repo := repository.NewPostgresRepository(pool)

	svc := service.NewOrderService(invClient, payClient, repo)

	h := &orderhttp.Handler{
		Service: svc,
	}

	r := chi.NewRouter()
	orderapi.HandlerFromMux(h, r)

	addr := ":" + cfg.HttpPort
	log.Printf("order listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
