package service

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"analytics-dashboard-be/internal/domain"
)

// RuleInterpreter is the deterministic, dependency-free interpreter. It is used
// directly when no LLM key is configured, and as the fallback when the Gemini
// interpreter errors — so /ask always works.
type RuleInterpreter struct {
	meta domain.Meta
}

func NewRuleInterpreter(meta domain.Meta) *RuleInterpreter {
	return &RuleInterpreter{meta: meta}
}

var lastMonthsRe = regexp.MustCompile(`last (\d+) months?`)

func (r *RuleInterpreter) Interpret(_ context.Context, question string) (domain.Interpretation, error) {
	p := r.parse(question)
	low := strings.ToLower(question)

	it := domain.Interpretation{
		Tool:    domain.ToolAnalyticsQuery,
		Window:  p.window,
		Filters: p.filters(),
		Horizon: 4,
		Source:  "rules",
	}

	switch {
	case matches(low, `predict|forecast|demand|inventory|plan`):
		it.Tool = domain.ToolForecastDemand
		it.Intent = "forecast"
		if len(p.categories) > 0 {
			it.Category = p.categories[0]
		}
	case strings.Contains(low, "delay") && strings.Contains(low, "week"):
		it.Intent = "delayed_by_week"
	case strings.Contains(low, "carrier") && matches(low, `delay|worst|highest|late`):
		it.Intent = "carrier_delay_rate"
	case strings.Contains(low, "carrier") && matches(low, `delivery time|average|speed|fastest`):
		it.Intent = "avg_time_by_carrier"
	case matches(low, `late|delayed`) && matches(low, `how many|count|number`):
		it.Intent = "late_count"
	case strings.Contains(low, "region"):
		it.Intent, it.Dimension = "breakdown", "region"
	case matches(low, `destination|city|route`):
		it.Intent, it.Dimension = "breakdown", "destination_city"
	case matches(low, `category|product|sku`):
		it.Intent, it.Dimension = "breakdown", "product_category"
	case strings.Contains(low, "carrier"):
		it.Intent, it.Dimension = "breakdown", "carrier"
	case matches(low, `volume|trend|over time|monthly`):
		it.Intent = "volume_over_time"
	default:
		it.Intent = "unsupported"
	}
	return it, nil
}

// parsed is the intermediate window+filter extraction shared by the handlers.
type parsed struct {
	from, to   string
	window     string
	carriers   []string
	regions    []string
	categories []string
}

func (r *RuleInterpreter) parse(q string) parsed {
	low := strings.ToLower(q)
	p := parsed{from: r.meta.DateMin, to: r.meta.DateMax, window: "full dataset (2025)"}

	switch {
	case lastMonthsRe.MatchString(low):
		m := lastMonthsRe.FindStringSubmatch(low)
		n, _ := strconv.Atoi(m[1])
		p.from = firstOfMonth(addMonths(monthOf(r.meta.DateMax), -(n - 1)))
		p.window = "last " + m[1] + " months"
	case strings.Contains(low, "last month"):
		p.from = firstOfMonth(monthOf(r.meta.DateMax))
		p.window = "last month"
	case strings.Contains(low, "last quarter"), strings.Contains(low, "last 3 months"), strings.Contains(low, "q4"):
		p.from = firstOfMonth(addMonths(monthOf(r.meta.DateMax), -2))
		p.window = "last 3 months"
	}

	for _, c := range r.meta.Carriers {
		if strings.Contains(low, strings.ToLower(c)) {
			p.carriers = append(p.carriers, c)
		}
	}
	for _, rg := range r.meta.Regions {
		if strings.Contains(low, strings.ToLower(rg)) {
			p.regions = append(p.regions, rg)
		}
	}
	for _, c := range r.meta.Categories {
		if strings.Contains(low, strings.ToLower(c)) {
			p.categories = append(p.categories, c)
		}
	}
	return p
}

func (p parsed) filters() domain.Filters {
	return domain.Filters{From: p.from, To: p.to, Regions: p.regions, Carriers: p.carriers, Categories: p.categories}
}

func (p parsed) filterMap() map[string]string {
	m := map[string]string{"order_date": p.from + " .. " + p.to}
	if len(p.carriers) > 0 {
		m["carrier"] = strings.Join(p.carriers, ", ")
	}
	if len(p.regions) > 0 {
		m["region"] = strings.Join(p.regions, ", ")
	}
	if len(p.categories) > 0 {
		m["product_category"] = strings.Join(p.categories, ", ")
	}
	return m
}
