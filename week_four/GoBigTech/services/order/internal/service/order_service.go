package service

import (
	"context"
	"errors"
)

type Order struct {
	ID     string
	UserID string
	Status string
	Items  []OrderItem
}
type OrderItem struct {
	ProductID string
	Quantity  int
}

type OrderService interface {
	CreateOrder(ctx context.Context, userID string, items []OrderItem) (Order, error)
	GetOrder(ctx context.Context, id string) (Order, error)
}

type InventoryClient interface {
	ReserveStock(ctx context.Context, productID string, qty int32) error
}

type PaymentClient interface {
	ProcessPayment(ctx context.Context, orderID, userID string, amount float64, method string) error
}

type OrderRepository interface {
	SaveOrder(ctx context.Context, order Order) error
	GetOrderByID(ctx context.Context, id string) (Order, error)
}

type orderService struct {
	inventory InventoryClient
	payment   PaymentClient
	repo      OrderRepository
}

func NewOrderService(inv InventoryClient, pay PaymentClient, repo OrderRepository) *orderService {
	return &orderService{
		inventory: inv,
		payment:   pay,
		repo:      repo,
	}
}

func (s *orderService) CreateOrder(ctx context.Context, userID string, items []OrderItem) (Order, error) {
	if userID == "" {
		return Order{}, errors.New("userID cannot be empty")
	}
	if len(items) == 0 {
		return Order{}, errors.New("items cannot be empty")
	}

	first := items[0]
	if err := s.inventory.ReserveStock(ctx, first.ProductID, int32(first.Quantity)); err != nil {
		return Order{}, err
	}

	order := Order{
		ID:     "order-124",
		UserID: userID,
		Status: "paid",
		Items:  items,
	}

	if err := s.payment.ProcessPayment(ctx, order.ID, userID, 100.0, "card"); err != nil {
		return Order{}, err
	}

	if err := s.repo.SaveOrder(ctx, order); err != nil {
		return Order{}, err
	}

	return order, nil
}

func (s *orderService) GetOrder(ctx context.Context, id string) (Order, error) {
	return s.repo.GetOrderByID(ctx, id)
}
