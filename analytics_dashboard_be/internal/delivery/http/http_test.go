package http

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"analytics-dashboard-be/internal/cache"
	"analytics-dashboard-be/internal/domain"
	"analytics-dashboard-be/internal/service"
)

// --- minimal fakes ---

type fakeRepo struct{}

func (fakeRepo) Meta(context.Context) (domain.Meta, error) {
	return domain.Meta{Carriers: []string{"DHL"}, Regions: []string{"EU"}, Categories: []string{"CRAYON"}, DateMin: "2025-01-01", DateMax: "2025-12-30"}, nil
}
func (fakeRepo) KPIs(context.Context, domain.Filters) (domain.KPIs, error) {
	return domain.KPIs{TotalOrders: 10}, nil
}
func (fakeRepo) TimeSeries(context.Context, domain.Filters, string) ([]domain.TimePoint, error) {
	return []domain.TimePoint{{Bucket: "2025-01"}}, nil
}
func (fakeRepo) StatusMix(context.Context, domain.Filters) ([]domain.StatusCount, error) {
	return []domain.StatusCount{{Status: "delivered", Count: 10}}, nil
}
func (fakeRepo) Breakdown(context.Context, domain.Filters, string, int) ([]domain.BreakdownRow, error) {
	return []domain.BreakdownRow{{Name: "DHL", Orders: 10}}, nil
}
func (fakeRepo) CategoryStack(context.Context, domain.Filters, int) (domain.CategoryStack, error) {
	return domain.CategoryStack{Keys: []string{"CRAYON"}}, nil
}
func (fakeRepo) Orders(context.Context, domain.OrderQuery) (domain.OrderPage, error) {
	return domain.OrderPage{Total: 0}, nil
}
func (fakeRepo) MonthlyUnits(context.Context, string) ([]domain.MonthUnits, error) {
	return []domain.MonthUnits{{Bucket: "2025-01", Units: 10}}, nil
}
func (fakeRepo) ImportOrders(_ context.Context, orders []domain.Order, _ bool) (int, error) {
	return len(orders), nil
}
func (fakeRepo) GetOrder(_ context.Context, id string) (domain.Order, bool, error) {
	if id == "ORD-1" {
		return domain.Order{OrderID: "ORD-1", Status: domain.StatusDelivered}, true, nil
	}
	return domain.Order{}, false, nil
}
func (fakeRepo) CreateOrder(_ context.Context, o domain.Order) error {
	if o.OrderID == "ORD-DUP" {
		return domain.ErrConflict
	}
	return nil
}
func (fakeRepo) UpdateOrder(_ context.Context, o domain.Order) (bool, error) {
	return o.OrderID == "ORD-1", nil
}
func (fakeRepo) DeleteOrder(_ context.Context, id string) (bool, error) {
	return id == "ORD-1", nil
}

func buildRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := fakeRepo{}
	c := cache.NoopCache{}
	analytics := service.NewAnalyticsService(repo, c, 60)
	forecast := service.NewForecastService(repo, c, 60)
	meta, _ := repo.Meta(context.Background())
	ask := service.NewAskService(repo, forecast, service.NewRuleInterpreter(meta))

	users := &fakeUserRepo{users: map[string]domain.User{}}
	hash, _ := service.HashPassword("pw")
	users.users["alice"] = domain.User{Username: "alice", PasswordHash: hash, Role: domain.RoleUser}
	users.users["boss"] = domain.User{Username: "boss", PasswordHash: hash, Role: domain.RoleAdmin}
	auth := service.NewAuthService(users, "secret", 24)

	h := NewHandler(analytics, forecast, ask, 200)
	authH := NewAuthHandler(auth)
	askRL := AskRateLimit{Limiter: cache.NewMemoryRateLimiter(), Limit: 1, WindowSeconds: 60}
	return NewRouter(h, authH, auth, askRL, 1024, 5*1024*1024, []string{"http://localhost:3000"})
}

type fakeUserRepo struct{ users map[string]domain.User }

func (r *fakeUserRepo) ByUsername(_ context.Context, u string) (domain.User, error) {
	usr, ok := r.users[u]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return usr, nil
}
func (r *fakeUserRepo) Upsert(context.Context, string, string, domain.Role) error { return nil }
func (r *fakeUserRepo) List(context.Context) ([]domain.User, error) {
	out := []domain.User{}
	for _, u := range r.users {
		u.PasswordHash = ""
		out = append(out, u)
	}
	return out, nil
}

// --- helpers ---

func do(t *testing.T, r *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func loginToken(t *testing.T, r *gin.Engine, user string) string {
	t.Helper()
	w := do(t, r, "POST", "/api/v1/auth/login", "", `{"username":"`+user+`","password":"pw"}`)
	if w.Code != 200 {
		t.Fatalf("login %s failed: %d %s", user, w.Code, w.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp.Token
}

func codeOf(w *httptest.ResponseRecorder) string {
	var e struct {
		Code string `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &e)
	return e.Code
}

// --- tests ---

func TestHealthz(t *testing.T) {
	if w := do(t, buildRouter(t), "GET", "/healthz", "", ""); w.Code != 200 {
		t.Fatalf("healthz = %d", w.Code)
	}
}

func TestProtectedRequiresToken(t *testing.T) {
	w := do(t, buildRouter(t), "GET", "/api/v1/dashboard", "", "")
	if w.Code != 401 || codeOf(w) != "AUTH_TOKEN_MISSING" {
		t.Fatalf("got %d %s", w.Code, codeOf(w))
	}
}

func TestLoginBadCredentials(t *testing.T) {
	w := do(t, buildRouter(t), "POST", "/api/v1/auth/login", "", `{"username":"alice","password":"nope"}`)
	if w.Code != 401 || codeOf(w) != "AUTH_INVALID_CREDENTIALS" {
		t.Fatalf("got %d %s", w.Code, codeOf(w))
	}
}

func TestDashboardWithToken(t *testing.T) {
	r := buildRouter(t)
	token := loginToken(t, r, "alice")
	w := do(t, r, "GET", "/api/v1/dashboard", token, "")
	if w.Code != 200 {
		t.Fatalf("dashboard = %d %s", w.Code, w.Body.String())
	}
}

func TestAdminGating(t *testing.T) {
	r := buildRouter(t)
	userTok := loginToken(t, r, "alice")
	adminTok := loginToken(t, r, "boss")

	if w := do(t, r, "GET", "/api/v1/admin/users", userTok, ""); w.Code != 403 || codeOf(w) != "AUTH_FORBIDDEN" {
		t.Fatalf("USER admin access = %d %s, want 403", w.Code, codeOf(w))
	}
	if w := do(t, r, "GET", "/api/v1/admin/users", adminTok, ""); w.Code != 200 {
		t.Fatalf("ADMIN admin access = %d, want 200", w.Code)
	}
}

func TestAskQuestionTooLong(t *testing.T) {
	r := buildRouter(t)
	token := loginToken(t, r, "alice")
	long := strings.Repeat("a", 250) // handler limit is 200; body limit is 1024
	w := do(t, r, "POST", "/api/v1/ask", token, `{"question":"`+long+`"}`)
	if w.Code != 400 || codeOf(w) != "VALIDATION_ERROR" {
		t.Fatalf("long question = %d %s, want 400", w.Code, codeOf(w))
	}
}

func TestAskRateLimited(t *testing.T) {
	r := buildRouter(t)
	token := loginToken(t, r, "alice")
	body := `{"question":"worst carrier?"}`

	if w := do(t, r, "POST", "/api/v1/ask", token, body); w.Code != 200 {
		t.Fatalf("first ask = %d %s", w.Code, w.Body.String())
	}
	w := do(t, r, "POST", "/api/v1/ask", token, body)
	if w.Code != 429 || codeOf(w) != "RATE_LIMITED" {
		t.Fatalf("second ask = %d %s, want 429", w.Code, codeOf(w))
	}
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After header")
	}
}

func TestGetEndpointsWithToken(t *testing.T) {
	r := buildRouter(t)
	token := loginToken(t, r, "alice")
	for _, path := range []string{
		"/api/v1/meta",
		"/api/v1/suggestions",
		"/api/v1/auth/me",
		"/api/v1/orders?q=x&status=delivered&sort=orderValue-desc&page=0&pageSize=5",
		"/api/v1/forecast?category=CRAYON&horizon=4",
	} {
		if w := do(t, r, "GET", path, token, ""); w.Code != 200 {
			t.Errorf("%s = %d %s", path, w.Code, w.Body.String())
		}
	}
}

func TestAskSucceeds(t *testing.T) {
	r := buildRouter(t)
	token := loginToken(t, r, "alice")
	w := do(t, r, "POST", "/api/v1/ask", token, `{"question":"which carrier has the highest delay rate?"}`)
	if w.Code != 200 {
		t.Fatalf("ask = %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"plan\"") {
		t.Fatal("answer missing plan")
	}
}

func TestAskEmptyQuestion(t *testing.T) {
	r := buildRouter(t)
	token := loginToken(t, r, "alice")
	w := do(t, r, "POST", "/api/v1/ask", token, `{"question":"  "}`)
	if w.Code != 400 || codeOf(w) != "VALIDATION_ERROR" {
		t.Fatalf("empty question = %d %s", w.Code, codeOf(w))
	}
}

func TestInvalidTokenRejected(t *testing.T) {
	w := do(t, buildRouter(t), "GET", "/api/v1/meta", "garbage.token.here", "")
	if w.Code != 401 || codeOf(w) != "AUTH_TOKEN_INVALID" {
		t.Fatalf("got %d %s", w.Code, codeOf(w))
	}
}

func TestOrderCRUD(t *testing.T) {
	r := buildRouter(t)
	userTok := loginToken(t, r, "alice")
	adminTok := loginToken(t, r, "boss")

	valid := `{"orderId":"ORD-NEW","orderDate":"2025-06-01","status":"delivered","quantity":1,"orderValue":10}`

	// USER cannot create.
	if w := do(t, r, "POST", "/api/v1/admin/orders", userTok, valid); w.Code != 403 {
		t.Fatalf("USER create = %d, want 403", w.Code)
	}
	// ADMIN create → 201.
	if w := do(t, r, "POST", "/api/v1/admin/orders", adminTok, valid); w.Code != 201 {
		t.Fatalf("create = %d %s, want 201", w.Code, w.Body.String())
	}
	// Duplicate → 409.
	dup := `{"orderId":"ORD-DUP","orderDate":"2025-06-01","status":"delivered"}`
	if w := do(t, r, "POST", "/api/v1/admin/orders", adminTok, dup); w.Code != 409 || codeOf(w) != "CONFLICT" {
		t.Fatalf("dup create = %d %s, want 409", w.Code, codeOf(w))
	}
	// Invalid date → 400.
	if w := do(t, r, "POST", "/api/v1/admin/orders", adminTok, `{"orderId":"X","orderDate":"nope","status":"delivered"}`); w.Code != 400 {
		t.Fatalf("bad date = %d, want 400", w.Code)
	}
	// Read single (both roles).
	if w := do(t, r, "GET", "/api/v1/orders/ORD-1", userTok, ""); w.Code != 200 {
		t.Fatalf("get existing = %d, want 200", w.Code)
	}
	if w := do(t, r, "GET", "/api/v1/orders/GHOST", userTok, ""); w.Code != 404 || codeOf(w) != "NOT_FOUND" {
		t.Fatalf("get missing = %d %s, want 404", w.Code, codeOf(w))
	}
	// Update existing → 200; missing → 404.
	upd := `{"orderId":"ORD-1","orderDate":"2025-06-02","status":"delayed"}`
	if w := do(t, r, "PUT", "/api/v1/admin/orders/ORD-1", adminTok, upd); w.Code != 200 {
		t.Fatalf("update = %d %s, want 200", w.Code, w.Body.String())
	}
	if w := do(t, r, "PUT", "/api/v1/admin/orders/GHOST", adminTok,
		`{"orderId":"GHOST","orderDate":"2025-06-02","status":"delayed"}`); w.Code != 404 {
		t.Fatalf("update missing = %d, want 404", w.Code)
	}
	// Delete existing → 200; missing → 404; USER → 403.
	if w := do(t, r, "DELETE", "/api/v1/admin/orders/ORD-1", userTok, ""); w.Code != 403 {
		t.Fatalf("USER delete = %d, want 403", w.Code)
	}
	if w := do(t, r, "DELETE", "/api/v1/admin/orders/ORD-1", adminTok, ""); w.Code != 200 {
		t.Fatalf("delete = %d, want 200", w.Code)
	}
	if w := do(t, r, "DELETE", "/api/v1/admin/orders/GHOST", adminTok, ""); w.Code != 404 {
		t.Fatalf("delete missing = %d, want 404", w.Code)
	}
}

func TestBodyTooLarge(t *testing.T) {
	r := buildRouter(t)
	token := loginToken(t, r, "alice")
	big := `{"question":"` + strings.Repeat("x", 2000) + `"}` // > 1024 body limit
	w := do(t, r, "POST", "/api/v1/ask", token, big)
	if w.Code != 413 || codeOf(w) != "PAYLOAD_TOO_LARGE" {
		t.Fatalf("big body = %d %s, want 413", w.Code, codeOf(w))
	}
}
