package service

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"analytics-dashboard-be/internal/domain"
)

// AskService is the AI orchestration layer. It delegates INTERPRETATION to an
// Interpreter (Gemini, or the rule-based fallback), then ROUTES the resulting
// plan to the correct deterministic tool. It never fabricates numbers — every
// figure in the answer comes from the repository.
type AskService struct {
	repo     domain.OrderRepository
	forecast *ForecastService
	interp   domain.Interpreter
}

func NewAskService(repo domain.OrderRepository, forecast *ForecastService, interp domain.Interpreter) *AskService {
	return &AskService{repo: repo, forecast: forecast, interp: interp}
}

// Suggestions are the example questions surfaced in the UI.
var Suggestions = []string{
	"Show delayed orders by week for the last 3 months",
	"Which carrier has the highest delay rate?",
	"How many orders were delivered late last month?",
	"Break down order volume by region",
	"What is the average delivery time by carrier?",
	"Predict demand for CRAYON for the next 4 months",
	"Top destinations by order volume",
}

// parsedFrom rebuilds the handler-facing window+filter struct from an
// interpretation, so the compute handlers are unchanged whether the plan came
// from Gemini or the rule parser.
func parsedFrom(it domain.Interpretation) parsed {
	return parsed{
		from:       it.Filters.From,
		to:         it.Filters.To,
		window:     it.Window,
		carriers:   it.Filters.Carriers,
		regions:    it.Filters.Regions,
		categories: it.Filters.Categories,
	}
}

// Ask runs the full flow: interpret → route → compute → explain.
func (s *AskService) Ask(ctx context.Context, question string) (domain.Answer, error) {
	it, err := s.interp.Interpret(ctx, question)
	if err != nil {
		return domain.Answer{}, err
	}
	p := parsedFrom(it)

	var ans domain.Answer
	switch it.Intent {
	case "forecast":
		ans, err = s.forecastAnswer(ctx, p, it.Category)
	case "delayed_by_week":
		ans, err = s.delayedByWeek(ctx, p)
	case "carrier_delay_rate":
		ans, err = s.carrierDelayRate(ctx, p)
	case "avg_time_by_carrier":
		ans, err = s.avgTimeByCarrier(ctx, p)
	case "late_count":
		ans, err = s.lateCount(ctx, p)
	case "breakdown":
		limit := 12
		if it.Dimension == "destination_city" {
			limit = 10
		}
		ans, err = s.dimensionBreakdown(ctx, p, it.Dimension, limit)
	case "volume_over_time":
		ans, err = s.volumeOverTime(ctx, p)
	default:
		ans, err = s.fallback(ctx, p, question)
	}
	if err != nil {
		return domain.Answer{}, err
	}

	// Record who interpreted the question, for explainability.
	src := "rule-based router"
	if it.Source == "gemini" {
		src = "Google Gemini (interpretation only — all figures computed from SQL)"
	}
	ans.Plan.Notes = append([]string{"Interpreted by: " + src}, ans.Plan.Notes...)
	return ans, nil
}

func (s *AskService) delayedByWeek(ctx context.Context, p parsed) (domain.Answer, error) {
	pts, err := s.repo.TimeSeries(ctx, p.filters(), "week")
	if err != nil {
		return domain.Answer{}, err
	}
	total, peakWeek, peak := 0, "", 0
	data := make([]map[string]interface{}, 0, len(pts))
	rows := make([][]interface{}, 0, len(pts))
	for _, pt := range pts {
		total += pt.Delayed
		if pt.Delayed > peak {
			peak, peakWeek = pt.Delayed, pt.Bucket
		}
		data = append(data, map[string]interface{}{"week": pt.Bucket, "delayed": pt.Delayed, "total": pt.Orders})
		rows = append(rows, []interface{}{pt.Bucket, pt.Delayed, pt.Orders})
	}
	return domain.Answer{
		Answer:     fmt.Sprintf("There were %d delayed orders over the %s, spread across %d weeks. The worst week began %s with %d delayed orders.", total, p.window, len(pts), peakWeek, peak),
		Confidence: "high",
		Plan: domain.QueryPlan{
			Tool: domain.ToolAnalyticsQuery, Intent: "Count delayed orders bucketed by week",
			Metrics: []string{"count(order_id) where status = 'delayed'"}, Dimensions: []string{"week(order_date)"},
			FiltersUsed: p.filterMap(), Granularity: "week",
			Notes: []string{"Weeks are ISO weeks labelled by their Monday."},
		},
		Chart: &domain.Chart{Kind: "bar", Title: "Delayed orders by week — " + p.window, XKey: "week",
			Series: []domain.ChartSeries{{Key: "delayed", Label: "Delayed orders", Color: colorDanger}}, Data: data},
		Table: &domain.Table{Columns: []string{"Week of", "Delayed", "Total orders"}, Rows: rows},
	}, nil
}

func (s *AskService) carrierDelayRate(ctx context.Context, p parsed) (domain.Answer, error) {
	b, err := s.repo.Breakdown(ctx, p.filters(), "carrier", 0)
	if err != nil {
		return domain.Answer{}, err
	}
	var filtered []domain.BreakdownRow
	for _, r := range b {
		if r.Orders >= 5 {
			filtered = append(filtered, r)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].DelayRate > filtered[j].DelayRate })
	data := make([]map[string]interface{}, 0, len(filtered))
	rows := make([][]interface{}, 0, len(filtered))
	for _, r := range filtered {
		data = append(data, map[string]interface{}{"name": r.Name, "delayRatePct": round1(r.DelayRate * 100)})
		rows = append(rows, []interface{}{r.Name, r.Orders, r.Delivered, r.Delayed, pct(r.DelayRate)})
	}
	answer := "Not enough data in this window to rank carriers."
	if len(filtered) > 0 {
		worst, best := filtered[0], filtered[len(filtered)-1]
		answer = fmt.Sprintf("%s has the highest delay rate at %s (%d delayed of %d settled orders). The best performer is %s at %s.",
			worst.Name, pct(worst.DelayRate), worst.Delayed, worst.Delivered+worst.Delayed, best.Name, pct(best.DelayRate))
	}
	return domain.Answer{
		Answer: answer, Confidence: "high",
		Plan: domain.QueryPlan{Tool: domain.ToolAnalyticsQuery, Intent: "Rank carriers by delay rate",
			Metrics: []string{"delay_rate = delayed / (delivered + delayed)"}, Dimensions: []string{"carrier"},
			FiltersUsed: p.filterMap(), Sort: "delay_rate DESC",
			Notes: []string{"Carriers with fewer than 5 orders are excluded as statistically noisy."}},
		Chart: &domain.Chart{Kind: "bar", Title: "Delay rate by carrier", XKey: "name",
			Series: []domain.ChartSeries{{Key: "delayRatePct", Label: "Delay rate %", Color: colorDanger}}, Data: data},
		Table: &domain.Table{Columns: []string{"Carrier", "Orders", "Delivered", "Delayed", "Delay rate"}, Rows: rows},
	}, nil
}

func (s *AskService) avgTimeByCarrier(ctx context.Context, p parsed) (domain.Answer, error) {
	b, err := s.repo.Breakdown(ctx, p.filters(), "carrier", 0)
	if err != nil {
		return domain.Answer{}, err
	}
	var withDelivered []domain.BreakdownRow
	for _, r := range b {
		if r.Delivered > 0 {
			withDelivered = append(withDelivered, r)
		}
	}
	sort.Slice(withDelivered, func(i, j int) bool { return withDelivered[i].AvgDeliveryDays < withDelivered[j].AvgDeliveryDays })
	data := make([]map[string]interface{}, 0, len(withDelivered))
	rows := make([][]interface{}, 0, len(withDelivered))
	for _, r := range withDelivered {
		data = append(data, map[string]interface{}{"name": r.Name, "days": round2(r.AvgDeliveryDays)})
		rows = append(rows, []interface{}{r.Name, r.Delivered, round2(r.AvgDeliveryDays)})
	}
	answer := "No delivered orders in this window."
	if len(withDelivered) > 0 {
		f, sl := withDelivered[0], withDelivered[len(withDelivered)-1]
		answer = fmt.Sprintf("%s is the fastest carrier at %.1f days average transit; %s is the slowest at %.1f days.",
			f.Name, f.AvgDeliveryDays, sl.Name, sl.AvgDeliveryDays)
	}
	return domain.Answer{
		Answer: answer, Confidence: "high",
		Plan: domain.QueryPlan{Tool: domain.ToolAnalyticsQuery, Intent: "Average transit time per carrier",
			Metrics: []string{"avg(delivery_date - order_date)"}, Dimensions: []string{"carrier"},
			FiltersUsed: mergeMap(p.filterMap(), "status", "delivered"), Sort: "avg_days ASC"},
		Chart: &domain.Chart{Kind: "bar", Title: "Average delivery time by carrier (days)", XKey: "name",
			Series: []domain.ChartSeries{{Key: "days", Label: "Avg days", Color: colorPrimary}}, Data: data},
		Table: &domain.Table{Columns: []string{"Carrier", "Delivered orders", "Avg days"}, Rows: rows},
	}, nil
}

func (s *AskService) lateCount(ctx context.Context, p parsed) (domain.Answer, error) {
	k, err := s.repo.KPIs(ctx, p.filters())
	if err != nil {
		return domain.Answer{}, err
	}
	series, err := s.repo.TimeSeries(ctx, p.filters(), "month")
	if err != nil {
		return domain.Answer{}, err
	}
	data := make([]map[string]interface{}, 0, len(series))
	rows := make([][]interface{}, 0, len(series))
	for _, pt := range series {
		data = append(data, map[string]interface{}{"bucket": fmtMonth(pt.Bucket), "delivered": pt.Delivered, "delayed": pt.Delayed})
		rows = append(rows, []interface{}{fmtMonth(pt.Bucket), pt.Orders, pt.Delivered, pt.Delayed})
	}
	return domain.Answer{
		Answer: fmt.Sprintf("%d orders were delayed during the %s, out of %d total orders — an on-time rate of %s.",
			k.DelayedOrders, p.window, k.TotalOrders, pct(k.OnTimeRate)),
		Confidence: "high",
		Plan: domain.QueryPlan{Tool: domain.ToolAnalyticsQuery, Intent: "Count delayed orders in the requested window",
			Metrics: []string{"count(order_id) where status = 'delayed'", "on_time_rate"}, Dimensions: []string{"month(order_date)"},
			FiltersUsed: p.filterMap(), Granularity: "month",
			Notes: []string{"'Late' is read from the status column; the dataset has no promised-date field, so lateness is not recomputed."}},
		Chart: &domain.Chart{Kind: "stacked-bar", Title: "Delivered vs delayed — " + p.window, XKey: "bucket",
			Series: []domain.ChartSeries{{Key: "delivered", Label: "Delivered", Color: colorSuccess}, {Key: "delayed", Label: "Delayed", Color: colorDanger}}, Data: data},
		Table: &domain.Table{Columns: []string{"Month", "Orders", "Delivered", "Delayed"}, Rows: rows},
	}, nil
}

func (s *AskService) dimensionBreakdown(ctx context.Context, p parsed, dimension string, limit int) (domain.Answer, error) {
	b, err := s.repo.Breakdown(ctx, p.filters(), dimension, limit)
	if err != nil {
		return domain.Answer{}, err
	}
	scoped, err := s.repo.KPIs(ctx, p.filters())
	if err != nil {
		return domain.Answer{}, err
	}
	data := make([]map[string]interface{}, 0, len(b))
	rows := make([][]interface{}, 0, len(b))
	for _, r := range b {
		data = append(data, map[string]interface{}{"name": r.Name, "orders": r.Orders})
		rows = append(rows, []interface{}{r.Name, r.Orders, r.Delayed, pct(r.DelayRate), fmt.Sprintf("$%.0f", r.Revenue)})
	}
	dimLabel := strings.ReplaceAll(dimension, "_", " ")
	answer := "No orders matched this window."
	if len(b) > 0 {
		top := b[0]
		share := 0.0
		if scoped.TotalOrders > 0 {
			share = float64(top.Orders) / float64(scoped.TotalOrders)
		}
		answer = fmt.Sprintf("Across %d orders in the %s, %s leads on %s with %d orders (%s of volume) and a %s delay rate.",
			scoped.TotalOrders, p.window, top.Name, dimLabel, top.Orders, pct(share), pct(top.DelayRate))
	}
	return domain.Answer{
		Answer: answer, Confidence: "high",
		Plan: domain.QueryPlan{Tool: domain.ToolAnalyticsQuery, Intent: "Order volume broken down by " + dimension,
			Metrics: []string{"count(order_id)", "sum(order_value_usd)", "delay_rate"}, Dimensions: []string{dimension},
			FiltersUsed: p.filterMap(), Sort: "orders DESC", Limit: limit},
		Chart: &domain.Chart{Kind: "bar", Title: "Orders by " + dimLabel, XKey: "name",
			Series: []domain.ChartSeries{{Key: "orders", Label: "Orders", Color: colorPrimary}}, Data: data},
		Table: &domain.Table{Columns: []string{dimension, "Orders", "Delayed", "Delay rate", "Revenue"}, Rows: rows},
	}, nil
}

func (s *AskService) volumeOverTime(ctx context.Context, p parsed) (domain.Answer, error) {
	series, err := s.repo.TimeSeries(ctx, p.filters(), "month")
	if err != nil {
		return domain.Answer{}, err
	}
	data := make([]map[string]interface{}, 0, len(series))
	rows := make([][]interface{}, 0, len(series))
	for _, pt := range series {
		data = append(data, map[string]interface{}{"bucket": fmtMonth(pt.Bucket), "orders": pt.Orders})
		rows = append(rows, []interface{}{fmtMonth(pt.Bucket), pt.Orders, pt.Delivered, pt.Delayed, fmt.Sprintf("$%.0f", pt.Revenue)})
	}
	answer := "No orders matched this window."
	if len(series) > 0 {
		first, last := series[0], series[len(series)-1]
		change := 0.0
		if first.Orders > 0 {
			change = float64(last.Orders-first.Orders) / float64(first.Orders)
		}
		answer = fmt.Sprintf("Order volume moved from %d in %s to %d in %s — a %s change over the %s.",
			first.Orders, fmtMonth(first.Bucket), last.Orders, fmtMonth(last.Bucket), pct(change), p.window)
	}
	return domain.Answer{
		Answer: answer, Confidence: "high",
		Plan: domain.QueryPlan{Tool: domain.ToolAnalyticsQuery, Intent: "Order volume over time",
			Metrics: []string{"count(order_id)", "sum(order_value_usd)"}, Dimensions: []string{"month(order_date)"},
			FiltersUsed: p.filterMap(), Granularity: "month"},
		Chart: &domain.Chart{Kind: "line", Title: "Order volume over time", XKey: "bucket",
			Series: []domain.ChartSeries{{Key: "orders", Label: "Orders", Color: colorPrimary}}, Data: data},
		Table: &domain.Table{Columns: []string{"Month", "Orders", "Delivered", "Delayed", "Revenue"}, Rows: rows},
	}, nil
}

func (s *AskService) forecastAnswer(ctx context.Context, p parsed, category string) (domain.Answer, error) {
	if category == "" && len(p.categories) > 0 {
		category = p.categories[0]
	}
	f, err := s.forecast.Forecast(ctx, category, 4)
	if err != nil {
		return domain.Answer{}, err
	}
	label := "all products"
	if category != "" {
		label = category + " products"
	}
	data := make([]map[string]interface{}, 0, len(f.Points))
	rows := make([][]interface{}, 0, len(f.Points))
	for _, pt := range f.Points {
		row := map[string]interface{}{"bucket": fmtMonth(pt.Bucket)}
		var a, fc interface{} = "—", "—"
		if pt.Actual != nil {
			row["actual"] = *pt.Actual
			a = *pt.Actual
		}
		if pt.Forecast != nil {
			row["forecast"] = *pt.Forecast
			fc = *pt.Forecast
		}
		data = append(data, row)
		rows = append(rows, []interface{}{fmtMonth(pt.Bucket), a, fc})
	}
	units := make([]string, len(f.ForecastUnits))
	for i, u := range f.ForecastUnits {
		units[i] = strconv.Itoa(int(u))
	}
	return domain.Answer{
		Answer: fmt.Sprintf("Forecast demand for %s over the next 4 months: %s units (average %.0f/month, trend %s%.1f units/month). Plan for roughly %.0f units on hand per month, including %.0f units of safety stock.",
			label, strings.Join(units, ", "), f.AvgMonthlyDemand, sign(f.Slope), f.Slope, f.RecommendedInventory, f.SafetyStock),
		Confidence: "medium",
		Plan: domain.QueryPlan{Tool: domain.ToolForecastDemand, Intent: "Forecast monthly unit demand for " + label,
			Metrics: []string{"sum(quantity)"}, Dimensions: []string{"month(order_date)"},
			FiltersUsed: mergeMap(p.filterMap(), "horizon_months", "4"), Granularity: "month",
			Notes: []string{f.Method, "400-row sample — treat the forecast as directional, not planning-grade."}},
		Chart: &domain.Chart{Kind: "forecast", Title: "Historical vs forecast demand — " + label, XKey: "bucket",
			Series: []domain.ChartSeries{{Key: "actual", Label: "Actual units", Color: colorPrimary}, {Key: "forecast", Label: "Forecast units", Color: colorAccent}}, Data: data},
		Table: &domain.Table{Columns: []string{"Month", "Actual units", "Forecast units"}, Rows: rows},
	}, nil
}

func (s *AskService) fallback(ctx context.Context, p parsed, question string) (domain.Answer, error) {
	k, err := s.repo.KPIs(ctx, p.filters())
	if err != nil {
		return domain.Answer{}, err
	}
	series, err := s.repo.TimeSeries(ctx, p.filters(), "month")
	if err != nil {
		return domain.Answer{}, err
	}
	data := make([]map[string]interface{}, 0, len(series))
	for _, pt := range series {
		data = append(data, map[string]interface{}{"bucket": fmtMonth(pt.Bucket), "delivered": pt.Delivered, "delayed": pt.Delayed})
	}
	return domain.Answer{
		Answer: fmt.Sprintf("I couldn't map %q onto a supported metric, so here is the KPI summary for the %s: %d orders, %d delivered, %d delayed, %s on-time, %.1f days average transit. Try asking about delays, carriers, regions, destinations, or a demand forecast.",
			question, p.window, k.TotalOrders, k.DeliveredOrders, k.DelayedOrders, pct(k.OnTimeRate), k.AvgDeliveryDays),
		Confidence: "low",
		Plan: domain.QueryPlan{Tool: domain.ToolAnalyticsQuery, Intent: "Unrecognised question — fell back to the KPI summary",
			Metrics: []string{"total_orders", "delivered", "delayed", "on_time_rate", "avg_delivery_days"}, Dimensions: []string{},
			FiltersUsed: p.filterMap(),
			Notes: []string{"The router refused to guess. In production the LLM returns a low-confidence plan and the API asks a clarifying question rather than inventing an answer."}},
		Chart: &domain.Chart{Kind: "stacked-bar", Title: "Delivered vs delayed by month", XKey: "bucket",
			Series: []domain.ChartSeries{{Key: "delivered", Label: "Delivered", Color: colorSuccess}, {Key: "delayed", Label: "Delayed", Color: colorDanger}}, Data: data},
	}, nil
}

// --- small helpers ---

func matches(s, pattern string) bool { return regexp.MustCompile(pattern).MatchString(s) }

func mergeMap(m map[string]string, k, v string) map[string]string {
	out := map[string]string{}
	for kk, vv := range m {
		out[kk] = vv
	}
	out[k] = v
	return out
}

func pct(x float64) string       { return fmt.Sprintf("%.1f%%", x*100) }
func round1(x float64) float64   { return float64(int(x*10+0.5)) / 10 }
func round2(x float64) float64   { return float64(int(x*100+0.5)) / 100 }
func sign(x float64) string      { if x >= 0 { return "+" }; return "" }
func monthOf(date string) string { if len(date) >= 7 { return date[:7] }; return date }
func firstOfMonth(m string) string { return m + "-01" }

var monthNames = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

func fmtMonth(bucket string) string {
	var y, m int
	fmt.Sscanf(bucket, "%d-%d", &y, &m)
	if m < 1 || m > 12 {
		return bucket
	}
	return fmt.Sprintf("%s %02d", monthNames[m-1], y%100)
}
