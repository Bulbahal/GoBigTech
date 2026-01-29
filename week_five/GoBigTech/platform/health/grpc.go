package health

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func RegisterGRPC(s *grpc.Server) *health.Server {
	h := health.NewServer()
	healthpb.RegisterHealthServer(s, h)
	return h
}
