package service

import (
	"context"

	platformevents "github.com/bulbahal/GoBigTech/platform/events"
	"github.com/bulbahal/GoBigTech/services/notification/internal/telegram"
	"github.com/bulbahal/GoBigTech/services/notification/internal/templates"
)

// NotificationService обрабатывает события оплаты и сборки: рендерит текст по шаблону и отправляет в Telegram.
type NotificationService interface {
	HandlePaymentCompleted(ctx context.Context, event platformevents.OrderPaymentCompleted) error
	HandleAssemblyCompleted(ctx context.Context, event platformevents.OrderAssemblyCompleted) error
}

type notificationService struct {
	sender telegram.Sender
}

// NewNotificationService создаёт сервис уведомлений с заданным отправителем (Telegram).
func NewNotificationService(sender telegram.Sender) NotificationService {
	return &notificationService{sender: sender}
}

// HandlePaymentCompleted: по событию оплаты формируем сообщение из шаблона и отправляем в Telegram.
func (s *notificationService) HandlePaymentCompleted(ctx context.Context, event platformevents.OrderPaymentCompleted) error {
	text, err := templates.RenderPaymentCompleted(event)
	if err != nil {
		return err
	}
	return s.sender.SendMessage(ctx, text)
}

// HandleAssemblyCompleted: по событию сборки формируем сообщение и отправляем в Telegram.
func (s *notificationService) HandleAssemblyCompleted(ctx context.Context, event platformevents.OrderAssemblyCompleted) error {
	text, err := templates.RenderAssemblyCompleted(event)
	if err != nil {
		return err
	}
	return s.sender.SendMessage(ctx, text)
}
