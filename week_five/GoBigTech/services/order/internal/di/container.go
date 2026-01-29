package di

import (
	"context"

	platformlogger "github.com/bulbahal/GoBigTech/platform/logger"
	"github.com/bulbahal/GoBigTech/services/order/internal/config"
	"github.com/bulbahal/GoBigTech/services/order/internal/repository"
	"github.com/bulbahal/GoBigTech/services/order/internal/service"

	inventorypb "github.com/bulbahal/GoBigTech/services/inventory/v1"
	paymentpb "github.com/bulbahal/GoBigTech/services/payment/v1"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Container struct {
	cfg *config.Config

	log  *zap.Logger
	pool *pgxpool.Pool

	invConn *grpc.ClientConn
	payConn *grpc.ClientConn

	inv service.InventoryClient
	pay service.PaymentClient

	repo service.OrderRepository
	svc  service.OrderService
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

func (c *Container) PGPool(ctx context.Context) *pgxpool.Pool {
	if c.pool == nil {
		pool, err := pgxpool.New(ctx, c.cfg.PostgresDSN)
		if err != nil {
			c.Logger().Fatal("pgxpool new", zap.Error(err))
		}
		c.pool = pool
	}
	return c.pool
}

func (c *Container) inventoryConn() *grpc.ClientConn {
	if c.invConn == nil {
		conn, err := grpc.Dial(
			c.cfg.InventoryAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			c.Logger().Fatal("connect inventory", zap.Error(err))
		}
		c.invConn = conn
	}
	return c.invConn
}

func (c *Container) paymentConn() *grpc.ClientConn {
	if c.payConn == nil {
		conn, err := grpc.Dial(
			c.cfg.PaymentAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			c.Logger().Fatal("connect payment", zap.Error(err))
		}
		c.payConn = conn
	}
	return c.payConn
}

type inventoryClientAdapter struct {
	client inventorypb.InventoryServiceClient
}

func (i *inventoryClientAdapter) ReserveStock(ctx context.Context, productID string, qty int32) error {
	_, err := i.client.ReserveStock(ctx, &inventorypb.ReserveStockRequest{
		ProductId: productID,
		Quantity:  qty,
	})
	return err
}

type paymentClientAdapter struct {
	client paymentpb.PaymentServiceClient
}

func (p *paymentClientAdapter) ProcessPayment(ctx context.Context, orderID, userID string, amount float64, method string) error {
	_, err := p.client.ProcessPayment(ctx, &paymentpb.ProcessPaymentRequest{
		OrderId: orderID,
		UserId:  userID,
		Amount:  amount,
		Method:  method,
	})
	return err
}

func (c *Container) InventoryClient() service.InventoryClient {
	if c.inv == nil {
		pbClient := inventorypb.NewInventoryServiceClient(c.inventoryConn())
		c.inv = &inventoryClientAdapter{client: pbClient}
	}
	return c.inv
}

func (c *Container) PaymentClient() service.PaymentClient {
	if c.pay == nil {
		pbClient := paymentpb.NewPaymentServiceClient(c.paymentConn())
		c.pay = &paymentClientAdapter{client: pbClient}
	}
	return c.pay
}

func (c *Container) OrderRepository(ctx context.Context) service.OrderRepository {
	if c.repo == nil {
		c.repo = repository.NewPostgresRepository(c.PGPool(ctx))
	}
	return c.repo
}

func (c *Container) OrderService(ctx context.Context) service.OrderService {
	if c.svc == nil {
		c.svc = service.NewOrderService(
			c.InventoryClient(),
			c.PaymentClient(),
			c.OrderRepository(ctx),
		)
	}
	return c.svc
}

func (c *Container) Close(ctx context.Context) error {
	if c.invConn != nil {
		_ = c.invConn.Close()
	}
	if c.payConn != nil {
		_ = c.payConn.Close()
	}
	if c.pool != nil {
		c.pool.Close()
	}
	if c.log != nil {
		_ = c.log.Sync()
	}
	return nil
}
