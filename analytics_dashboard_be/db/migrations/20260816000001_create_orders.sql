-- migrate:up
CREATE TABLE IF NOT EXISTS orders (
    id                 BIGSERIAL PRIMARY KEY,
    client_id          TEXT        NOT NULL,
    order_id           TEXT        NOT NULL UNIQUE,
    order_date         DATE        NOT NULL,
    delivery_date      DATE,
    carrier            TEXT        NOT NULL,
    origin_city        TEXT        NOT NULL,
    destination_city   TEXT        NOT NULL,
    status             TEXT        NOT NULL,
    sku                TEXT        NOT NULL,
    product_category   TEXT        NOT NULL,
    quantity           INTEGER     NOT NULL DEFAULT 0,
    unit_price_usd     NUMERIC(12,2) NOT NULL DEFAULT 0,
    order_value_usd    NUMERIC(12,2) NOT NULL DEFAULT 0,
    is_promo           BOOLEAN     NOT NULL DEFAULT FALSE,
    promo_discount_pct NUMERIC(6,2) NOT NULL DEFAULT 0,
    region             TEXT        NOT NULL,
    warehouse          TEXT        NOT NULL,
    transit_days       INTEGER
);

-- Indexes for the common filter/aggregation paths.
CREATE INDEX IF NOT EXISTS idx_orders_order_date ON orders (order_date);
CREATE INDEX IF NOT EXISTS idx_orders_status     ON orders (status);
CREATE INDEX IF NOT EXISTS idx_orders_carrier    ON orders (carrier);
CREATE INDEX IF NOT EXISTS idx_orders_region     ON orders (region);
CREATE INDEX IF NOT EXISTS idx_orders_category   ON orders (product_category);

-- migrate:down
DROP TABLE IF EXISTS orders;
