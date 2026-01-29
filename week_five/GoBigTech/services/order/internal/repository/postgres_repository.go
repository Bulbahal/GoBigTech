package repository

import (
	"context"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bulbahal/GoBigTech/services/order/internal/service"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool: pool,
	}
}

func (r *PostgresRepository) SaveOrder(ctx context.Context, order service.Order) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	orderSQL, orderArgs, err := psql.Insert("orders").Columns("id", "user_id", "status").Values(order.ID, order.UserID, order.Status).ToSql()
	if err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if _, err := tx.Exec(ctx, orderSQL, orderArgs...); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	for _, item := range order.Items {
		itemSQL, itemArgs, err := psql.Insert("order_items").Columns("order_id", "product_id", "quantity").Values(order.ID, item.ProductID, item.Quantity).ToSql()
		if err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if _, err := tx.Exec(ctx, itemSQL, itemArgs...); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (r *PostgresRepository) GetOrderByID(ctx context.Context, id string) (service.Order, error) {
	var order service.Order
	order.ID = id

	row := r.pool.QueryRow(ctx,
		`SELECT user_id, status FROM orders WHERE id = $1`,
		id,
	)
	if err := row.Scan(&order.UserID, &order.Status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.Order{}, errors.New("order not found")
		}
		return service.Order{}, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT product_id, quantity FROM order_items WHERE order_id = $1`,
		id,
	)
	if err != nil {
		return service.Order{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var item service.OrderItem
		if err := rows.Scan(&item.ProductID, &item.Quantity); err != nil {
			return service.Order{}, err
		}
		order.Items = append(order.Items, item)
	}

	if err := rows.Err(); err != nil {
		return service.Order{}, err
	}

	return order, nil
}
