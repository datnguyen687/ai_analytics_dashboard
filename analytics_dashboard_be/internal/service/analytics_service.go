package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"golang.org/x/sync/errgroup"

	"analytics-dashboard-be/internal/domain"
)

// AnalyticsService orchestrates the read models. It depends only on the
// repository and cache ports, so it can be unit-tested with fakes.
type AnalyticsService struct {
	repo     domain.OrderRepository
	cache    domain.Cache
	cacheTTL int
}

func NewAnalyticsService(repo domain.OrderRepository, cache domain.Cache, ttl int) *AnalyticsService {
	return &AnalyticsService{repo: repo, cache: cache, cacheTTL: ttl}
}

func (s *AnalyticsService) Meta(ctx context.Context) (domain.Meta, error) {
	var m domain.Meta
	if ok, _ := s.cache.Get(ctx, "meta", &m); ok {
		return m, nil
	}
	m, err := s.repo.Meta(ctx)
	if err != nil {
		return m, err
	}
	_ = s.cache.Set(ctx, "meta", m, s.cacheTTL)
	return m, nil
}

// Dashboard assembles every overview aggregate in a single service call so the
// frontend needs exactly one HTTP request per filter change. The whole payload
// is cached under a hash of the filters.
func (s *AnalyticsService) Dashboard(ctx context.Context, f domain.Filters) (domain.Dashboard, error) {
	key := "dashboard:" + hashFilters(f)
	var cached domain.Dashboard
	if ok, _ := s.cache.Get(ctx, key, &cached); ok {
		return cached, nil
	}

	var d domain.Dashboard
	d.Filters = f

	// The seven aggregates are independent — run them concurrently. Each goroutine
	// writes a distinct field of d (no shared-write race). errgroup cancels the
	// shared context and returns the first error.
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { d.KPIs, err = s.repo.KPIs(gctx, f); return })
	g.Go(func() (err error) { d.RevenueTrend, err = s.repo.TimeSeries(gctx, f, "month"); return })
	g.Go(func() (err error) { d.StatusMix, err = s.repo.StatusMix(gctx, f); return })
	g.Go(func() (err error) { d.CategoryStack, err = s.repo.CategoryStack(gctx, f, 5); return })
	g.Go(func() (err error) { d.Carriers, err = s.repo.Breakdown(gctx, f, "carrier", 0); return })
	g.Go(func() (err error) { d.Categories, err = s.repo.Breakdown(gctx, f, "product_category", 0); return })
	g.Go(func() (err error) { d.Destinations, err = s.repo.Breakdown(gctx, f, "destination_city", 8); return })
	if err := g.Wait(); err != nil {
		return domain.Dashboard{}, err
	}

	_ = s.cache.Set(ctx, key, d, s.cacheTTL)
	return d, nil
}

// Orders serves the paginated table from cache when possible. The cache key is a
// hash of the FULL normalized query (filters + search + status + sort + page), so
// repeated searches, sort toggles, and page navigation are served from Redis
// instead of re-querying Postgres each time.
func (s *AnalyticsService) Orders(ctx context.Context, q domain.OrderQuery) (domain.OrderPage, error) {
	key := "orders:" + hashOrderQuery(q)
	var cached domain.OrderPage
	if ok, _ := s.cache.Get(ctx, key, &cached); ok {
		return cached, nil
	}
	page, err := s.repo.Orders(ctx, q)
	if err != nil {
		return page, err
	}
	_ = s.cache.Set(ctx, key, page, s.cacheTTL)
	return page, nil
}

// normalizeFilters sorts the dimension lists so that filter sets differing only
// in order (e.g. [EU,UK] vs [UK,EU]) map to the SAME cache key.
func normalizeFilters(f domain.Filters) domain.Filters {
	cp := f
	cp.Regions = sortedCopy(f.Regions)
	cp.Carriers = sortedCopy(f.Carriers)
	cp.Categories = sortedCopy(f.Categories)
	return cp
}

func sortedCopy(xs []string) []string {
	if len(xs) == 0 {
		return xs
	}
	out := append([]string(nil), xs...)
	sort.Strings(out)
	return out
}

func hashFilters(f domain.Filters) string {
	b, _ := json.Marshal(normalizeFilters(f))
	sum := sha1.Sum(b)
	return hex.EncodeToString(sum[:])
}

func hashOrderQuery(q domain.OrderQuery) string {
	// Normalize into a canonical, order-independent shape before hashing.
	norm := struct {
		F        domain.Filters
		Search   string
		Status   string
		SortKey  string
		SortDir  string
		Page     int
		PageSize int
	}{
		F:        normalizeFilters(q.Filters),
		Search:   strings.ToLower(strings.TrimSpace(q.Search)),
		Status:   q.Status,
		SortKey:  q.SortKey,
		SortDir:  strings.ToLower(q.SortDir),
		Page:     q.Page,
		PageSize: q.PageSize,
	}
	b, _ := json.Marshal(norm)
	sum := sha1.Sum(b)
	return hex.EncodeToString(sum[:])
}
