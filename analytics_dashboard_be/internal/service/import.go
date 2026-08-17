package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"analytics-dashboard-be/internal/domain"
)

// requiredCSVColumns are the columns an orders CSV must contain (matched by name,
// so column order doesn't matter).
var requiredCSVColumns = []string{
	"client_id", "order_id", "order_date", "delivery_date", "carrier", "origin_city",
	"destination_city", "status", "sku", "product_category", "quantity",
	"unit_price_usd", "order_value_usd", "is_promo", "promo_discount_pct", "region", "warehouse",
}

const maxReportedRowErrors = 20

// ParseOrdersCSV reads an orders CSV into domain.Order values. It maps columns by
// header name, computes transit_days, and collects per-row errors (bad rows are
// skipped, not fatal) up to a cap.
func ParseOrdersCSV(r io.Reader) ([]domain.Order, []string, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1 // tolerate ragged rows; we validate per row
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("could not read CSV: %w", err)
	}
	if len(records) < 2 {
		return nil, nil, fmt.Errorf("CSV has a header but no data rows")
	}

	col := map[string]int{}
	for i, h := range records[0] {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	for _, c := range requiredCSVColumns {
		if _, ok := col[c]; !ok {
			return nil, nil, fmt.Errorf("missing required column %q", c)
		}
	}

	get := func(rec []string, name string) string {
		i, ok := col[name]
		if !ok || i >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[i])
	}

	orders := make([]domain.Order, 0, len(records)-1)
	var rowErrs []string
	for n, rec := range records[1:] {
		line := n + 2 // 1-based, plus header

		orderDate, err := time.Parse("2006-01-02", get(rec, "order_date"))
		if err != nil {
			addRowErr(&rowErrs, line, "invalid order_date (want YYYY-MM-DD)")
			continue
		}
		if get(rec, "order_id") == "" {
			addRowErr(&rowErrs, line, "missing order_id")
			continue
		}

		o := domain.Order{
			ClientID:        get(rec, "client_id"),
			OrderID:         get(rec, "order_id"),
			OrderDate:       orderDate,
			Carrier:         get(rec, "carrier"),
			OriginCity:      get(rec, "origin_city"),
			DestinationCity: get(rec, "destination_city"),
			Status:          domain.OrderStatus(get(rec, "status")),
			SKU:             get(rec, "sku"),
			Category:        get(rec, "product_category"),
			Quantity:        atoiSafe(get(rec, "quantity")),
			UnitPrice:       atofSafe(get(rec, "unit_price_usd")),
			OrderValue:      atofSafe(get(rec, "order_value_usd")),
			IsPromo:         get(rec, "is_promo") == "1" || strings.EqualFold(get(rec, "is_promo"), "true"),
			PromoDiscount:   atofSafe(get(rec, "promo_discount_pct")),
			Region:          get(rec, "region"),
			Warehouse:       get(rec, "warehouse"),
		}
		if dd := get(rec, "delivery_date"); dd != "" {
			if parsed, err := time.Parse("2006-01-02", dd); err == nil {
				o.DeliveryDate = &parsed
				days := int(parsed.Sub(orderDate).Hours() / 24)
				o.TransitDays = &days
			}
		}
		orders = append(orders, o)
	}

	if len(orders) == 0 {
		return nil, rowErrs, fmt.Errorf("no valid rows found")
	}
	return orders, rowErrs, nil
}

func addRowErr(errs *[]string, line int, msg string) {
	if len(*errs) < maxReportedRowErrors {
		*errs = append(*errs, fmt.Sprintf("row %d: %s", line, msg))
	}
}

func atoiSafe(s string) int       { n, _ := strconv.Atoi(s); return n }
func atofSafe(s string) float64   { f, _ := strconv.ParseFloat(s, 64); return f }
