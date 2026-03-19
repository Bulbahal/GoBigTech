package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// SessionTTL определяет время жизни сессии.
const SessionTTL = 24 * time.Hour

// SessionRepository описывает операции с сессиями в Redis.
type SessionRepository interface {
	CreateSession(ctx context.Context, sessionID, userID string) error
	GetUserIDBySession(ctx context.Context, sessionID string) (string, error)
	DeleteSession(ctx context.Context, sessionID string) error
}

// RedisSessionRepository — реализация SessionRepository поверх Redis.
type RedisSessionRepository struct {
	client *redis.Client
}

// NewRedisSessionRepository создаёт репозиторий сессий на основе Redis-клиента.
func NewRedisSessionRepository(client *redis.Client) *RedisSessionRepository {
	return &RedisSessionRepository{client: client}
}

// sessionKey формирует ключ сессии в Redis.
func sessionKey(sessionID string) string {
	return fmt.Sprintf("session:%s", sessionID)
}

// CreateSession создаёт новую сессию:
// записывает user_id и created_at в HASH и выставляет TTL.
func (r *RedisSessionRepository) CreateSession(ctx context.Context, sessionID, userID string) error {
	key := sessionKey(sessionID)

	now := time.Now().UTC().Format(time.RFC3339Nano)

	if err := r.client.HSet(ctx, key, map[string]string{
		"user_id":    userID,
		"created_at": now,
	}).Err(); err != nil {
		return err
	}

	if err := r.client.Expire(ctx, key, SessionTTL).Err(); err != nil {
		return err
	}

	return nil
}

// GetUserIDBySession возвращает user_id по session_id.
// Если сессия отсутствует (или истекла), вернётся redis.Nil.
func (r *RedisSessionRepository) GetUserIDBySession(ctx context.Context, sessionID string) (string, error) {
	key := sessionKey(sessionID)

	userID, err := r.client.HGet(ctx, key, "user_id").Result()
	if err != nil {
		return "", err
	}

	return userID, nil
}

// DeleteSession удаляет сессию из Redis.
func (r *RedisSessionRepository) DeleteSession(ctx context.Context, sessionID string) error {
	key := sessionKey(sessionID)
	return r.client.Del(ctx, key).Err()
}

