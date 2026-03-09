package kafka

import (
	"context"
	"errors"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Message — реэкспорт сообщения Kafka для обработчиков (чтобы не тянуть segmentio в сервисы).
type Message = kafka.Message

type Producer struct {
	writer *kafka.Writer
	log    *zap.Logger
}

func NewProducer(brokers []string, logger *zap.Logger) *Producer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}

	return &Producer{
		writer: w,
		log:    logger,
	}
}

func (p *Producer) Publish(ctx context.Context, topic, key string, value []byte) error {
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
		Time:  time.Now().UTC(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		p.log.Error("kafka publish error", zap.String("topic", topic), zap.String("key", key), zap.Error(err))
		return err
	}

	return nil
}

func (p *Producer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

type MessageHandler func(ctx context.Context, msg kafka.Message) error

type Consumer struct {
	reader *kafka.Reader
	log    *zap.Logger
}

func NewConsumer(brokers []string, groupID string, topics []string, logger *zap.Logger) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     groupID,
		GroupTopics: topics,
	})

	return &Consumer{
		reader: r,
		log:    logger,
	}
}

func (c *Consumer) Run(ctx context.Context, handler MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}

			c.log.Error("kafka read message error", zap.Error(err))
			continue
		}

		handleCtx, cancel := context.WithCancel(ctx)
		err = handler(handleCtx, msg)
		cancel()
		if err != nil {
			c.log.Error("kafka handler error", zap.Error(err))
			continue
		}
	}
}

func (c *Consumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}
	return c.reader.Close()
}

