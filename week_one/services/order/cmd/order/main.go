package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"

	inventorypb "github.com/bulbahal/GoBigTech/services/inventory/v1"
	orderapi "github.com/bulbahal/GoBigTech/services/order/api"
	paymentpb "github.com/bulbahal/GoBigTech/services/payment/v1"
)

type orderServer struct {
	invClient inventorypb.InventoryServiceClient
	payClient paymentpb.PaymentServiceClient
}

func (s *orderServer) PostOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var body orderapi.CreateOrder
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.UserId == "" || len(body.Items) == 0 {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	first := body.Items[0]
	if _, err := s.invClient.ReserveStock(ctx, &inventorypb.ReserveStockRequest{
		ProductId: first.ProductId,
		Quantity:  int32(first.Quantity),
	}); err != nil {
		http.Error(w, "inventory error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if _, err := s.payClient.ProcessPayment(ctx, &paymentpb.ProcessPaymentRequest{
		OrderId: "order-123",
		UserId:  body.UserId,
		Amount:  100.0,
		Method:  "card",
	}); err != nil {
		http.Error(w, "payment error: "+err.Error(), http.StatusBadGateway)
		return
	}

	id := "order-123"
	status := "paid"
	resp := orderapi.Order{
		Id:     id,
		UserId: body.UserId,
		Status: orderapi.OrderStatus(status),
		Items:  body.Items,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *orderServer) GetOrdersId(w http.ResponseWriter, r *http.Request, id string) {
	status := "paid"
	user := "u1"
	items := []orderapi.OrderItem{
		{ProductId: "p1", Quantity: 2},
	}
	resp := orderapi.Order{
		Id:     id,
		UserId: user,
		Status: orderapi.OrderStatus(status),
		Items:  items,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
func main() {
	connInv, _ := grpc.Dial("127.0.0.1:50051", grpc.WithInsecure())
	defer connInv.Close()
	connPay, _ := grpc.Dial("127.0.0.1:50052", grpc.WithInsecure())
	defer connPay.Close()

	s := &orderServer{
		invClient: inventorypb.NewInventoryServiceClient(connInv),
		payClient: paymentpb.NewPaymentServiceClient(connPay),
	}
	r := chi.NewRouter()
	orderapi.HandlerFromMux(s, r)

	log.Println("order listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
