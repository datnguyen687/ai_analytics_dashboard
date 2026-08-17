package service

import (
	"context"
	"testing"
	"time"

	"analytics-dashboard-be/internal/domain"
)

func newOrder(id string, status domain.OrderStatus) domain.Order {
	return domain.Order{OrderID: id, Status: status, OrderDate: time.Now(), Quantity: 1, OrderValue: 10}
}

func TestCreateOrderValidation(t *testing.T) {
	svc := NewAnalyticsService(&fakeOrderRepo{}, newFakeCache(), 60)

	if _, err := svc.CreateOrder(context.Background(), newOrder("", domain.StatusDelivered)); err == nil {
		t.Fatal("empty order_id should fail validation")
	}
	if _, err := svc.CreateOrder(context.Background(), newOrder("O1", "bogus")); err == nil {
		t.Fatal("invalid status should fail validation")
	}
	bad := newOrder("O2", domain.StatusDelivered)
	bad.Quantity = -1
	if _, err := svc.CreateOrder(context.Background(), bad); err == nil {
		t.Fatal("negative quantity should fail validation")
	}
}

func TestCreateOrderSuccessAndConflict(t *testing.T) {
	repo := &fakeOrderRepo{}
	cache := newFakeCache()
	// Prime a cached dashboard so we can assert invalidation.
	_ = cache.Set(context.Background(), "dashboard:x", 1, 60)
	svc := NewAnalyticsService(repo, cache, 60)

	if _, err := svc.CreateOrder(context.Background(), newOrder("O1", domain.StatusDelivered)); err != nil {
		t.Fatal(err)
	}
	if _, ok := cache.data["dashboard:x"]; ok {
		t.Fatal("create should invalidate cached read models")
	}
	// Duplicate → conflict.
	if _, err := svc.CreateOrder(context.Background(), newOrder("O1", domain.StatusDelivered)); err != domain.ErrConflict {
		t.Fatalf("err = %v, want ErrConflict", err)
	}
}

func TestUpdateOrderNotFound(t *testing.T) {
	svc := NewAnalyticsService(&fakeOrderRepo{}, newFakeCache(), 60)
	if _, err := svc.UpdateOrder(context.Background(), newOrder("ghost", domain.StatusDelivered)); err != domain.ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestUpdateAndDeleteOrder(t *testing.T) {
	repo := &fakeOrderRepo{existing: map[string]bool{"O1": true}}
	svc := NewAnalyticsService(repo, newFakeCache(), 60)

	if _, err := svc.UpdateOrder(context.Background(), newOrder("O1", domain.StatusDelayed)); err != nil {
		t.Fatalf("update existing: %v", err)
	}
	if err := svc.DeleteOrder(context.Background(), "O1"); err != nil {
		t.Fatalf("delete existing: %v", err)
	}
	if err := svc.DeleteOrder(context.Background(), "O1"); err != domain.ErrNotFound {
		t.Fatalf("delete again err = %v, want ErrNotFound", err)
	}
}

func TestGetOrderNotFound(t *testing.T) {
	svc := NewAnalyticsService(&fakeOrderRepo{}, newFakeCache(), 60)
	if _, err := svc.GetOrder(context.Background(), "nope"); err != domain.ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
