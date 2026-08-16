package domain

// ForecastPoint is one month on the historical-vs-forecast demand chart.
// Exactly one of Actual / Forecast is set per past/future month (they overlap
// on the boundary month so the two lines join without a gap).
type ForecastPoint struct {
	Bucket   string   `json:"bucket"`
	Actual   *float64 `json:"actual"`
	Forecast *float64 `json:"forecast"`
}

// Forecast is the full output of the demand-forecasting tool.
type Forecast struct {
	Category             string          `json:"category"`
	HorizonMonths        int             `json:"horizonMonths"`
	Method               string          `json:"method"`
	Points               []ForecastPoint `json:"points"`
	ForecastUnits        []float64       `json:"forecastUnits"`
	AvgMonthlyDemand     float64         `json:"avgMonthlyDemand"`
	Slope                float64         `json:"slope"`
	SafetyStock          float64         `json:"safetyStock"`
	RecommendedInventory float64         `json:"recommendedInventory"`
	Explanation          []string        `json:"explanation"`
}
