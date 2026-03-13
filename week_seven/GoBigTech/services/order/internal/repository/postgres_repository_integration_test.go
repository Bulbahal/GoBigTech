package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bulbahal/GoBigTech/services/order/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testPool *pgxpool.Pool
	testRepo *PostgresRepository
)

// твой DDL из миграции (без goose-комментов — тесту они не нужны)
const schemaSQL = `
CREATE TABLE IF NOT EXISTS orders (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	status TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS order_items (
	id SERIAL PRIMARY KEY,
	order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
	product_id TEXT NOT NULL,
	quantity INT NOT NULL
);
`

func TestMain(m *testing.M) {
	ctx := context.Background()

	// 1) Поднимаем Postgres контейнер
	pg, err := postgres.Run(ctx,
		"postgres:16",
		postgres.WithDatabase("appdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		tc.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		fmt.Println("failed to start postgres container:", err)
		os.Exit(1)
	}

	// 2) Берём DSN и создаём pool
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Println("failed to get connection string:", err)
		_ = pg.Terminate(ctx)
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Println("failed to create pgx pool:", err)
		_ = pg.Terminate(ctx)
		os.Exit(1)
	}

	// 3) Применяем схему
	if _, err := pool.Exec(ctx, schemaSQL); err != nil {
		fmt.Println("failed to apply schema:", err)
		pool.Close()
		_ = pg.Terminate(ctx)
		os.Exit(1)
	}

	testPool = pool
	testRepo = NewPostgresRepository(testPool)

	// 4) Запускаем тесты
	code := m.Run()

	// 5) TearDownSuite
	testPool.Close()
	_ = pg.Terminate(ctx)

	os.Exit(code)
}

func TestPostgresRepository_SaveAndGetOrder(t *testing.T) {
	ctx := context.Background()

	_, _ = testPool.Exec(ctx, `TRUNCATE TABLE order_items, orders`)

	want := service.Order{
		ID:     "order-1",
		UserID: "user-1",
		Status: "paid",
		Items: []service.OrderItem{
			{ProductID: "p1", Quantity: 2},
			{ProductID: "p2", Quantity: 1},
		},
	}

	if err := testRepo.SaveOrder(ctx, want); err != nil {
		t.Fatalf("SaveOrder error: %v", err)
	}

	got, err := testRepo.GetOrderByID(ctx, want.ID)
	if err != nil {
		t.Fatalf("GetOrderByID error: %v", err)
	}

	if got.ID != want.ID || got.UserID != want.UserID || got.Status != want.Status {
		t.Fatalf("order mismatch: got=%+v want=%+v", got, want)
	}

	if len(got.Items) != len(want.Items) {
		t.Fatalf("items len mismatch: got=%d want=%d", len(got.Items), len(want.Items))
	}

	for i := range want.Items {
		if got.Items[i].ProductID != want.Items[i].ProductID || got.Items[i].Quantity != want.Items[i].Quantity {
			t.Fatalf("item[%d] mismatch: got=%+v want=%+v", i, got.Items[i], want.Items[i])
		}
	}
}
