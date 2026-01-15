package main

import (
	"context"
	platformhealth "github.com/bulbahal/GoBigTech/platform/health"
	"github.com/bulbahal/GoBigTech/platform/shutdown"
	paymentpb "github.com/bulbahal/GoBigTech/services/payment/v1"
	"google.golang.org/grpc"
	"log"
	"net"
	"time"
)

type server struct {
	paymentpb.UnimplementedPaymentServiceServer
}

func (s *server) ProcessPayment(ctx context.Context, req *paymentpb.ProcessPaymentRequest) (*paymentpb.ProcessPaymentResponse, error) {
	return &paymentpb.ProcessPaymentResponse{Success: true, TransactionId: "tx_123"}, nil
}

func main() {
	cfg := LoadConfig()
	shutdownMgr := shutdown.New()

	l, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	shutdownMgr.Add(func(ctx context.Context) error {
		log.Println("closing listener")
		return l.Close()
	})

	grpcServer := grpc.NewServer()
	paymentpb.RegisterPaymentServiceServer(grpcServer, &server{})
	platformhealth.RegisterGRPC(grpcServer)

	shutdownMgr.Add(func(ctx context.Context) error {
		log.Println("stopping grpc server")
		grpcServer.GracefulStop()
		return nil
	})

	go func() {
		log.Printf("payment grpc server listening on %s", cfg.GRPCAddr)
		if err := grpcServer.Serve(l); err != nil {
			log.Printf("grpc serve error: %v", err)
		}
	}()

	sig := shutdown.WaitSignal()
	log.Printf("shutdown signal received: %v", sig)

	if err := shutdownMgr.Shutdown(10 * time.Second); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	log.Println("payment stopped gracefully")
}
