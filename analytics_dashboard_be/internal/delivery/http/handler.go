package http

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"analytics-dashboard-be/internal/domain"
	"analytics-dashboard-be/internal/service"
)

// Handler holds the services the routes delegate to.
type Handler struct {
	analytics        *service.AnalyticsService
	forecast         *service.ForecastService
	ask              *service.AskService
	maxQuestionChars int
}

func NewHandler(a *service.AnalyticsService, f *service.ForecastService, ask *service.AskService, maxQuestionChars int) *Handler {
	if maxQuestionChars <= 0 {
		maxQuestionChars = 1000
	}
	return &Handler{analytics: a, forecast: f, ask: ask, maxQuestionChars: maxQuestionChars}
}

// parseFilters reads the shared filter contract from the query string. List
// params are comma-separated, e.g. ?regions=EU,UK&carriers=DHL.
func parseFilters(c *gin.Context) domain.Filters {
	list := func(key string) []string {
		raw := strings.TrimSpace(c.Query(key))
		if raw == "" {
			return nil
		}
		parts := strings.Split(raw, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	return domain.Filters{
		From:       c.Query("from"),
		To:         c.Query("to"),
		Regions:    list("regions"),
		Carriers:   list("carriers"),
		Categories: list("categories"),
	}
}

func (h *Handler) Meta(c *gin.Context) {
	m, err := h.analytics.Meta(c.Request.Context())
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *Handler) Suggestions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"suggestions": service.Suggestions})
}

func (h *Handler) Dashboard(c *gin.Context) {
	d, err := h.analytics.Dashboard(c.Request.Context(), parseFilters(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, d)
}

// importMaxBytes bounds the CSV upload; set from config in the router.
var importMaxBytes int64 = 10 * 1024 * 1024

// ImportOrders ingests an uploaded orders CSV (admin-only). Multipart field
// "file" is the CSV; optional "replace=true" truncates before importing.
func (h *Handler) ImportOrders(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, importMaxBytes)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		fail(c, domain.NewAPIError(http.StatusBadRequest, "VALIDATION_ERROR", "a CSV file (field 'file') is required"))
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		fail(c, domain.ErrInternal)
		return
	}
	defer f.Close()

	replace := c.PostForm("replace") == "true"
	result, err := h.analytics.ImportOrders(c.Request.Context(), f, replace)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) Orders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "15"))
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 15
	}
	if page < 0 {
		page = 0
	}
	sortKey, sortDir := "orderDate", "desc"
	if sort := c.Query("sort"); sort != "" {
		if idx := strings.LastIndex(sort, "-"); idx > 0 {
			sortKey, sortDir = sort[:idx], sort[idx+1:]
		}
	}
	q := domain.OrderQuery{
		Filters:  parseFilters(c),
		Search:   c.Query("q"),
		Status:   c.Query("status"),
		SortKey:  sortKey,
		SortDir:  sortDir,
		Page:     page,
		PageSize: pageSize,
	}
	res, err := h.analytics.Orders(c.Request.Context(), q)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Forecast(c *gin.Context) {
	horizon, _ := strconv.Atoi(c.DefaultQuery("horizon", "4"))
	f, err := h.forecast.Forecast(c.Request.Context(), c.Query("category"), horizon)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, f)
}

type askRequest struct {
	Question string `json:"question"`
}

func (h *Handler) Ask(c *gin.Context) {
	var req askRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Question) == "" {
		fail(c, domain.NewAPIError(http.StatusBadRequest, "VALIDATION_ERROR", "question is required"))
		return
	}
	// Cap the question length (rune count, so multi-byte input is measured fairly).
	if utf8.RuneCountInString(req.Question) > h.maxQuestionChars {
		fail(c, domain.NewAPIError(http.StatusBadRequest, "VALIDATION_ERROR",
			fmt.Sprintf("question must be at most %d characters", h.maxQuestionChars)))
		return
	}
	ans, err := h.ask.Ask(c.Request.Context(), req.Question)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, ans)
}

// fail writes a structured {code, message} error. Known domain.APIError values
// keep their code/status; anything else collapses to a generic INTERNAL_ERROR so
// we never leak internals to the client.
func fail(c *gin.Context, err error) {
	if apiErr, ok := err.(*domain.APIError); ok {
		c.AbortWithStatusJSON(apiErr.HTTPStatus(), apiErr)
		return
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, domain.ErrInternal)
}
