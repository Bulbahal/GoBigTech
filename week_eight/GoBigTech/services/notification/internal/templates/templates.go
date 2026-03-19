package templates

import (
	"bytes"
	"fmt"
	"text/template"

	platformevents "github.com/bulbahal/GoBigTech/platform/events"
)

// Шаблоны текста уведомлений (text/template). Поля берутся из событий platform/events.
var (
	// Сообщение при оплате заказа: подставляем order_id, user_id, время.
	paymentCompletedTmpl = template.Must(template.New("payment").Parse(
		`✅ Оплата получена
Заказ: {{.OrderID}}
Пользователь: {{.UserID}}
Время: {{.OccurredAt.Format "02.01.2006 15:04:05"}}`,
	))
	// Сообщение при сборке заказа.
	assemblyCompletedTmpl = template.Must(template.New("assembly").Parse(
		`📦 Сборка завершена
Заказ: {{.OrderID}}
Пользователь: {{.UserID}}
Время: {{.OccurredAt.Format "02.01.2006 15:04:05"}}`,
	))
)

// RenderPaymentCompleted формирует текст уведомления по событию оплаты.
// Использует шаблон paymentCompletedTmpl и структуру OrderPaymentCompleted.
func RenderPaymentCompleted(event platformevents.OrderPaymentCompleted) (string, error) {
	var buf bytes.Buffer
	if err := paymentCompletedTmpl.Execute(&buf, event); err != nil {
		return "", fmt.Errorf("templates: payment: %w", err)
	}
	return buf.String(), nil
}

// RenderAssemblyCompleted формирует текст уведомления по событию сборки.
func RenderAssemblyCompleted(event platformevents.OrderAssemblyCompleted) (string, error) {
	var buf bytes.Buffer
	if err := assemblyCompletedTmpl.Execute(&buf, event); err != nil {
		return "", fmt.Errorf("templates: assembly: %w", err)
	}
	return buf.String(), nil
}
