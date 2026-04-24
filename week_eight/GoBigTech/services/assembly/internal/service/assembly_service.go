package service

import (
	"context"
	"encoding/json"
	"time"

	platformevents "github.com/bulbahal/GoBigTech/platform/events"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
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

var (
	asmMeter            = otel.GetMeterProvider().Meter("assembly-service")
	assemblyDuration, _ = asmMeter.Float64Histogram("assembly_duration_seconds")
)

func NewAssemblyService(publisher EventPublisher) AssemblyService {
	return &assemblyService{publisher: publisher}
}

func (s *assemblyService) HandlePaymentCompleted(ctx context.Context, event platformevents.OrderPaymentCompleted) error {
	start := time.Now()
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

	if err := s.publisher.Publish(ctx, platformevents.TopicOrderAssemblyCompleted, event.OrderID, payload); err != nil {
		return err
	}
	
	assemblyDuration.Record(ctx, time.Since(start).Seconds())

	return nil
}
