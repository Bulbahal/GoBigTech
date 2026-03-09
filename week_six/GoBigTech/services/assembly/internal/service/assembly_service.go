package service

import (
	"context"
	"encoding/json"
	"time"

	platformevents "github.com/bulbahal/GoBigTech/platform/events"
	"github.com/google/uuid"
)

// EventPublisher публикует события в Kafka.
type EventPublisher interface {
	Publish(ctx context.Context, topic, key string, value []byte) error
}

// AssemblyService обрабатывает событие оплаты: ждёт 10 сек, публикует событие сборки.
type AssemblyService interface {
	HandlePaymentCompleted(ctx context.Context, event platformevents.OrderPaymentCompleted) error
}

type assemblyService struct {
	publisher EventPublisher
}

func NewAssemblyService(publisher EventPublisher) AssemblyService {
	return &assemblyService{publisher: publisher}
}

func (s *assemblyService) HandlePaymentCompleted(ctx context.Context, event platformevents.OrderPaymentCompleted) error {
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}

	completed := platformevents.OrderAssemblyCompleted{
		EventID:    uuid.New().String(),
		OccurredAt: time.Now().UTC(),
		OrderID:    event.OrderID,
		UserID:     event.UserID,
	}

	payload, err := json.Marshal(completed)
	if err != nil {
		return err
	}

	return s.publisher.Publish(ctx, platformevents.TopicOrderAssemblyCompleted, event.OrderID, payload)
}
