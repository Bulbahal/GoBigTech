package main

import (
	"context"
	"log"
	"net"

	paymentpb "github.com/bulbahal/GoBigTech/services/payment/v1"
	"google.golang.org/grpc"
)

type server struct {
	paymentpb.UnimplementedPaymentServiceServer
}

func (s *server) ProcessPayment(ctx context.Context, req *paymentpb.ProcessPaymentRequest) (*paymentpb.ProcessPaymentResponse, error) {
	return &paymentpb.ProcessPaymentResponse{Success: true, TransactionId: "tx_123"}, nil
}

func main() {
	l, err := net.Listen("tcp4", "127.0.0.1:50052")
	if err != nil {
		log.Fatal(err)
	}
	g := grpc.NewServer()
	paymentpb.RegisterPaymentServiceServer(g, &server{})
	log.Println("payment listening on 127.0.0.1:50052")
	log.Fatal(g.Serve(l))
}
