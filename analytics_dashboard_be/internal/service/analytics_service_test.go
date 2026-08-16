package service

import (
	"context"
	"testing"

	"analytics-dashboard-be/internal/domain"
)

func TestAnalyticsMetaCaches(t *testing.T) {
	cache := newFakeCache()
	svc := NewAnalyticsService(&fakeOrderRepo{}, cache, 60)

	m1, err := svc.Meta(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(m1.Carriers) != 3 {
		t.Fatalf("carriers = %d, want 3", len(m1.Carriers))
	}
	if _, ok := cache.data["meta"]; !ok {
		t.Fatal("meta not cached")
	}
	// Second call served from cache.
	if _, err := svc.Meta(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cache.sets != 1 {
		t.Fatalf("cache sets = %d, want 1 (second call should hit cache)", cache.sets)
	}
}

func TestAnalyticsDashboard(t *testing.T) {
	cache := newFakeCache()
	svc := NewAnalyticsService(&fakeOrderRepo{}, cache, 60)

	d, err := svc.Dashboard(context.Background(), domain.Filters{})
	if err != nil {
		t.Fatal(err)
	}
	if d.KPIs.TotalOrders != 100 {
		t.Fatalf("total orders = %d, want 100", d.KPIs.TotalOrders)
	}
	if len(d.RevenueTrend) != 2 || len(d.Carriers) != 2 {
		t.Fatal("unexpected dashboard shape")
	}
	// Cached on second call (no new Set).
	setsAfterFirst := cache.sets
	if _, err := svc.Dashboard(context.Background(), domain.Filters{}); err != nil {
		t.Fatal(err)
	}
	if cache.sets != setsAfterFirst {
		t.Fatal("second dashboard call should be served from cache")
	}
}

func TestOrdersCaches(t *testing.T) {
	cache := newFakeCache()
	repo := &fakeOrderRepo{}
	svc := NewAnalyticsService(repo, cache, 60)

	q := domain.OrderQuery{Filters: domain.Filters{}, Page: 0, PageSize: 15}
	if _, err := svc.Orders(context.Background(), q); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Orders(context.Background(), q); err != nil {
		t.Fatal(err)
	}
	if repo.ordersCalls != 1 {
		t.Fatalf("repo.Orders called %d times, want 1 (second from cache)", repo.ordersCalls)
	}
}

func TestHashFiltersOrderIndependent(t *testing.T) {
	a := domain.Filters{Regions: []string{"EU", "UK"}, Carriers: []string{"DHL"}}
	b := domain.Filters{Regions: []string{"UK", "EU"}, Carriers: []string{"DHL"}}
	if hashFilters(a) != hashFilters(b) {
		t.Fatal("filter hash should be independent of slice order")
	}
	c := domain.Filters{Regions: []string{"EU"}}
	if hashFilters(a) == hashFilters(c) {
		t.Fatal("different filters should hash differently")
	}
}

func TestHashOrderQueryNormalizesSearch(t *testing.T) {
	a := domain.OrderQuery{Filters: domain.Filters{}, Search: "  Sticker ", SortDir: "DESC", Page: 1}
	b := domain.OrderQuery{Filters: domain.Filters{}, Search: "sticker", SortDir: "desc", Page: 1}
	if hashOrderQuery(a) != hashOrderQuery(b) {
		t.Fatal("order-query hash should normalize search/sortdir")
	}
}
