package service

import (
	"context"
	"errors"
	"testing"
)

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

func TestCreateOrder_Success(t *testing.T) {
	ctx := context.Background()

	invMock := &mockInventoryClient{}
	payMock := &mockPaymentClient{}

	svc := NewOrderService(invMock, payMock)

	items := []OrderItem{
		{ProductID: "p1", Quantity: 2},
	}

	order, err := svc.CreateOrder(ctx, "u1", items)
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	if !invMock.called {
		t.Errorf("expected inventory.ReserveStock to be called")
	}
	if invMock.calledProdID != "p1" || invMock.calledQty != 2 {
		t.Errorf("inventory called with wrong args: product=%v qty=%v",
			invMock.calledProdID, invMock.calledQty)
	}

	if !payMock.called {
		t.Errorf("expected payment to be called")
	}
	if order.ID != "order-123" {
		t.Errorf("order ID is wrong: %v", order.ID)
	}
	if order.UserID != "u1" {
		t.Errorf("order UserID is wrong: %v", order.UserID)
	}
	if order.Status != "paid" {
		t.Errorf("order Status is wrong: %v", order.Status)
	}

	if len(order.Items) != 1 {
		t.Errorf("order Items is wrong: %v", order.Items)
	}
	if order.Items[0].ProductID != "p1" || order.Items[0].Quantity != 2 {
		t.Errorf("order Items[0].ProductID is wrong: %v", order.Items[0].ProductID)
	}
}
func TestCreateOrder_InventoryError(t *testing.T) {
	ctx := context.Background()

	invMock := &mockInventoryClient{
		reserveErr: errors.New("no stock"),
	}
	payMock := &mockPaymentClient{}

	svc := NewOrderService(invMock, payMock)

	_, err := svc.CreateOrder(ctx, "u1", []OrderItem{
		{ProductID: "p1", Quantity: 1},
	})

	if err == nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	if !invMock.called {
		t.Errorf("expected inventory.ReserveStock to be called")
	}
	if payMock.called {
		t.Errorf("expected payment to be called")
	}
}
