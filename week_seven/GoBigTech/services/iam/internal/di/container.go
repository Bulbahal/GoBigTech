package di

import (
	"context"
	"net"

	platformlogger "github.com/bulbahal/GoBigTech/platform/logger"
	"github.com/bulbahal/GoBigTech/services/iam/internal/config"
	"github.com/bulbahal/GoBigTech/services/iam/internal/service"
	transportgprc "github.com/bulbahal/GoBigTech/services/iam/internal/transport/grpc"
	iampb "github.com/bulbahal/GoBigTech/services/iam/v1"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
)

// Container — простой DI-контейнер для IAM Service.
// Он лениво создаёт и хранит общие зависимости: логгер, подключения к БД, Redis, gRPC-сервер и обработчик.
type Container struct {
	cfg *config.Config

	log *zap.Logger

	pgPool *pgxpool.Pool
	redis  *redis.Client

	lis    net.Listener
	server *googlegrpc.Server

	handler iampb.IAMServiceServer
}

// New создаёт контейнер с переданной конфигурацией.
func New(cfg config.Config) *Container {
	return &Container{cfg: &cfg}
}

// Logger создаёт или возвращает общий zap-логгер для сервиса.
func (c *Container) Logger() *zap.Logger {
	if c.log == nil {
		c.log = platformlogger.New(c.cfg.Env)
	}
	return c.log
}

// PGPool создаёт пул соединений к PostgreSQL для работы с пользователями.
func (c *Container) PGPool(ctx context.Context) *pgxpool.Pool {
	if c.pgPool == nil {
		pool, err := pgxpool.New(ctx, c.cfg.PostgresDSN)
		if err != nil {
			c.Logger().Fatal("pgxpool new", zap.Error(err))
		}
		c.pgPool = pool
	}
	return c.pgPool
}

// RedisClient создаёт Redis-клиент для работы с сессиями.
func (c *Container) RedisClient() *redis.Client {
	if c.redis == nil {
		c.redis = redis.NewClient(&redis.Options{
			Addr: c.cfg.RedisAddr,
		})
	}
	return c.redis
}

// Listener возвращает сетевой слушатель для gRPC-сервера.
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

// GRPCServer создаёт gRPC-сервер без интерцепторов (их можно добавить позже).
func (c *Container) GRPCServer() *googlegrpc.Server {
	if c.server == nil {
		c.server = googlegrpc.NewServer()
	}
	return c.server
}

// Handler создаёт обработчик IAMService на основе бизнес-сервиса.
// Пока бизнес-логика минимальна, но структура уже совпадает с другими сервисами.
func (c *Container) Handler(ctx context.Context) iampb.IAMServiceServer {
	if c.handler == nil {
		svc := service.NewIAMService(c.PGPool(ctx), c.RedisClient(), c.Logger())
		c.handler = &transportgprc.Server{Service: svc}
	}
	return c.handler
}

// Close аккуратно закрывает все ресурсы: gRPC-сервер, listener, подключения к Redis и Postgres, логгер.
func (c *Container) Close(ctx context.Context) error {
	if c.server != nil {
		c.server.GracefulStop()
	}
	if c.lis != nil {
		_ = c.lis.Close()
	}
	if c.redis != nil {
		_ = c.redis.Close()
	}
	if c.pgPool != nil {
		c.pgPool.Close()
	}
	if c.log != nil {
		_ = c.log.Sync()
	}
	return nil
}
