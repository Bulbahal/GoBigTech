package di

import (
	"context"
	"net"

	platformlogger "github.com/bulbahal/GoBigTech/platform/logger"
	"github.com/bulbahal/GoBigTech/services/inventory/internal/config"
	"github.com/bulbahal/GoBigTech/services/inventory/internal/repository"
	transportgrpc "github.com/bulbahal/GoBigTech/services/inventory/internal/transport/grpc"
	inventorypb "github.com/bulbahal/GoBigTech/services/inventory/v1"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Container struct {
	cfg *config.Config

	log *zap.Logger

	mongo *mongo.Client
	repo  *repository.MongoInventoryRepository

	lis    net.Listener
	server *grpc.Server

	handler inventorypb.InventoryServiceServer
}

func New(cfg config.Config) *Container {
	return &Container{cfg: &cfg}
}

func (c *Container) Logger() *zap.Logger {
	if c.log == nil {
		c.log = platformlogger.New(c.cfg.Env)
	}
	return c.log
}

func (c *Container) Mongo(ctx context.Context) *mongo.Client {
	if c.mongo == nil {
		client, err := repository.ConnectMongo(ctx, c.cfg.MongoURI)
		if err != nil {
			c.Logger().Fatal("mongo connect", zap.Error(err))
		}
		c.mongo = client
	}
	return c.mongo
}

func (c *Container) Repository(ctx context.Context) *repository.MongoInventoryRepository {
	if c.repo == nil {
		c.repo = repository.NewMongoInventoryRepository(c.Mongo(ctx), c.cfg.MongoDB)
	}
	return c.repo
}

func (c *Container) GRPCListener() net.Listener {
	if c.lis == nil {
		l, err := net.Listen("tcp", c.cfg.GRPCAddr)
		if err != nil {
			c.Logger().Fatal("grpc listen", zap.Error(err))
		}
		c.lis = l
	}
	return c.lis
}

func (c *Container) GRPCServer() *grpc.Server {
	if c.server == nil {
		c.server = grpc.NewServer()
	}
	return c.server
}

func (c *Container) Handler(ctx context.Context) inventorypb.InventoryServiceServer {
	if c.handler == nil {
		c.handler = &transportgrpc.Server{Repo: c.Repository(ctx)}
	}
	return c.handler
}

func (c *Container) Close(ctx context.Context) error {
	if c.server != nil {
		c.server.GracefulStop()
	}
	if c.lis != nil {
		_ = c.lis.Close()
	}
	if c.mongo != nil {
		_ = c.mongo.Disconnect(ctx)
	}
	if c.log != nil {
		_ = c.log.Sync()
	}
	return nil
}
