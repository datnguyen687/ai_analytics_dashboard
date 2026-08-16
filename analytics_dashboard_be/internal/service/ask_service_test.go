package service

import (
	"context"
	"strings"
	"testing"

	"analytics-dashboard-be/internal/domain"
)

type fakeInterpreter struct {
	it domain.Interpretation
}

func (f fakeInterpreter) Interpret(context.Context, string) (domain.Interpretation, error) {
	return f.it, nil
}

func newAskSvc(it domain.Interpretation) *AskService {
	repo := &fakeOrderRepo{}
	forecast := NewForecastService(repo, newFakeCache(), 60)
	return NewAskService(repo, forecast, fakeInterpreter{it: it})
}

func baseInterp(intent string) domain.Interpretation {
	return domain.Interpretation{
		Intent:  intent,
		Tool:    domain.ToolAnalyticsQuery,
		Window:  "full dataset (2025)",
		Filters: domain.Filters{From: "2025-01-01", To: "2025-12-30"},
		Source:  "rules",
	}
}

func TestAskRoutesAllIntents(t *testing.T) {
	intents := []struct {
		it        domain.Interpretation
		wantChart bool
	}{
		{baseInterp("delayed_by_week"), true},
		{baseInterp("carrier_delay_rate"), true},
		{baseInterp("avg_time_by_carrier"), true},
		{baseInterp("late_count"), true},
		{func() domain.Interpretation { i := baseInterp("breakdown"); i.Dimension = "carrier"; return i }(), true},
		{func() domain.Interpretation { i := baseInterp("breakdown"); i.Dimension = "destination_city"; return i }(), true},
		{baseInterp("volume_over_time"), true},
		{func() domain.Interpretation {
			i := baseInterp("forecast")
			i.Tool = domain.ToolForecastDemand
			i.Category = "CRAYON"
			return i
		}(), true},
		{baseInterp("unsupported"), true},
	}
	for _, tc := range intents {
		ans, err := newAskSvc(tc.it).Ask(context.Background(), "q")
		if err != nil {
			t.Fatalf("intent %s: %v", tc.it.Intent, err)
		}
		if ans.Answer == "" {
			t.Errorf("intent %s: empty answer", tc.it.Intent)
		}
		if tc.wantChart && ans.Chart == nil {
			t.Errorf("intent %s: missing chart", tc.it.Intent)
		}
		if len(ans.Plan.Notes) == 0 || !strings.Contains(ans.Plan.Notes[0], "Interpreted by") {
			t.Errorf("intent %s: plan should note the interpreter", tc.it.Intent)
		}
	}
}

func TestAskGeminiSourceNote(t *testing.T) {
	it := baseInterp("carrier_delay_rate")
	it.Source = "gemini"
	ans, err := newAskSvc(it).Ask(context.Background(), "worst carrier?")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ans.Plan.Notes[0], "Gemini") {
		t.Fatalf("note = %q, want Gemini attribution", ans.Plan.Notes[0])
	}
}

func TestAskConfidence(t *testing.T) {
	ans, _ := newAskSvc(baseInterp("carrier_delay_rate")).Ask(context.Background(), "q")
	if ans.Confidence != "high" {
		t.Fatalf("confidence = %q, want high", ans.Confidence)
	}
}
