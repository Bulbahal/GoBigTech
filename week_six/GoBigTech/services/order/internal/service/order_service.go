package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	platformevents "github.com/bulbahal/GoBigTech/platform/events"
	"github.com/google/uuid"
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
	HandleAssemblyCompleted(ctx context.Context, event platformevents.OrderAssemblyCompleted) error
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
	UpdateOrderStatus(ctx context.Context, orderID, status string) error
}

type EventProducer interface {
	Publish(ctx context.Context, topic, key string, value []byte) error
}

type orderService struct {
	inventory InventoryClient
	payment   PaymentClient
	repo      OrderRepository

	events EventProducer
}

func NewOrderService(inv InventoryClient, pay PaymentClient, repo OrderRepository, events EventProducer) *orderService {
	return &orderService{
		inventory: inv,
		payment:   pay,
		repo:      repo,
		events:    events,
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
		ID:     uuid.New().String(),
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

	event := platformevents.OrderPaymentCompleted{
		EventID: uuid.New().String(),
		OccurredAt: time.Now().UTC(),
		OrderID: order.ID,
		UserID: order.UserID,
	  }

	payload, err := json.Marshal(event)
	if err != nil {
		return Order{}, err
	}

	if err := s.events.Publish(ctx, platformevents.TopicOrderPaymentCompleted, order.ID, payload); err != nil {
		return Order{}, err
	}

	return order, nil
}

func (s *orderService) GetOrder(ctx context.Context, id string) (Order, error) {
	return s.repo.GetOrderByID(ctx, id)
}

func (s *orderService) HandleAssemblyCompleted(ctx context.Context, event platformevents.OrderAssemblyCompleted) error {
	return s.repo.UpdateOrderStatus(ctx, event.OrderID, "assembled")
}
