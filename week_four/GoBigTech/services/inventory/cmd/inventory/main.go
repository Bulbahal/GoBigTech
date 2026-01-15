package main

import (
	"context"
	platformhealth "github.com/bulbahal/GoBigTech/platform/health"
	"github.com/bulbahal/GoBigTech/platform/shutdown"
	"github.com/bulbahal/GoBigTech/services/inventory/internal/repository"
	inventorypb "github.com/bulbahal/GoBigTech/services/inventory/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"net"
	"time"
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

	shutdownMgr := shutdown.New()

	mongoClient, err := repository.ConnectMongo(ctx, config.MongoURI)
	if err != nil {
		log.Fatalf("mongo connect error: %v", err)
	}
	shutdownMgr.Add(func(ctx context.Context) error {
		log.Println("closing mongo")
		return mongoClient.Disconnect(ctx)
	})

	repo := repository.NewMongoInventoryRepository(mongoClient, config.MongoDB)
	_ = repo.SetStock(ctx, "p1", 10)
	_ = repo.SetStock(ctx, "p2", 10)

	l, err := net.Listen("tcp", config.GRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	shutdownMgr.Add(func(ctx context.Context) error {
		log.Println("closing listener")
		return l.Close()
	})

	g := grpc.NewServer()
	inventorypb.RegisterInventoryServiceServer(g, &server{repo: repo})

	platformhealth.RegisterGRPC(g)

	shutdownMgr.Add(func(ctx context.Context) error {
		log.Println("stopping grpc server")
		g.GracefulStop()
		return nil
	})
	go func() {
		log.Printf("inventory grpc server listening on %s", config.GRPCAddr)
		if err := g.Serve(l); err != nil {
			log.Printf("grpc serve error: %v", err)
		}
	}()

	sig := shutdown.WaitSignal()
	log.Printf("shutdown signal received: %v", sig)

	if err := shutdownMgr.Shutdown(10 * time.Second); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	log.Println("inventory stopped gracefully")
}
