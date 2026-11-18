package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	platformevents "github.com/bulbahal/GoBigTech/platform/events"
)

type mockRepo struct {
	saveErr   error
	getErr    error
	updateErr error
	getOrder  Order

	saveCalled   bool
	savedOrder   Order
	updatedID    string
	updatedState string
}

func (m *mockRepo) SaveOrder(ctx context.Context, order Order) error {
	m.savedOrder = order
	m.saveCalled = true
	return m.saveErr
}

func (m *mockRepo) GetOrderByID(ctx context.Context, id string) (Order, error) {
	if m.getErr != nil {
		return Order{}, m.getErr
	}
	return m.getOrder, nil
}

func (m *mockRepo) UpdateOrderStatus(ctx context.Context, orderID, status string) error {
	m.updatedID = orderID
	m.updatedState = status
	return m.updateErr
}

type mockInventoryClient struct {
	reserveErr error

	called       bool
	calledProdID string
	calledQty    int32
}

func (m *mockInventoryClient) ReserveStock(ctx context.Context, productID string, qty int32) error {
	m.called = true
	m.calledProdID = productID
	m.calledQty = qty
	return m.reserveErr
}

type mockPaymentClient struct {
	payErr error

	called       bool
	calledOrder  string
	calledUser   string
	calledAmt    float64
	calledMethod string
}

func (m *mockPaymentClient) ProcessPayment(ctx context.Context, orderID, userID string, amount float64, method string) error {
	m.called = true
	m.calledOrder = orderID
	m.calledUser = userID
	m.calledAmt = amount
	m.calledMethod = method
	return m.payErr
}

type mockEventProducer struct {
	publishErr error

	called      bool
	calledTopic string
	calledKey   string
	calledValue []byte
}

func (m *mockEventProducer) Publish(ctx context.Context, topic, key string, value []byte) error {
	m.called = true
	m.calledTopic = topic
	m.calledKey = key
	m.calledValue = value
	return m.publishErr
}

func TestOrderService_CreateOrder(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		items          []OrderItem
		invErr         error
		payErr         error
		saveErr        error
		publishErr     error
		wantErr        bool
		wantInvCalled  bool
		wantPayCalled  bool
		wantSaveCalled bool
		wantPubCalled  bool
	}{
		{
			name:           "success",
			userID:         "u1",
			items:          []OrderItem{{ProductID: "p1", Quantity: 2}},
			wantErr:        false,
			wantInvCalled:  true,
			wantPayCalled:  true,
			wantSaveCalled: true,
			wantPubCalled:  true,
		},
		{
			name:    "empty userID",
			userID:  "",
			items:   []OrderItem{{ProductID: "p1", Quantity: 1}},
			wantErr: true,
		},
		{
			name:    "empty items",
			userID:  "u1",
			items:   nil,
			wantErr: true,
		},
		{
			name:          "inventory error",
			userID:        "u1",
			items:         []OrderItem{{ProductID: "p1", Quantity: 1}},
			invErr:        errors.New("no stock"),
			wantErr:       true,
			wantInvCalled: true,
		},
		{
			name:          "payment error",
			userID:        "u1",
			items:         []OrderItem{{ProductID: "p1", Quantity: 1}},
			payErr:        errors.New("payment declined"),
			wantErr:       true,
			wantInvCalled: true,
			wantPayCalled: true,
		},
		{
			name:           "repository save error",
			userID:         "u1",
			items:          []OrderItem{{ProductID: "p1", Quantity: 1}},
			saveErr:        errors.New("db unavailable"),
			wantErr:        true,
			wantInvCalled:  true,
			wantPayCalled:  true,
			wantSaveCalled: true,
		},
		{
			name:           "event publish error",
			userID:         "u1",
			items:          []OrderItem{{ProductID: "p1", Quantity: 1}},
			publishErr:     errors.New("kafka broker down"),
			wantErr:        true,
			wantInvCalled:  true,
			wantPayCalled:  true,
			wantSaveCalled: true,
			wantPubCalled:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invMock := &mockInventoryClient{reserveErr: tt.invErr}
			payMock := &mockPaymentClient{payErr: tt.payErr}
			repoMock := &mockRepo{saveErr: tt.saveErr}
			pubMock := &mockEventProducer{publishErr: tt.publishErr}

			svc := NewOrderService(invMock, payMock, repoMock, pubMock)

			order, err := svc.CreateOrder(context.Background(), tt.userID, tt.items)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CreateOrder() error = %v, wantErr %v", err, tt.wantErr)
			}

			if invMock.called != tt.wantInvCalled {
				t.Errorf("inventory called = %v, want %v", invMock.called, tt.wantInvCalled)
			}
			if payMock.called != tt.wantPayCalled {
				t.Errorf("payment called = %v, want %v", payMock.called, tt.wantPayCalled)
			}
			if repoMock.saveCalled != tt.wantSaveCalled {
				t.Errorf("repo.SaveOrder called = %v, want %v", repoMock.saveCalled, tt.wantSaveCalled)
			}
			if pubMock.called != tt.wantPubCalled {
				t.Errorf("events.Publish called = %v, want %v", pubMock.called, tt.wantPubCalled)
			}

			if !tt.wantErr {
				if order.ID == "" {
					t.Errorf("expected generated non-empty order ID")
				}
				if order.UserID != tt.userID || order.Status != "paid" {
					t.Errorf("unexpected order state: %+v", order)
				}
				if pubMock.calledTopic != platformevents.TopicOrderPaymentCompleted {
					t.Errorf("unexpected topic: got %s, want %s", pubMock.calledTopic, platformevents.TopicOrderPaymentCompleted)
				}

				var published platformevents.OrderPaymentCompleted
				if err := json.Unmarshal(pubMock.calledValue, &published); err != nil {
					t.Fatalf("failed to unmarshal published event: %v", err)
				}
				if published.OrderID != order.ID || published.UserID != tt.userID {
					t.Errorf("published event mismatch: %+v", published)
				}
			}
		})
	}
}

func TestOrderService_HandleAssemblyCompleted(t *testing.T) {
	repoMock := &mockRepo{}
	svc := NewOrderService(&mockInventoryClient{}, &mockPaymentClient{}, repoMock, &mockEventProducer{})

	err := svc.HandleAssemblyCompleted(context.Background(), platformevents.OrderAssemblyCompleted{
		OrderID: "order-123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repoMock.updatedID != "order-123" || repoMock.updatedState != "assembled" {
		t.Errorf("unexpected status update: id=%s, status=%s", repoMock.updatedID, repoMock.updatedState)
	}
}
