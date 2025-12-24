package main

import (
	"context"
	paymentpb "github.com/bulbahal/GoBigTech/services/payment/v1"
	"google.golang.org/grpc"
	"log"
	"net"
)

type server struct {
	paymentpb.UnimplementedPaymentServiceServer
}

func (s *server) ProcessPayment(ctx context.Context, req *paymentpb.ProcessPaymentRequest) (*paymentpb.ProcessPaymentResponse, error) {
	return &paymentpb.ProcessPaymentResponse{Success: true, TransactionId: "tx_123"}, nil
}

func main() {
	cfg := LoadConfig()

	l, err := net.Listen("tcp4", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	paymentpb.RegisterPaymentServiceServer(grpcServer, &server{})

	log.Printf("Listening on %s", cfg.GRPCAddr)
	log.Fatal(grpcServer.Serve(l))
}
