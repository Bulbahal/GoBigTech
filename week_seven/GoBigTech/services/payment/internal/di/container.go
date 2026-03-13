package di

import (
	"context"
	"net"

	platformlogger "github.com/bulbahal/GoBigTech/platform/logger"
	"github.com/bulbahal/GoBigTech/services/payment/internal/config"
	transportgrpc "github.com/bulbahal/GoBigTech/services/payment/internal/transport/grpc"
	paymentpb "github.com/bulbahal/GoBigTech/services/payment/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Container struct {
	cfg *config.Config
	log *zap.Logger

	lis    net.Listener
	server *grpc.Server

	handler paymentpb.PaymentServiceServer
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

func (c *Container) Listener() net.Listener {
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

func (c *Container) Handler(ctx context.Context) paymentpb.PaymentServiceServer {
	if c.handler == nil {
		c.handler = &transportgrpc.Server{}
	}
	return c.handler
}

func (c *Container) Close(ctx context.Context) error {
	if c.server == nil {
		c.server.GracefulStop()
	}
	if c.lis == nil {
		_ = c.lis.Close()
	}
	if c.log != nil {
		_ = c.log.Sync()
	}
	return nil
}
