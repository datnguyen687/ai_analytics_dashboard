package domain

import "context"

// OrderRepository is the data-access port. The Postgres/sqlx adapter implements
// it; services depend only on this interface, which keeps them unit-testable
// with a fake repository.
type OrderRepository interface {
	Meta(ctx context.Context) (Meta, error)
	KPIs(ctx context.Context, f Filters) (KPIs, error)
	TimeSeries(ctx context.Context, f Filters, granularity string) ([]TimePoint, error)
	StatusMix(ctx context.Context, f Filters) ([]StatusCount, error)
	Breakdown(ctx context.Context, f Filters, dimension string, limit int) ([]BreakdownRow, error)
	CategoryStack(ctx context.Context, f Filters, topN int) (CategoryStack, error)
	Orders(ctx context.Context, q OrderQuery) (OrderPage, error)
	// MonthlyUnits returns summed quantity per month, oldest first — the input
	// to the forecasting tool. Category "" means all categories.
	MonthlyUnits(ctx context.Context, category string) ([]MonthUnits, error)
	// ImportOrders writes orders per ImportOptions (truncate + conflict handling).
	// Returns rows imported (inserted/updated) and skipped (ignored duplicates).
	ImportOrders(ctx context.Context, orders []Order, opts ImportOptions) (imported, skipped int, err error)

	// --- single-order CRUD ---
	GetOrder(ctx context.Context, orderID string) (Order, bool, error)
	CreateOrder(ctx context.Context, o Order) error                 // ErrConflict if order_id exists
	UpdateOrder(ctx context.Context, o Order) (found bool, err error)
	DeleteOrder(ctx context.Context, orderID string) (found bool, err error)
}

// MonthUnits is one month of summed shipped units for forecasting.
type MonthUnits struct {
	Bucket string `db:"bucket" json:"bucket"`
	Units  int    `db:"units" json:"units"`
}

// UserRepository is the accounts data-access port.
type UserRepository interface {
	ByUsername(ctx context.Context, username string) (User, error)
	// Upsert creates or updates an account by username — the injection path for
	// the `user` CLI command (there is no public sign-up).
	Upsert(ctx context.Context, username, passwordHash string, role Role) error
	// List returns all accounts (admin-only), never including password hashes.
	List(ctx context.Context) ([]User, error)
}

// ErrUserNotFound is returned by ByUsername when no account matches.
var ErrUserNotFound = NewAPIError(401, "AUTH_INVALID_CREDENTIALS", "user not found")

// RateLimiter is the rate-limiting port. Allow atomically counts a hit against
// key within a fixed window and reports whether it is permitted, plus the
// seconds until the window resets when it is not. Implementations fail OPEN (an
// infra error must never lock users out of the whole API).
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit, windowSeconds int) (allowed bool, retryAfter int, err error)
}

// Cache is the caching port (Redis adapter). Get returns (false, nil) on a miss.
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) (bool, error)
	Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error
	// DeleteByPrefix removes all keys starting with prefix (used to invalidate
	// cached read models after a data import).
	DeleteByPrefix(ctx context.Context, prefix string) error
}

// Interpretation is the structured, validated output of the interpretation
// step: which tool to run and with what parameters. The LLM (or the rule-based
// fallback) produces ONLY this — never the numbers. The tools compute from it.
type Interpretation struct {
	Tool      Tool    // analytics.query | forecast.demand
	Intent    string  // canonical handler key (delayed_by_week, breakdown, forecast, …)
	Dimension string  // for breakdowns: carrier | region | product_category | destination_city
	Category  string  // for forecasts / category filters
	Horizon   int     // forecast horizon in months
	Window    string  // human label for the time window ("last 3 months")
	Filters   Filters // resolved absolute filters (dates + dimension lists)
	Source    string  // "gemini" | "rules" — for explainability
}

// Interpreter turns a natural-language question into a validated Interpretation.
// The default is rule-based; an LLM-backed implementation is swapped in behind
// this same interface without touching the router or the tools.
type Interpreter interface {
	Interpret(ctx context.Context, question string) (Interpretation, error)
}
