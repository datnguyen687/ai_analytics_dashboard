package domain

// Filters is the shared query contract every analytics read accepts.
// Empty slices mean "no restriction on this dimension".
type Filters struct {
	From       string   `json:"from"`
	To         string   `json:"to"`
	Regions    []string `json:"regions"`
	Carriers   []string `json:"carriers"`
	Categories []string `json:"categories"`
}

// KPIs are the headline dashboard metrics required by the assignment.
type KPIs struct {
	TotalOrders     int     `json:"totalOrders"`
	DeliveredOrders int     `json:"deliveredOrders"`
	DelayedOrders   int     `json:"delayedOrders"`
	OnTimeRate      float64 `json:"onTimeRate"`
	AvgDeliveryDays float64 `json:"avgDeliveryDays"`
	TotalRevenue    float64 `json:"totalRevenue"`
}

// TimePoint is one time-bucketed row (month or week) of the volume/revenue series.
type TimePoint struct {
	Bucket    string  `json:"bucket"`
	Orders    int     `json:"orders"`
	Delivered int     `json:"delivered"`
	Delayed   int     `json:"delayed"`
	Revenue   float64 `json:"revenue"`
}

// StatusCount is one slice of the status-mix donut.
type StatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// BreakdownRow aggregates metrics for one value of a dimension (carrier, region…).
type BreakdownRow struct {
	Name            string  `json:"name"`
	Orders          int     `json:"orders"`
	Delivered       int     `json:"delivered"`
	Delayed         int     `json:"delayed"`
	DelayRate       float64 `json:"delayRate"`
	AvgDeliveryDays float64 `json:"avgDeliveryDays"`
	Revenue         float64 `json:"revenue"`
}

// CategoryStack is the monthly order counts split by category for the stacked area.
type CategoryStack struct {
	Keys []string                 `json:"keys"`
	Data []map[string]interface{} `json:"data"`
}

// Meta powers the filter controls: the distinct dimension values and date bounds.
type Meta struct {
	Carriers   []string `json:"carriers"`
	Regions    []string `json:"regions"`
	Categories []string `json:"categories"`
	Statuses   []string `json:"statuses"`
	DateMin    string   `json:"dateMin"`
	DateMax    string   `json:"dateMax"`
}

// Dashboard bundles every overview aggregate into a single response so the
// frontend renders the whole page from one API call.
type Dashboard struct {
	Filters       Filters        `json:"filters"`
	KPIs          KPIs           `json:"kpis"`
	RevenueTrend  []TimePoint    `json:"revenueTrend"`
	StatusMix     []StatusCount  `json:"statusMix"`
	CategoryStack CategoryStack  `json:"categoryStack"`
	Carriers      []BreakdownRow `json:"carriers"`
	Categories    []BreakdownRow `json:"categories"`
	Destinations  []BreakdownRow `json:"destinations"`
}

// OrderQuery is the parameter set for the paginated orders table.
type OrderQuery struct {
	Filters
	Search   string
	Status   string
	SortKey  string // orderDate | orderValue | transitDays | quantity
	SortDir  string // asc | desc
	Page     int
	PageSize int
}

// OrderPage is a paginated slice of orders plus the total match count.
type OrderPage struct {
	Rows     []Order `json:"rows"`
	Total    int     `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"pageSize"`
}

// ImportResult summarises a CSV import.
type ImportResult struct {
	Imported int      `json:"imported"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors"`
	Replaced bool     `json:"replaced"`
}
