package events

import "time"

const (
	TopicOrderPaymentCompleted  = "order.payment.completed.v1"
	TopicOrderAssemblyCompleted = "order.assembly.completed.v1"
)

type OrderPaymentCompleted struct {
	EventID    string    `json:"event_id"`
	OccurredAt time.Time `json:"occurred_at"`

	OrderID string `json:"order_id"`
	UserID  string `json:"user_id"`
}

type OrderAssemblyCompleted struct {
	EventID    string    `json:"event_id"`
	OccurredAt time.Time `json:"occurred_at"`

	OrderID string `json:"order_id"`
	UserID  string `json:"user_id"`
}
