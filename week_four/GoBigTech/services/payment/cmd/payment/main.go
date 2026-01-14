package main

import (
	"context"
	platformhealth "github.com/bulbahal/GoBigTech/platform/health"
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

	l, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	paymentpb.RegisterPaymentServiceServer(grpcServer, &server{})

	platformhealth.RegisterGRPC(grpcServer)

	log.Printf("payment listening on %s", cfg.GRPCAddr)
	log.Fatal(grpcServer.Serve(l))
}
