package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"

	"analytics-dashboard-be/internal/domain"
)

// testDB connects to the dev database. These are integration-style tests: they
// skip when Postgres isn't reachable so `go test` stays green without a DB.
func testDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Generic local default; set DATABASE_URL to run these against your DB.
		dsn = "postgres://postgres:postgres@localhost:5432/analytics_dashboard?sslmode=disable"
	}
	db, err := Connect(dsn)
	if err != nil {
		t.Skipf("skipping: no database (%v)", err)
	}
	var n int
	if err := db.Get(&n, "SELECT COUNT(*) FROM orders"); err != nil || n == 0 {
		db.Close()
		t.Skipf("skipping: orders table missing or empty (%v)", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOrderRepoAggregates(t *testing.T) {
	repo := NewOrderRepo(testDB(t))
	ctx := context.Background()
	f := domain.Filters{}

	meta, err := repo.Meta(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(meta.Carriers) == 0 || meta.DateMin == "" {
		t.Fatal("meta empty")
	}

	k, err := repo.KPIs(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	if k.TotalOrders == 0 {
		t.Fatal("kpis: no orders")
	}
	if k.OnTimeRate < 0 || k.OnTimeRate > 1 {
		t.Fatalf("on-time rate out of range: %f", k.OnTimeRate)
	}

	months, err := repo.TimeSeries(ctx, f, "month")
	if err != nil {
		t.Fatal(err)
	}
	if len(months) == 0 {
		t.Fatal("no monthly buckets")
	}
	if _, err := repo.TimeSeries(ctx, f, "week"); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.StatusMix(ctx, f); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CategoryStack(ctx, f, 5); err != nil {
		t.Fatal(err)
	}

	carriers, err := repo.Breakdown(ctx, f, "carrier", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(carriers) == 0 {
		t.Fatal("no carrier breakdown")
	}
	if _, err := repo.Breakdown(ctx, f, "bogus_dim", 0); err == nil {
		t.Fatal("unsupported dimension should error")
	}

	units, err := repo.MonthlyUnits(ctx, "CRAYON")
	if err != nil {
		t.Fatal(err)
	}
	if len(units) == 0 {
		t.Fatal("no monthly units")
	}
}

func TestOrderRepoFiltersAndOrders(t *testing.T) {
	repo := NewOrderRepo(testDB(t))
	ctx := context.Background()

	// Filter should reduce or equal the full set.
	full, _ := repo.KPIs(ctx, domain.Filters{})
	eu, _ := repo.KPIs(ctx, domain.Filters{Regions: []string{"EU"}})
	if eu.TotalOrders > full.TotalOrders {
		t.Fatal("filtered count exceeds total")
	}

	page, err := repo.Orders(ctx, domain.OrderQuery{
		Filters: domain.Filters{}, SortKey: "orderValue", SortDir: "desc", Page: 0, PageSize: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) > 5 || page.Total == 0 {
		t.Fatalf("orders page invalid: rows=%d total=%d", len(page.Rows), page.Total)
	}
	// Search narrows results.
	sticker, err := repo.Orders(ctx, domain.OrderQuery{
		Filters: domain.Filters{}, Search: "STICKER", Page: 0, PageSize: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if sticker.Total > page.Total {
		t.Fatal("search should not increase total")
	}
}

func TestUserRepoRoundtrip(t *testing.T) {
	repo := NewUserRepo(testDB(t))
	ctx := context.Background()
	const uname = "__test_user__"

	if err := repo.Upsert(ctx, uname, "hash1", domain.RoleUser); err != nil {
		t.Fatal(err)
	}
	u, err := repo.ByUsername(ctx, uname)
	if err != nil {
		t.Fatal(err)
	}
	if u.Role != domain.RoleUser || u.PasswordHash != "hash1" {
		t.Fatalf("user = %+v", u)
	}
	// Upsert updates in place.
	if err := repo.Upsert(ctx, uname, "hash2", domain.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	u, _ = repo.ByUsername(ctx, uname)
	if u.Role != domain.RoleAdmin || u.PasswordHash != "hash2" {
		t.Fatalf("upsert did not update: %+v", u)
	}

	if _, err := repo.List(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ByUsername(ctx, "does-not-exist"); err != domain.ErrUserNotFound {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}

	// cleanup
	repo.db.Exec("DELETE FROM users WHERE username = $1", uname)
}
