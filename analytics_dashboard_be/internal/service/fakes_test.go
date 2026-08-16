package service

import (
	"context"
	"encoding/json"
	"sync"

	"analytics-dashboard-be/internal/domain"
)

// --- fake OrderRepository with canned, deterministic data ---

type fakeOrderRepo struct {
	ordersCalls int
}

func (f *fakeOrderRepo) Meta(context.Context) (domain.Meta, error) {
	return domain.Meta{
		Carriers:   []string{"DHL", "FedEx", "UPS"},
		Regions:    []string{"EU", "UK", "US-E"},
		Categories: []string{"CRAYON", "PAPER", "STICKER"},
		Statuses:   []string{"delivered", "delayed"},
		DateMin:    "2025-01-01",
		DateMax:    "2025-12-30",
	}, nil
}

func (f *fakeOrderRepo) KPIs(context.Context, domain.Filters) (domain.KPIs, error) {
	return domain.KPIs{
		TotalOrders: 100, DeliveredOrders: 80, DelayedOrders: 10,
		OnTimeRate: 0.8888, AvgDeliveryDays: 3.2, TotalRevenue: 12345,
	}, nil
}

func (f *fakeOrderRepo) TimeSeries(_ context.Context, _ domain.Filters, granularity string) ([]domain.TimePoint, error) {
	b := "2025-01"
	if granularity == "week" {
		b = "2025-01-06"
	}
	return []domain.TimePoint{
		{Bucket: b, Orders: 50, Delivered: 40, Delayed: 5, Revenue: 6000},
		{Bucket: "2025-02", Orders: 50, Delivered: 40, Delayed: 5, Revenue: 6345},
	}, nil
}

func (f *fakeOrderRepo) StatusMix(context.Context, domain.Filters) ([]domain.StatusCount, error) {
	return []domain.StatusCount{{Status: "delivered", Count: 80}, {Status: "delayed", Count: 10}}, nil
}

func (f *fakeOrderRepo) Breakdown(_ context.Context, _ domain.Filters, dimension string, _ int) ([]domain.BreakdownRow, error) {
	return []domain.BreakdownRow{
		{Name: "DHL", Orders: 60, Delivered: 50, Delayed: 8, DelayRate: 0.13, AvgDeliveryDays: 2.5, Revenue: 8000},
		{Name: "UPS", Orders: 40, Delivered: 30, Delayed: 2, DelayRate: 0.0625, AvgDeliveryDays: 3.5, Revenue: 4345},
	}, nil
}

func (f *fakeOrderRepo) CategoryStack(context.Context, domain.Filters, int) (domain.CategoryStack, error) {
	return domain.CategoryStack{
		Keys: []string{"CRAYON", "Other"},
		Data: []map[string]interface{}{{"bucket": "2025-01", "CRAYON": 5, "Other": 3}},
	}, nil
}

func (f *fakeOrderRepo) Orders(context.Context, domain.OrderQuery) (domain.OrderPage, error) {
	f.ordersCalls++
	return domain.OrderPage{
		Rows:     []domain.Order{{OrderID: "ORD-1", Carrier: "DHL"}},
		Total:    1, Page: 0, PageSize: 15,
	}, nil
}

func (f *fakeOrderRepo) MonthlyUnits(context.Context, string) ([]domain.MonthUnits, error) {
	return []domain.MonthUnits{
		{Bucket: "2025-01", Units: 30}, {Bucket: "2025-02", Units: 24},
		{Bucket: "2025-03", Units: 20}, {Bucket: "2025-04", Units: 16},
	}, nil
}

// --- fake in-memory Cache ---

type fakeCache struct {
	mu   sync.Mutex
	data map[string][]byte
	sets int
}

func newFakeCache() *fakeCache { return &fakeCache{data: map[string][]byte{}} }

func (c *fakeCache) Get(_ context.Context, key string, dest interface{}) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b, ok := c.data[key]
	if !ok {
		return false, nil
	}
	return true, json.Unmarshal(b, dest)
}

func (c *fakeCache) Set(_ context.Context, key string, value interface{}, _ int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	b, _ := json.Marshal(value)
	c.data[key] = b
	c.sets++
	return nil
}

// --- fake UserRepository ---

type fakeUserRepo struct {
	users map[string]domain.User
}

func (r *fakeUserRepo) ByUsername(_ context.Context, username string) (domain.User, error) {
	u, ok := r.users[username]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *fakeUserRepo) Upsert(_ context.Context, username, hash string, role domain.Role) error {
	if r.users == nil {
		r.users = map[string]domain.User{}
	}
	r.users[username] = domain.User{Username: username, PasswordHash: hash, Role: role}
	return nil
}

func (r *fakeUserRepo) List(context.Context) ([]domain.User, error) {
	out := make([]domain.User, 0, len(r.users))
	for _, u := range r.users {
		u.PasswordHash = ""
		out = append(out, u)
	}
	return out, nil
}
