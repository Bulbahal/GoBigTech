package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User представляет пользователя IAM, как он хранится в таблице users.
type User struct {
	ID           string
	Login        string
	PasswordHash string
	TelegramID   *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ErrUserNotFound возвращается, когда пользователь не найден в БД.
var ErrUserNotFound = errors.New("user not found")

// UserRepository описывает операции репозитория пользователей.
type UserRepository interface {
	GetUserByLogin(ctx context.Context, login string) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
	CreateUser(ctx context.Context, user User) error
}

// PostgresUserRepository — реализация UserRepository на PostgreSQL через pgxpool.
type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepository создаёт репозиторий пользователей поверх пула соединений.
func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (r *PostgresUserRepository) GetUserByLogin(ctx context.Context, login string) (User, error) {
	const query = `
SELECT id, login, password_hash, telegram_id, created_at, updated_at
FROM users
WHERE login = $1
`

	row := r.pool.QueryRow(ctx, query, login)

	var u User
	if err := row.Scan(&u.ID, &u.Login, &u.PasswordHash, &u.TelegramID, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}

	return u, nil
}

func (r *PostgresUserRepository) GetUserByID(ctx context.Context, id string) (User, error) {
	const query = `
SELECT id, login, password_hash, telegram_id, created_at, updated_at
FROM users
WHERE id = $1
`

	row := r.pool.QueryRow(ctx, query, id)

	var u User
	if err := row.Scan(&u.ID, &u.Login, &u.PasswordHash, &u.TelegramID, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}

	return u, nil
}

func (r *PostgresUserRepository) CreateUser(ctx context.Context, user User) error {
	const query = `
INSERT INTO users (id, login, password_hash, telegram_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
`

	// created_at/updated_at можно передавать снаружи или оставить нулевыми,
	// тогда сработают DEFAULT значения в таблице. Здесь явно прокидываем поля,
	// чтобы репозиторий был предсказуем.
	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Login,
		user.PasswordHash,
		user.TelegramID,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

