package grpc

import (
	"context"

	paymentpb "github.com/bulbahal/GoBigTech/services/payment/v1"
)

type Server struct {
	paymentpb.UnimplementedPaymentServiceServer
}

func (s *Server) ProcessPayment(ctx context.Context, req *paymentpb.ProcessPaymentRequest) (*paymentpb.ProcessPaymentResponse, error) {
	return &paymentpb.ProcessPaymentResponse{
		Success:       true,
		TransactionId: "tx_123",
	}, nil
}
