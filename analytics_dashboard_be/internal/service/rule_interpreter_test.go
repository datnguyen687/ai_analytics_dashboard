package service

import (
	"context"
	"testing"

	"analytics-dashboard-be/internal/domain"
)

func ruleMeta() domain.Meta {
	return domain.Meta{
		Carriers:   []string{"DHL", "FedEx"},
		Regions:    []string{"EU", "UK"},
		Categories: []string{"CRAYON", "PAPER"},
		DateMin:    "2025-01-01",
		DateMax:    "2025-12-30",
	}
}

func TestRuleInterpreterIntents(t *testing.T) {
	r := NewRuleInterpreter(ruleMeta())
	cases := []struct {
		q          string
		wantIntent string
		wantTool   domain.Tool
	}{
		{"Show delayed orders by week", "delayed_by_week", domain.ToolAnalyticsQuery},
		{"Which carrier has the highest delay rate?", "carrier_delay_rate", domain.ToolAnalyticsQuery},
		{"average delivery time by carrier", "avg_time_by_carrier", domain.ToolAnalyticsQuery},
		{"how many orders were delayed", "late_count", domain.ToolAnalyticsQuery},
		{"break down volume by region", "breakdown", domain.ToolAnalyticsQuery},
		{"top destinations", "breakdown", domain.ToolAnalyticsQuery},
		{"order volume over time", "volume_over_time", domain.ToolAnalyticsQuery},
		{"predict demand for CRAYON", "forecast", domain.ToolForecastDemand},
		{"what is the meaning of life", "unsupported", domain.ToolAnalyticsQuery},
	}
	for _, c := range cases {
		it, err := r.Interpret(context.Background(), c.q)
		if err != nil {
			t.Fatal(err)
		}
		if it.Intent != c.wantIntent {
			t.Errorf("%q -> intent %q, want %q", c.q, it.Intent, c.wantIntent)
		}
		if it.Tool != c.wantTool {
			t.Errorf("%q -> tool %q, want %q", c.q, it.Tool, c.wantTool)
		}
		if it.Source != "rules" {
			t.Errorf("source = %q, want rules", it.Source)
		}
	}
}

func TestRuleInterpreterExtractsFilters(t *testing.T) {
	r := NewRuleInterpreter(ruleMeta())
	it, _ := r.Interpret(context.Background(), "predict demand for CRAYON in EU")
	if it.Category != "CRAYON" {
		t.Errorf("category = %q, want CRAYON", it.Category)
	}
	if len(it.Filters.Regions) != 1 || it.Filters.Regions[0] != "EU" {
		t.Errorf("regions = %v, want [EU]", it.Filters.Regions)
	}
}

func TestRuleInterpreterWindow(t *testing.T) {
	r := NewRuleInterpreter(ruleMeta())
	it, _ := r.Interpret(context.Background(), "delayed orders by week for the last 3 months")
	if it.Window != "last 3 months" {
		t.Errorf("window = %q, want 'last 3 months'", it.Window)
	}
	if it.Filters.From != "2025-10-01" {
		t.Errorf("from = %q, want 2025-10-01", it.Filters.From)
	}
}
