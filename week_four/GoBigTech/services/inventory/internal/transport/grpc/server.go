package grpc

import (
	"context"

	"github.com/bulbahal/GoBigTech/services/inventory/internal/repository"
	inventorypb "github.com/bulbahal/GoBigTech/services/inventory/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	inventorypb.UnimplementedInventoryServiceServer
	Repo *repository.MongoInventoryRepository
}

func (s *Server) GetStock(ctx context.Context, req *inventorypb.GetStockRequest) (*inventorypb.GetStockResponse, error) {
	qty, err := s.Repo.Get(ctx, req.GetProductId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "mongo get: %v", err)
	}
	return &inventorypb.GetStockResponse{
		ProductId: req.GetProductId(),
		Available: qty,
	}, nil
}

func (s *Server) ReserveStock(ctx context.Context, req *inventorypb.ReserveStockRequest) (*inventorypb.ReserveStockResponse, error) {
	if req.GetQuantity() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "quantity must be greater than 0")
	}

	err := s.Repo.Reserve(ctx, req.GetProductId(), req.GetQuantity())
	if err != nil {
		if err == repository.ErrNotEnoughStock {
			return nil, status.Error(codes.FailedPrecondition, "not enough stock")
		}
		return nil, status.Errorf(codes.Internal, "mongo reserve: %v", err)
	}
	return &inventorypb.ReserveStockResponse{Success: true}, nil
}
