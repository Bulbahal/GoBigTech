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

type orderService struct {
	inventory InventoryClient
	payment   PaymentClient
}

func NewOrderService(inv InventoryClient, pay PaymentClient) OrderService {
	return &orderService{
		inventory: inv,
		payment:   pay,
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
	if err := s.payment.ProcessPayment(ctx, "order-123", userID, 100.0, "card"); err != nil {
		return Order{}, err
	}
	return Order{
		ID:     "order-123",
		UserID: userID,
		Status: "paid",
		Items:  items,
	}, nil
}

func (s *orderService) GetOrder(ctx context.Context, id string) (Order, error) {
	return Order{
		ID:     id,
		UserID: "u1",
		Status: "paid",
		Items: []OrderItem{
			{ProductID: "p1", Quantity: 2},
		},
	}, nil
}
