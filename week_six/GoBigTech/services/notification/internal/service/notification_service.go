package service

import (
	"context"

	iampb "github.com/bulbahal/GoBigTech/services/iam/v1"
	platformevents "github.com/bulbahal/GoBigTech/platform/events"
	"github.com/bulbahal/GoBigTech/services/notification/internal/telegram"
	"github.com/bulbahal/GoBigTech/services/notification/internal/templates"
	"go.uber.org/zap"
)

// NotificationService обрабатывает события оплаты и сборки: рендерит текст по шаблону и отправляет в Telegram.
type NotificationService interface {
	HandlePaymentCompleted(ctx context.Context, event platformevents.OrderPaymentCompleted) error
	HandleAssemblyCompleted(ctx context.Context, event platformevents.OrderAssemblyCompleted) error
}

type notificationService struct {
	sender telegram.Sender
	iam    iampb.IAMServiceClient
	log    *zap.Logger
}

// NewNotificationService создаёт сервис уведомлений с заданным отправителем (Telegram).
func NewNotificationService(sender telegram.Sender, iam iampb.IAMServiceClient, log *zap.Logger) NotificationService {
	return &notificationService{
		sender: sender,
		iam:    iam,
		log:    log,
	}
}

// HandlePaymentCompleted: по событию оплаты формируем сообщение из шаблона и отправляем в Telegram.
func (s *notificationService) HandlePaymentCompleted(ctx context.Context, event platformevents.OrderPaymentCompleted) error {
	contact, err := s.iam.GetUserContact(ctx, &iampb.GetUserContactRequest{
		UserId: event.UserID,
	})
	if err != nil {
		s.log.Warn("get user contact for payment", zap.Error(err), zap.String("user_id", event.UserID))
		return nil
	}
	if contact.GetTelegramId() == "" {
		s.log.Warn("empty telegram id for payment", zap.String("user_id", event.UserID))
		return nil
	}

	text, err := templates.RenderPaymentCompleted(event)
	if err != nil {
		return err
	}
	return s.sender.SendMessage(ctx, contact.GetTelegramId(), text)
}

// HandleAssemblyCompleted: по событию сборки формируем сообщение и отправляем в Telegram.
func (s *notificationService) HandleAssemblyCompleted(ctx context.Context, event platformevents.OrderAssemblyCompleted) error {
	contact, err := s.iam.GetUserContact(ctx, &iampb.GetUserContactRequest{
		UserId: event.UserID,
	})
	if err != nil {
		s.log.Warn("get user contact for assembly", zap.Error(err), zap.String("user_id", event.UserID))
		return nil
	}
	if contact.GetTelegramId() == "" {
		s.log.Warn("empty telegram id for assembly", zap.String("user_id", event.UserID))
		return nil
	}

	text, err := templates.RenderAssemblyCompleted(event)
	if err != nil {
		return err
	}
	return s.sender.SendMessage(ctx, contact.GetTelegramId(), text)
}
