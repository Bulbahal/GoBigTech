package di

import (
	"context"
	"net"

	platformlogger "github.com/bulbahal/GoBigTech/platform/logger"
	iampb "github.com/bulbahal/GoBigTech/services/iam/v1"
	"github.com/bulbahal/GoBigTech/services/inventory/internal/config"
	"github.com/bulbahal/GoBigTech/services/inventory/internal/repository"
	transportgrpc "github.com/bulbahal/GoBigTech/services/inventory/internal/transport/grpc"
	inventorypb "github.com/bulbahal/GoBigTech/services/inventory/v1"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Container struct {
	cfg *config.Config

	log *zap.Logger

	mongo *mongo.Client
	repo  *repository.MongoInventoryRepository

	lis    net.Listener
	server *grpc.Server

	handler inventorypb.InventoryServiceServer

	iamConn   *grpc.ClientConn
	iamClient iampb.IAMServiceClient
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
		c.server = grpc.NewServer(
			grpc.UnaryInterceptor(c.authInterceptor()),
		)
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
	if c.iamConn != nil {
		_ = c.iamConn.Close()
	}
	if c.mongo != nil {
		_ = c.mongo.Disconnect(ctx)
	}
	if c.log != nil {
		_ = c.log.Sync()
	}
	return nil
}

func (c *Container) IAMClient() iampb.IAMServiceClient {
	if c.iamClient == nil {
		conn, err := grpc.Dial(
			c.cfg.IAMAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			c.Logger().Fatal("dial iam", zap.Error(err))
		}
		c.iamConn = conn
		c.iamClient = iampb.NewIAMServiceClient(conn)
	}
	return c.iamClient
}

const sessionMetadataKey = "x-session-id"

func (c *Container) authInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		values := md.Get(sessionMetadataKey)
		if len(values) == 0 || values[0] == "" {
			return nil, status.Error(codes.Unauthenticated, "missing session id")
		}
		sessionID := values[0]

		resp, err := c.IAMClient().ValidateSession(ctx, &iampb.ValidateSessionRequest{
			SessionId: sessionID,
		})
		if err != nil {
			return nil, err
		}
		if !resp.GetValid() {
			return nil, status.Error(codes.Unauthenticated, "invalid session")
		}
		
		return handler(ctx, req)
	}
}
