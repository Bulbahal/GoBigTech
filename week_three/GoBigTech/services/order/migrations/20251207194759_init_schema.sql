-- +goose Up
CREATE TABLE orders (
                        id TEXT PRIMARY KEY,
                        user_id TEXT NOT NULL,
                        status TEXT NOT NULL
);

CREATE TABLE order_items (
                             id SERIAL PRIMARY KEY,
                             order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
                             product_id TEXT NOT NULL,
                             quantity INT NOT NULL
);

-- +goose Down
DROP TABLE order_items;
DROP TABLE orders;