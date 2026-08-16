package service

import (
	"context"
	"fmt"
	"math"

	"analytics-dashboard-be/internal/domain"
)

// ForecastService is the forecasting tool. It applies a linear-regression trend
// blended with a 3-month moving average — deliberately simple and fully
// auditable, mirroring the frontend prototype's methodology.
type ForecastService struct {
	repo     domain.OrderRepository
	cache    domain.Cache
	cacheTTL int
}

func NewForecastService(repo domain.OrderRepository, cache domain.Cache, ttl int) *ForecastService {
	return &ForecastService{repo: repo, cache: cache, cacheTTL: ttl}
}

func (s *ForecastService) Forecast(ctx context.Context, category string, horizon int) (domain.Forecast, error) {
	if horizon <= 0 {
		horizon = 4
	}
	// Tiny keyspace (categories × horizons) and fully deterministic — cache it.
	key := fmt.Sprintf("forecast:%s:%d", category, horizon)
	var cached domain.Forecast
	if ok, _ := s.cache.Get(ctx, key, &cached); ok {
		return cached, nil
	}
	history, err := s.repo.MonthlyUnits(ctx, category)
	if err != nil {
		return domain.Forecast{}, err
	}
	f := computeForecast(category, horizon, history)
	_ = s.cache.Set(ctx, key, f, s.cacheTTL)
	return f, nil
}

func computeForecast(category string, horizon int, history []domain.MonthUnits) domain.Forecast {
	n := len(history)
	y := make([]float64, n)
	for i, h := range history {
		y[i] = float64(h.Units)
	}

	// Ordinary least-squares trend on the month index.
	var meanX, meanY float64
	for i := 0; i < n; i++ {
		meanX += float64(i)
		meanY += y[i]
	}
	if n > 0 {
		meanX /= float64(n)
		meanY /= float64(n)
	}
	var num, den float64
	for i := 0; i < n; i++ {
		dx := float64(i) - meanX
		num += dx * (y[i] - meanY)
		den += dx * dx
	}
	slope := 0.0
	if den != 0 {
		slope = num / den
	}
	intercept := meanY - slope*meanX

	// 3-month moving average damps single-month spikes.
	movingAvg := 0.0
	if n > 0 {
		start := n - 3
		if start < 0 {
			start = 0
		}
		cnt := 0
		for i := start; i < n; i++ {
			movingAvg += y[i]
			cnt++
		}
		movingAvg /= float64(cnt)
	}

	forecastUnits := make([]float64, 0, horizon)
	for h := 1; h <= horizon; h++ {
		trend := intercept + slope*float64(n-1+h)
		blended := 0.6*trend + 0.4*movingAvg
		forecastUnits = append(forecastUnits, math.Max(0, math.Round(blended)))
	}

	points := make([]domain.ForecastPoint, 0, n+horizon)
	for i, h := range history {
		actual := y[i]
		points = append(points, domain.ForecastPoint{Bucket: h.Bucket, Actual: &actual})
	}
	// Join the two lines at the boundary month.
	if len(points) > 0 {
		last := points[len(points)-1].Actual
		points[len(points)-1].Forecast = last
	}
	lastBucket := "2025-12"
	if n > 0 {
		lastBucket = history[n-1].Bucket
	}
	for i, u := range forecastUnits {
		u := u
		points = append(points, domain.ForecastPoint{Bucket: addMonths(lastBucket, i+1), Forecast: &u})
	}

	// Residual std dev → safety stock at ~95% service level.
	var ss float64
	for i := 0; i < n; i++ {
		r := y[i] - (intercept + slope*float64(i))
		ss += r * r
	}
	stdDev := 0.0
	if n > 1 {
		stdDev = math.Sqrt(ss / float64(n-1))
	}
	avgDemand := 0.0
	for _, u := range forecastUnits {
		avgDemand += u
	}
	if len(forecastUnits) > 0 {
		avgDemand /= float64(len(forecastUnits))
	}
	safety := math.Round(1.65 * stdDev)
	recommended := math.Round(avgDemand + safety)

	label := "all products"
	if category != "" {
		label = category + " products"
	}
	return domain.Forecast{
		Category:             category,
		HorizonMonths:        horizon,
		Method:               "Linear trend (60%) blended with 3-month moving average (40%)",
		Points:               points,
		ForecastUnits:        forecastUnits,
		AvgMonthlyDemand:     avgDemand,
		Slope:                slope,
		SafetyStock:          safety,
		RecommendedInventory: recommended,
		Explanation: []string{
			fmt.Sprintf("Aggregated historical orders into %d monthly buckets of units shipped for %s.", n, label),
			fmt.Sprintf("Fitted an ordinary least-squares trend: slope %.1f units/month.", slope),
			fmt.Sprintf("Blended the trend with the last 3 months' moving average (%.0f units) to damp spikes.", movingAvg),
			fmt.Sprintf("Safety stock = 1.65 × residual std dev (%.1f) ≈ %.0f units, targeting ~95%% service level.", stdDev, safety),
			fmt.Sprintf("Recommended on-hand inventory = average forecast demand (%.0f) + safety stock.", avgDemand),
		},
	}
}

// addMonths advances a YYYY-MM bucket by delta months.
func addMonths(bucket string, delta int) string {
	var y, m int
	fmt.Sscanf(bucket, "%d-%d", &y, &m)
	total := (y*12 + (m - 1)) + delta
	ny := total / 12
	nm := total%12 + 1
	return fmt.Sprintf("%04d-%02d", ny, nm)
}
