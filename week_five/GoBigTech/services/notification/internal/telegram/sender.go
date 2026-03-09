package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	// Базовый URL Telegram Bot API (без токена).
	apiBase = "https://api.telegram.org"
	// Таймаут запроса, чтобы не зависнуть при проблемах с сетью.
	requestTimeout = 10 * time.Second
)

// Sender отправляет текстовые сообщения в Telegram через Bot API.
type Sender interface {
	SendMessage(ctx context.Context, text string) error
}

// sender — реализация Sender через HTTP-вызовы sendMessage.
type sender struct {
	client    *http.Client
	botToken  string
	chatID    string
	requestURL string
}

// NewSender создаёт отправителя сообщений в Telegram.
// botToken и chatID берутся из конфига (env TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID).
func NewSender(botToken, chatID string) Sender {
	return &sender{
		client:    &http.Client{Timeout: requestTimeout},
		botToken:  botToken,
		chatID:    chatID,
		requestURL: apiBase + "/bot" + botToken + "/sendMessage",
	}
}

// sendMessageRequest — тело запроса к Telegram API (только нужные поля).
type sendMessageRequest struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

// SendMessage отправляет текст в чат, заданный при создании Sender.
// Уважает context: при отмене контекста запрос не выполняется или прерывается.
func (s *sender) SendMessage(ctx context.Context, text string) error {
	if s.botToken == "" || s.chatID == "" {
		return fmt.Errorf("telegram: bot token or chat id not set")
	}

	body := sendMessageRequest{ChatID: s.chatID, Text: text}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("telegram: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.requestURL, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("telegram: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram: unexpected status %d", resp.StatusCode)
	}
	return nil
}
