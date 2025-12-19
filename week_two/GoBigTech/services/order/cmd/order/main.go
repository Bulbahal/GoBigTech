package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"

	inventorypb "github.com/bulbahal/GoBigTech/services/inventory/v1"
	orderapi "github.com/bulbahal/GoBigTech/services/order/api"
	paymentpb "github.com/bulbahal/GoBigTech/services/payment/v1"

	"github.com/bulbahal/GoBigTech/services/order/internal/service"
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
	connInv, err := grpc.Dial("127.0.0.1:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect inventory: %v", err)
	}
	defer connInv.Close()

	connPay, err := grpc.Dial("127.0.0.1:50052", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect payment: %v", err)
	}
	defer connPay.Close()

	invNative := inventorypb.NewInventoryServiceClient(connInv)
	payNative := paymentpb.NewPaymentServiceClient(connPay)

	invClient := &InventoryClientAdapter{client: invNative}
	payClient := &PaymentClientAdapter{client: payNative}

	svc := service.NewOrderService(invClient, payClient)

	h := &orderhttp.Handler{
		Service: svc,
	}

	r := chi.NewRouter()
	orderapi.HandlerFromMux(h, r)

	log.Println("order listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
