package postgres

import (
	"context"
	"strings"
	"testing"

	"analytics-dashboard-be/internal/domain"
)

// These tests exercise the query-building logic directly and need no database.

func TestWhereClauseEmpty(t *testing.T) {
	clause, args, next := whereClause(domain.Filters{}, 1)
	if clause != "" {
		t.Fatalf("clause = %q, want empty", clause)
	}
	if len(args) != 0 {
		t.Fatalf("args = %v, want none", args)
	}
	if next != 1 {
		t.Fatalf("next = %d, want 1", next)
	}
}

func TestWhereClauseDates(t *testing.T) {
	clause, args, next := whereClause(domain.Filters{From: "2025-01-01", To: "2025-06-01"}, 1)
	if !strings.Contains(clause, "order_date >= $1") || !strings.Contains(clause, "order_date <= $2") {
		t.Fatalf("clause = %q", clause)
	}
	if len(args) != 2 || next != 3 {
		t.Fatalf("args=%v next=%d", args, next)
	}
}

func TestWhereClauseAllDimensions(t *testing.T) {
	f := domain.Filters{
		From:       "2025-01-01",
		To:         "2025-12-30",
		Regions:    []string{"EU"},
		Carriers:   []string{"DHL", "UPS"},
		Categories: []string{"CRAYON"},
	}
	clause, args, next := whereClause(f, 1)
	for _, frag := range []string{"region = ANY($3)", "carrier = ANY($4)", "product_category = ANY($5)"} {
		if !strings.Contains(clause, frag) {
			t.Errorf("clause missing %q: %s", frag, clause)
		}
	}
	if !strings.HasPrefix(clause, "WHERE ") {
		t.Errorf("clause should start with WHERE: %q", clause)
	}
	if len(args) != 5 || next != 6 {
		t.Fatalf("args=%d next=%d, want 5/6", len(args), next)
	}
}

func TestDimensionAndSortMaps(t *testing.T) {
	if dimensionColumns["category"] != "product_category" {
		t.Error("category alias should map to product_category")
	}
	if _, ok := dimensionColumns["evil; DROP"]; ok {
		t.Error("injection-style dimension must not be in the allow-list")
	}
	if sortColumns["orderValue"] != "order_value_usd" {
		t.Error("orderValue sort mapping wrong")
	}
}

func TestBreakdownRejectsUnknownDimensionWithoutDB(t *testing.T) {
	// The dimension is validated before any query runs, so a nil DB is fine here.
	repo := NewOrderRepo(nil)
	if _, err := repo.Breakdown(context.Background(), domain.Filters{}, "bogus", 0); err == nil {
		t.Fatal("unknown dimension should error before touching the DB")
	}
}
