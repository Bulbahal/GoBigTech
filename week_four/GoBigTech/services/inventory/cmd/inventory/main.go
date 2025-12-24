package main

import (
	"context"
	"github.com/bulbahal/GoBigTech/services/inventory/internal/repository"
	inventorypb "github.com/bulbahal/GoBigTech/services/inventory/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"net"
)

type server struct {
	inventorypb.UnimplementedInventoryServiceServer
	repo *repository.MongoInventoryRepository
}

func (s *server) GetStock(ctx context.Context, req *inventorypb.GetStockRequest) (*inventorypb.GetStockResponse, error) {
	qty, err := s.repo.Get(ctx, req.GetProductId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "mongo get: %v", err)
	}
	return &inventorypb.GetStockResponse{
		ProductId: req.GetProductId(),
		Available: qty,
	}, nil
}
func (s *server) ReserveStock(ctx context.Context, req *inventorypb.ReserveStockRequest) (*inventorypb.ReserveStockResponse, error) {
	if req.GetQuantity() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "quantity must be greater than 0")
	}

	err := s.repo.Reserve(ctx, req.GetProductId(), req.GetQuantity())
	if err != nil {
		if err == repository.ErrNotEnoughStock {
			return nil, status.Error(codes.FailedPrecondition, "not enough stock")
		}
		return nil, status.Errorf(codes.Internal, "mongo reserve: %v", err)
	}
	return &inventorypb.ReserveStockResponse{Success: true}, nil
}

func main() {
	ctx := context.Background()
	config := LoadConfig()

	mongoClient, err := repository.ConnectMongo(ctx, config.MongoURI)
	if err != nil {
		log.Fatalf("mongo connect error: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	repo := repository.NewMongoInventoryRepository(mongoClient, config.MongoDB)
	_ = repo.SetStock(ctx, "p1", 10)
	_ = repo.SetStock(ctx, "p2", 10)

	l, err := net.Listen("tcp", config.GRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	g := grpc.NewServer()
	inventorypb.RegisterInventoryServiceServer(g, &server{repo: repo})

	log.Printf("inventory listening on %s", config.GRPCAddr)
	log.Fatal(g.Serve(l))
}
