package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"analytics-dashboard-be/internal/domain"
)

func newGemini() *GeminiInterpreter {
	return NewGeminiInterpreter("key", "model", ruleMeta(), NewRuleInterpreter(ruleMeta()))
}

func TestGeminiValidateValidPlan(t *testing.T) {
	g := newGemini()
	it, ok := g.validate(llmPlan{
		Tool: "analytics.query", Intent: "carrier_delay_rate",
		Regions: []string{"EU"}, Carriers: []string{"DHL"},
		From: "2025-03-01", To: "2025-06-01", Window: "spring",
	})
	if !ok {
		t.Fatal("valid plan rejected")
	}
	if it.Source != "gemini" || it.Intent != "carrier_delay_rate" {
		t.Fatalf("it = %+v", it)
	}
	if len(it.Filters.Regions) != 1 || it.Filters.From != "2025-03-01" {
		t.Fatalf("filters not applied: %+v", it.Filters)
	}
}

func TestGeminiValidateRejectsUnknownIntent(t *testing.T) {
	if _, ok := newGemini().validate(llmPlan{Intent: "drop_tables"}); ok {
		t.Fatal("unknown intent should fail validation")
	}
}

func TestGeminiValidateClampsFilters(t *testing.T) {
	g := newGemini()
	it, ok := g.validate(llmPlan{
		Intent:   "breakdown",
		Dimension: "carrier",
		Regions:  []string{"EU", "MARS"}, // MARS not in meta → dropped
		Category: "UNOBTANIUM",           // not in meta → dropped
		From:     "1999-01-01",           // out of range → default
	})
	if !ok {
		t.Fatal("plan rejected")
	}
	if len(it.Filters.Regions) != 1 || it.Filters.Regions[0] != "EU" {
		t.Fatalf("regions = %v, want [EU]", it.Filters.Regions)
	}
	if it.Category != "" {
		t.Fatalf("category = %q, want empty (clamped)", it.Category)
	}
	if it.Filters.From != "2025-01-01" {
		t.Fatalf("from = %q, want dataset min (out-of-range clamped)", it.Filters.From)
	}
}

func TestGeminiForecastHorizonBounds(t *testing.T) {
	g := newGemini()
	it, _ := g.validate(llmPlan{Intent: "forecast", Horizon: 99})
	if it.Tool != domain.ToolForecastDemand {
		t.Fatal("forecast intent should set forecast tool")
	}
	if it.Horizon != 4 {
		t.Fatalf("horizon = %d, want 4 (out-of-bounds ignored)", it.Horizon)
	}
}

func TestIntersectAndDateRange(t *testing.T) {
	got := intersect([]string{"EU", "X", "UK"}, []string{"EU", "UK"})
	if len(got) != 2 {
		t.Fatalf("intersect = %v", got)
	}
	if !isDateInRange("2025-06-01", "2025-01-01", "2025-12-30") {
		t.Fatal("in-range date rejected")
	}
	if isDateInRange("bad", "2025-01-01", "2025-12-30") {
		t.Fatal("malformed date accepted")
	}
}

func geminiWithServer(t *testing.T, status int, body string) (*GeminiInterpreter, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	g := NewGeminiInterpreter("key", "gemini-x", ruleMeta(), NewRuleInterpreter(ruleMeta()))
	g.baseURL = srv.URL + "/"
	return g, srv.Close
}

func geminiEnvelope(planJSON string) string {
	return `{"candidates":[{"content":{"parts":[{"text":` + jsonString(planJSON) + `}]}}]}`
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestGeminiInterpretSuccess(t *testing.T) {
	plan := `{"tool":"analytics.query","intent":"carrier_delay_rate","window":"full"}`
	g, done := geminiWithServer(t, 200, geminiEnvelope(plan))
	defer done()

	it, err := g.Interpret(context.Background(), "worst carrier?")
	if err != nil {
		t.Fatal(err)
	}
	if it.Source != "gemini" || it.Intent != "carrier_delay_rate" {
		t.Fatalf("it = %+v", it)
	}
}

func TestGeminiInterpretFallsBackOnHTTPError(t *testing.T) {
	g, done := geminiWithServer(t, 500, `{"error":{"message":"boom"}}`)
	defer done()

	it, err := g.Interpret(context.Background(), "which carrier has the highest delay rate?")
	if err != nil {
		t.Fatal(err)
	}
	// Fell back to the rule interpreter.
	if it.Source != "rules" || it.Intent != "carrier_delay_rate" {
		t.Fatalf("expected rules fallback, got %+v", it)
	}
}

func TestGeminiInterpretFallsBackOnBadJSON(t *testing.T) {
	g, done := geminiWithServer(t, 200, geminiEnvelope(`not json at all`))
	defer done()

	it, err := g.Interpret(context.Background(), "top destinations")
	if err != nil {
		t.Fatal(err)
	}
	if it.Source != "rules" {
		t.Fatalf("expected rules fallback on bad JSON, got source %q", it.Source)
	}
}
