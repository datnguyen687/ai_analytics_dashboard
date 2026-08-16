package service

import (
	"context"
	"testing"

	"analytics-dashboard-be/internal/domain"
)

func TestComputeForecast(t *testing.T) {
	history := []domain.MonthUnits{
		{Bucket: "2025-01", Units: 30}, {Bucket: "2025-02", Units: 24},
		{Bucket: "2025-03", Units: 20}, {Bucket: "2025-04", Units: 16},
	}
	f := computeForecast("CRAYON", 4, history)

	if len(f.ForecastUnits) != 4 {
		t.Fatalf("forecast units = %d, want 4", len(f.ForecastUnits))
	}
	if f.Slope >= 0 {
		t.Fatalf("slope = %f, want negative (declining series)", f.Slope)
	}
	// history (4) + boundary-joined + horizon (4) points.
	if len(f.Points) != len(history)+4 {
		t.Fatalf("points = %d, want %d", len(f.Points), len(history)+4)
	}
	for _, u := range f.ForecastUnits {
		if u < 0 {
			t.Fatal("forecast units must never be negative")
		}
	}
	if f.RecommendedInventory < f.SafetyStock {
		t.Fatal("recommended inventory should include safety stock")
	}
	if len(f.Explanation) == 0 {
		t.Fatal("explanation missing")
	}
}

func TestForecastServiceCaches(t *testing.T) {
	cache := newFakeCache()
	svc := NewForecastService(&fakeOrderRepo{}, cache, 60)

	f1, err := svc.Forecast(context.Background(), "CRAYON", 4)
	if err != nil {
		t.Fatal(err)
	}
	if f1.Category != "CRAYON" {
		t.Fatalf("category = %q", f1.Category)
	}
	setsAfter := cache.sets
	if _, err := svc.Forecast(context.Background(), "CRAYON", 4); err != nil {
		t.Fatal(err)
	}
	if cache.sets != setsAfter {
		t.Fatal("second forecast should be cached")
	}
}

func TestForecastHorizonDefault(t *testing.T) {
	svc := NewForecastService(&fakeOrderRepo{}, newFakeCache(), 60)
	f, _ := svc.Forecast(context.Background(), "", 0)
	if f.HorizonMonths != 4 {
		t.Fatalf("default horizon = %d, want 4", f.HorizonMonths)
	}
}

func TestAddMonths(t *testing.T) {
	cases := []struct{ in string; d int; want string }{
		{"2025-01", 1, "2025-02"},
		{"2025-12", 1, "2026-01"},
		{"2025-03", -2, "2025-01"},
	}
	for _, c := range cases {
		if got := addMonths(c.in, c.d); got != c.want {
			t.Errorf("addMonths(%q,%d) = %q, want %q", c.in, c.d, got, c.want)
		}
	}
}
