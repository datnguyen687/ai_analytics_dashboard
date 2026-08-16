package domain

// Tool identifies which computation path the AI layer routed a question to.
type Tool string

const (
	ToolAnalyticsQuery Tool = "analytics.query"
	ToolForecastDemand Tool = "forecast.demand"
)

// QueryPlan is the structured interpretation the AI layer emits. The AI ONLY
// ever produces this plan — never the numbers — and it is validated against an
// allow-list before any computation runs.
type QueryPlan struct {
	Tool        Tool              `json:"tool"`
	Intent      string            `json:"intent"`
	Metrics     []string          `json:"metrics"`
	Dimensions  []string          `json:"dimensions"`
	FiltersUsed map[string]string `json:"filters"`
	Granularity string            `json:"granularity,omitempty"`
	Sort        string            `json:"sort,omitempty"`
	Limit       int               `json:"limit,omitempty"`
	Notes       []string          `json:"notes,omitempty"`
}

// Chart is a renderable chart spec returned alongside an answer.
type Chart struct {
	Kind   string                   `json:"kind"` // line | bar | stacked-bar | pie | forecast
	Title  string                   `json:"title"`
	XKey   string                   `json:"xKey"`
	Series []ChartSeries            `json:"series"`
	Data   []map[string]interface{} `json:"data"`
}

type ChartSeries struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Color string `json:"color"`
}

// Table is the "access to underlying data" required by the explainability rules.
type Table struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
}

// Answer is the full response to a natural-language question.
type Answer struct {
	Answer     string    `json:"answer"`
	Plan       QueryPlan `json:"plan"`
	Chart      *Chart    `json:"chart,omitempty"`
	Table      *Table    `json:"table,omitempty"`
	Confidence string    `json:"confidence"` // high | medium | low
}
