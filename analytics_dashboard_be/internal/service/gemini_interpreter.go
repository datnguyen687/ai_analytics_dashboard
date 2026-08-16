package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"analytics-dashboard-be/internal/domain"
)

// GeminiInterpreter uses Google Gemini to interpret a question into a validated
// Interpretation. Gemini ONLY chooses the tool and extracts parameters — it
// never sees or produces the numbers. Its output is validated against the
// dataset's allow-lists before any query runs; on any error it falls back to
// the deterministic interpreter, so /ask always works.
type GeminiInterpreter struct {
	apiKey   string
	model    string
	meta     domain.Meta
	fallback domain.Interpreter
	client   *http.Client
	baseURL  string // overridable in tests
}

const geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models/"

func NewGeminiInterpreter(apiKey, model string, meta domain.Meta, fallback domain.Interpreter) *GeminiInterpreter {
	return &GeminiInterpreter{
		apiKey:   apiKey,
		model:    model,
		meta:     meta,
		fallback: fallback,
		client:   &http.Client{Timeout: 12 * time.Second},
		baseURL:  geminiBaseURL,
	}
}

// llmPlan is the constrained JSON shape we ask Gemini to return.
type llmPlan struct {
	Tool       string   `json:"tool"`
	Intent     string   `json:"intent"`
	Dimension  string   `json:"dimension"`
	Category   string   `json:"category"`
	Horizon    int      `json:"horizon"`
	From       string   `json:"from"`
	To         string   `json:"to"`
	Window     string   `json:"window"`
	Regions    []string `json:"regions"`
	Carriers   []string `json:"carriers"`
	Categories []string `json:"categories"`
}

var validIntents = map[string]bool{
	"delayed_by_week": true, "carrier_delay_rate": true, "avg_time_by_carrier": true,
	"late_count": true, "breakdown": true, "volume_over_time": true,
	"forecast": true, "unsupported": true,
}
var validDimensions = map[string]bool{
	"carrier": true, "region": true, "product_category": true, "destination_city": true,
}

func (g *GeminiInterpreter) Interpret(ctx context.Context, question string) (domain.Interpretation, error) {
	plan, err := g.callGemini(ctx, question)
	if err != nil {
		log.Printf("gemini interpret failed (%v) — falling back to rules", err)
		return g.fallback.Interpret(ctx, question)
	}
	it, ok := g.validate(plan)
	if !ok {
		log.Printf("gemini plan failed validation — falling back to rules")
		return g.fallback.Interpret(ctx, question)
	}
	return it, nil
}

// validate turns the raw LLM plan into a safe Interpretation, clamping every
// field to the dataset's allow-lists. Anything unknown is dropped, never trusted.
func (g *GeminiInterpreter) validate(p llmPlan) (domain.Interpretation, bool) {
	intent := strings.ToLower(strings.TrimSpace(p.Intent))
	if !validIntents[intent] {
		return domain.Interpretation{}, false
	}

	it := domain.Interpretation{
		Intent:  intent,
		Tool:    domain.ToolAnalyticsQuery,
		Horizon: 4,
		Source:  "gemini",
		Window:  strings.TrimSpace(p.Window),
	}
	if intent == "forecast" {
		it.Tool = domain.ToolForecastDemand
		if p.Horizon >= 1 && p.Horizon <= 12 {
			it.Horizon = p.Horizon
		}
	}
	if validDimensions[p.Dimension] {
		it.Dimension = p.Dimension
	} else if intent == "breakdown" {
		it.Dimension = "carrier" // safe default for an under-specified breakdown
	}
	if inList(p.Category, g.meta.Categories) {
		it.Category = p.Category
	}

	// Date window: default to the full dataset range, accept only in-range dates.
	from, to := g.meta.DateMin, g.meta.DateMax
	if isDateInRange(p.From, g.meta.DateMin, g.meta.DateMax) {
		from = p.From
	}
	if isDateInRange(p.To, g.meta.DateMin, g.meta.DateMax) {
		to = p.To
	}
	if it.Window == "" {
		it.Window = "full dataset (2025)"
	}
	it.Filters = domain.Filters{
		From:       from,
		To:         to,
		Regions:    intersect(p.Regions, g.meta.Regions),
		Carriers:   intersect(p.Carriers, g.meta.Carriers),
		Categories: intersect(p.Categories, g.meta.Categories),
	}
	return it, true
}

func (g *GeminiInterpreter) systemPrompt() string {
	return fmt.Sprintf(`You interpret natural-language questions about a logistics orders dataset.
You DO NOT answer with numbers or data. You ONLY output a JSON object choosing a tool and parameters.
A separate deterministic engine runs the actual computation from your plan.

Dataset facts:
- Date range: %s to %s (year 2025).
- Carriers: %s
- Regions: %s
- Product categories: %s
- Order statuses: delivered, delayed, in_transit, exception, canceled.

Return ONLY this JSON object (no prose):
{
  "tool": "analytics.query" | "forecast.demand",
  "intent": one of ["delayed_by_week","carrier_delay_rate","avg_time_by_carrier","late_count","breakdown","volume_over_time","forecast","unsupported"],
  "dimension": "carrier" | "region" | "product_category" | "destination_city" | "",  // only for intent "breakdown"
  "category": one of the product categories or "",   // for forecasts or a category filter
  "horizon": integer months for a forecast (default 4),
  "from": "YYYY-MM-DD" or "",   // resolve relative windows ("last 3 months") to absolute dates within the range
  "to": "YYYY-MM-DD" or "",
  "window": short human label for the time window (e.g. "last 3 months", "full dataset (2025)"),
  "regions": [], "carriers": [], "categories": []   // filters explicitly named in the question, exact values only
}

Intent guide:
- "delayed_by_week": delayed orders bucketed by week.
- "carrier_delay_rate": which carrier delays most / delay rate ranking.
- "avg_time_by_carrier": average delivery/transit time per carrier.
- "late_count": how many orders were late/delayed in a window.
- "breakdown": order volume grouped by a dimension (set "dimension").
- "volume_over_time": order volume trend over months.
- "forecast": predict future demand / inventory planning (set "category" if named).
- "unsupported": anything the tools can't answer.`,
		g.meta.DateMin, g.meta.DateMax,
		strings.Join(g.meta.Carriers, ", "),
		strings.Join(g.meta.Regions, ", "),
		strings.Join(g.meta.Categories, ", "),
	)
}

// --- Gemini REST call ---

type geminiRequest struct {
	SystemInstruction geminiContent   `json:"systemInstruction"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  genConfig       `json:"generationConfig"`
}
type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}
type geminiPart struct {
	Text string `json:"text"`
}
type genConfig struct {
	ResponseMimeType string  `json:"responseMimeType"`
	Temperature      float64 `json:"temperature"`
}
type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

func (g *GeminiInterpreter) callGemini(ctx context.Context, question string) (llmPlan, error) {
	reqBody := geminiRequest{
		SystemInstruction: geminiContent{Parts: []geminiPart{{Text: g.systemPrompt()}}},
		Contents:          []geminiContent{{Role: "user", Parts: []geminiPart{{Text: question}}}},
		GenerationConfig:  genConfig{ResponseMimeType: "application/json", Temperature: 0},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return llmPlan{}, err
	}

	url := fmt.Sprintf("%s%s:generateContent", g.baseURL, g.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return llmPlan{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.apiKey)

	resp, err := g.client.Do(req)
	if err != nil {
		return llmPlan{}, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		// Include a snippet of the body — Google's error messages (retired model,
		// gated key, etc.) are the fastest way to diagnose a failure.
		snippet := string(respBody)
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		return llmPlan{}, fmt.Errorf("gemini status %d: %s", resp.StatusCode, snippet)
	}

	var gr geminiResponse
	if err := json.Unmarshal(respBody, &gr); err != nil {
		return llmPlan{}, err
	}
	if len(gr.Candidates) == 0 {
		return llmPlan{}, fmt.Errorf("gemini returned no candidates")
	}
	// Thinking models can emit multiple parts; concatenate the text parts.
	var sb strings.Builder
	for _, p := range gr.Candidates[0].Content.Parts {
		sb.WriteString(p.Text)
	}
	text := strings.TrimSpace(sb.String())
	if text == "" {
		return llmPlan{}, fmt.Errorf("gemini returned empty text")
	}

	var plan llmPlan
	if err := json.Unmarshal([]byte(text), &plan); err != nil {
		return llmPlan{}, fmt.Errorf("parse gemini json: %w", err)
	}
	return plan, nil
}

// --- validation helpers ---

func inList(v string, list []string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func intersect(want, allowed []string) []string {
	var out []string
	for _, w := range want {
		if inList(w, allowed) {
			out = append(out, w)
		}
	}
	return out
}

func isDateInRange(d, min, max string) bool {
	if len(d) != 10 {
		return false
	}
	if _, err := time.Parse("2006-01-02", d); err != nil {
		return false
	}
	return d >= min && d <= max
}
