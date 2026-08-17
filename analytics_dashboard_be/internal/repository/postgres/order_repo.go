package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"analytics-dashboard-be/internal/domain"
)

// OrderRepo is the Postgres/sqlx implementation of domain.OrderRepository.
type OrderRepo struct {
	db *sqlx.DB
}

func NewOrderRepo(db *sqlx.DB) *OrderRepo { return &OrderRepo{db: db} }

// dimensionColumns whitelists which columns a caller may group by. This is the
// guardrail that lets us build dynamic SQL without ever interpolating
// user-supplied identifiers.
var dimensionColumns = map[string]string{
	"carrier":          "carrier",
	"region":           "region",
	"product_category": "product_category",
	"category":         "product_category",
	"destination_city": "destination_city",
	"warehouse":        "warehouse",
	"sku":              "sku",
}

// orderColumns is the explicit projection matching domain.Order (excludes the
// surrogate id PK, which has no struct field).
const orderColumns = `client_id, order_id, order_date, delivery_date, carrier, origin_city,
	destination_city, status, sku, product_category, quantity, unit_price_usd, order_value_usd,
	is_promo, promo_discount_pct, region, warehouse, transit_days`

var sortColumns = map[string]string{
	"orderDate":   "order_date",
	"orderValue":  "order_value_usd",
	"transitDays": "transit_days",
	"quantity":    "quantity",
}

// whereClause builds a parameterised WHERE fragment from Filters. Every value is
// a bind parameter ($n) or a pq.Array — never string-interpolated.
func whereClause(f domain.Filters, start int) (string, []interface{}, int) {
	var conds []string
	var args []interface{}
	i := start

	if f.From != "" {
		conds = append(conds, fmt.Sprintf("order_date >= $%d", i))
		args = append(args, f.From)
		i++
	}
	if f.To != "" {
		conds = append(conds, fmt.Sprintf("order_date <= $%d", i))
		args = append(args, f.To)
		i++
	}
	if len(f.Regions) > 0 {
		conds = append(conds, fmt.Sprintf("region = ANY($%d)", i))
		args = append(args, pq.Array(f.Regions))
		i++
	}
	if len(f.Carriers) > 0 {
		conds = append(conds, fmt.Sprintf("carrier = ANY($%d)", i))
		args = append(args, pq.Array(f.Carriers))
		i++
	}
	if len(f.Categories) > 0 {
		conds = append(conds, fmt.Sprintf("product_category = ANY($%d)", i))
		args = append(args, pq.Array(f.Categories))
		i++
	}

	if len(conds) == 0 {
		return "", args, i
	}
	return "WHERE " + strings.Join(conds, " AND "), args, i
}

func (r *OrderRepo) Meta(ctx context.Context) (domain.Meta, error) {
	var m domain.Meta
	distinct := func(col string, dest *[]string) error {
		return r.db.SelectContext(ctx, dest,
			fmt.Sprintf("SELECT DISTINCT %s FROM orders ORDER BY %s", col, col))
	}
	if err := distinct("carrier", &m.Carriers); err != nil {
		return m, err
	}
	if err := distinct("region", &m.Regions); err != nil {
		return m, err
	}
	if err := distinct("product_category", &m.Categories); err != nil {
		return m, err
	}
	if err := distinct("status", &m.Statuses); err != nil {
		return m, err
	}
	if err := r.db.GetContext(ctx, &m,
		`SELECT to_char(MIN(order_date),'YYYY-MM-DD') AS "datemin",
		        to_char(MAX(order_date),'YYYY-MM-DD') AS "datemax" FROM orders`); err != nil {
		return m, err
	}
	return m, nil
}

func (r *OrderRepo) KPIs(ctx context.Context, f domain.Filters) (domain.KPIs, error) {
	where, args, _ := whereClause(f, 1)
	var row struct {
		Total     int     `db:"total"`
		Delivered int     `db:"delivered"`
		Delayed   int     `db:"delayed"`
		AvgDays   float64 `db:"avg_days"`
		Revenue   float64 `db:"revenue"`
	}
	q := fmt.Sprintf(`
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'delivered') AS delivered,
			COUNT(*) FILTER (WHERE status = 'delayed')   AS delayed,
			COALESCE(AVG(transit_days) FILTER (WHERE status = 'delivered'), 0) AS avg_days,
			COALESCE(SUM(order_value_usd), 0) AS revenue
		FROM orders %s`, where)
	if err := r.db.GetContext(ctx, &row, q, args...); err != nil {
		return domain.KPIs{}, err
	}
	settled := row.Delivered + row.Delayed
	rate := 0.0
	if settled > 0 {
		rate = float64(row.Delivered) / float64(settled)
	}
	return domain.KPIs{
		TotalOrders:     row.Total,
		DeliveredOrders: row.Delivered,
		DelayedOrders:   row.Delayed,
		OnTimeRate:      rate,
		AvgDeliveryDays: row.AvgDays,
		TotalRevenue:    row.Revenue,
	}, nil
}

func (r *OrderRepo) TimeSeries(ctx context.Context, f domain.Filters, granularity string) ([]domain.TimePoint, error) {
	trunc, format := "month", "YYYY-MM"
	if granularity == "week" {
		trunc, format = "week", "YYYY-MM-DD"
	}
	where, args, _ := whereClause(f, 1)
	q := fmt.Sprintf(`
		SELECT
			to_char(date_trunc('%s', order_date), '%s') AS bucket,
			COUNT(*) AS orders,
			COUNT(*) FILTER (WHERE status = 'delivered') AS delivered,
			COUNT(*) FILTER (WHERE status = 'delayed')   AS delayed,
			COALESCE(SUM(order_value_usd), 0) AS revenue
		FROM orders %s
		GROUP BY 1 ORDER BY 1`, trunc, format, where)
	var pts []domain.TimePoint
	if err := r.db.SelectContext(ctx, &pts, q, args...); err != nil {
		return nil, err
	}
	return pts, nil
}

func (r *OrderRepo) StatusMix(ctx context.Context, f domain.Filters) ([]domain.StatusCount, error) {
	where, args, _ := whereClause(f, 1)
	q := fmt.Sprintf(`SELECT status, COUNT(*) AS count FROM orders %s GROUP BY status ORDER BY count DESC`, where)
	var out []domain.StatusCount
	if err := r.db.SelectContext(ctx, &out, q, args...); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *OrderRepo) Breakdown(ctx context.Context, f domain.Filters, dimension string, limit int) ([]domain.BreakdownRow, error) {
	col, ok := dimensionColumns[dimension]
	if !ok {
		return nil, fmt.Errorf("unsupported dimension %q", dimension)
	}
	where, args, next := whereClause(f, 1)
	limitClause := ""
	if limit > 0 {
		limitClause = fmt.Sprintf("LIMIT $%d", next)
		args = append(args, limit)
	}
	q := fmt.Sprintf(`
		SELECT
			%s AS name,
			COUNT(*) AS orders,
			COUNT(*) FILTER (WHERE status = 'delivered') AS delivered,
			COUNT(*) FILTER (WHERE status = 'delayed')   AS delayed,
			COALESCE(AVG(transit_days) FILTER (WHERE status = 'delivered'), 0) AS avgdeliverydays,
			COALESCE(SUM(order_value_usd), 0) AS revenue
		FROM orders %s
		GROUP BY %s ORDER BY orders DESC %s`, col, where, col, limitClause)

	rows := []struct {
		Name            string  `db:"name"`
		Orders          int     `db:"orders"`
		Delivered       int     `db:"delivered"`
		Delayed         int     `db:"delayed"`
		AvgDeliveryDays float64 `db:"avgdeliverydays"`
		Revenue         float64 `db:"revenue"`
	}{}
	if err := r.db.SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, err
	}
	out := make([]domain.BreakdownRow, 0, len(rows))
	for _, r := range rows {
		settled := r.Delivered + r.Delayed
		rate := 0.0
		if settled > 0 {
			rate = float64(r.Delayed) / float64(settled)
		}
		out = append(out, domain.BreakdownRow{
			Name: r.Name, Orders: r.Orders, Delivered: r.Delivered, Delayed: r.Delayed,
			DelayRate: rate, AvgDeliveryDays: r.AvgDeliveryDays, Revenue: r.Revenue,
		})
	}
	return out, nil
}

func (r *OrderRepo) CategoryStack(ctx context.Context, f domain.Filters, topN int) (domain.CategoryStack, error) {
	top, err := r.Breakdown(ctx, f, "product_category", topN)
	if err != nil {
		return domain.CategoryStack{}, err
	}
	topNames := make([]string, 0, len(top))
	isTop := map[string]bool{}
	for _, t := range top {
		topNames = append(topNames, t.Name)
		isTop[t.Name] = true
	}

	where, args, _ := whereClause(f, 1)
	q := fmt.Sprintf(`
		SELECT to_char(date_trunc('month', order_date), 'YYYY-MM') AS bucket,
		       product_category AS cat, COUNT(*) AS c
		FROM orders %s GROUP BY 1, 2 ORDER BY 1`, where)
	rows := []struct {
		Bucket string `db:"bucket"`
		Cat    string `db:"cat"`
		C      int    `db:"c"`
	}{}
	if err := r.db.SelectContext(ctx, &rows, q, args...); err != nil {
		return domain.CategoryStack{}, err
	}

	keys := append([]string{}, topNames...)
	hasOther := false
	// Pivot monthly rows into one map per bucket.
	byBucket := map[string]map[string]interface{}{}
	order := []string{}
	for _, row := range rows {
		m, ok := byBucket[row.Bucket]
		if !ok {
			m = map[string]interface{}{"bucket": row.Bucket}
			for _, k := range topNames {
				m[k] = 0
			}
			byBucket[row.Bucket] = m
			order = append(order, row.Bucket)
		}
		key := row.Cat
		if !isTop[key] {
			key = "Other"
			hasOther = true
		}
		if cur, ok := m[key].(int); ok {
			m[key] = cur + row.C
		} else {
			m[key] = row.C
		}
	}
	if hasOther {
		keys = append(keys, "Other")
	}
	data := make([]map[string]interface{}, 0, len(order))
	for _, b := range order {
		data = append(data, byBucket[b])
	}
	return domain.CategoryStack{Keys: keys, Data: data}, nil
}

func (r *OrderRepo) Orders(ctx context.Context, q domain.OrderQuery) (domain.OrderPage, error) {
	where, args, next := whereClause(q.Filters, 1)
	conds := []string{}
	if where != "" {
		conds = append(conds, strings.TrimPrefix(where, "WHERE "))
	}
	if q.Status != "" && q.Status != "all" {
		conds = append(conds, fmt.Sprintf("status = $%d", next))
		args = append(args, q.Status)
		next++
	}
	if q.Search != "" {
		conds = append(conds, fmt.Sprintf(
			"(order_id ILIKE $%d OR sku ILIKE $%d OR client_id ILIKE $%d OR destination_city ILIKE $%d OR carrier ILIKE $%d)",
			next, next, next, next, next))
		args = append(args, "%"+q.Search+"%")
		next++
	}
	whereSQL := ""
	if len(conds) > 0 {
		whereSQL = "WHERE " + strings.Join(conds, " AND ")
	}

	var page domain.OrderPage
	page.Page, page.PageSize = q.Page, q.PageSize
	if err := r.db.GetContext(ctx, &page.Total,
		fmt.Sprintf("SELECT COUNT(*) FROM orders %s", whereSQL), args...); err != nil {
		return page, err
	}

	sortCol, ok := sortColumns[q.SortKey]
	if !ok {
		sortCol = "order_date"
	}
	dir := "DESC"
	if strings.EqualFold(q.SortDir, "asc") {
		dir = "ASC"
	}
	args = append(args, q.PageSize, q.Page*q.PageSize)
	list := fmt.Sprintf(`SELECT %s FROM orders %s ORDER BY %s %s NULLS LAST LIMIT $%d OFFSET $%d`,
		orderColumns, whereSQL, sortCol, dir, next, next+1)
	if err := r.db.SelectContext(ctx, &page.Rows, list, args...); err != nil {
		return page, err
	}
	return page, nil
}

const orderInsertCols = `client_id, order_id, order_date, delivery_date, carrier, origin_city,
	destination_city, status, sku, product_category, quantity, unit_price_usd, order_value_usd,
	is_promo, promo_discount_pct, region, warehouse, transit_days`

func orderArgs(o domain.Order) []interface{} {
	return []interface{}{
		o.ClientID, o.OrderID, o.OrderDate, o.DeliveryDate, o.Carrier, o.OriginCity, o.DestinationCity,
		o.Status, o.SKU, o.Category, o.Quantity, o.UnitPrice, o.OrderValue, o.IsPromo,
		o.PromoDiscount, o.Region, o.Warehouse, o.TransitDays,
	}
}

func (r *OrderRepo) GetOrder(ctx context.Context, orderID string) (domain.Order, bool, error) {
	var o domain.Order
	err := r.db.GetContext(ctx, &o,
		fmt.Sprintf("SELECT %s FROM orders WHERE order_id = $1", orderColumns), orderID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Order{}, false, nil
	}
	return o, err == nil, err
}

func (r *OrderRepo) CreateOrder(ctx context.Context, o domain.Order) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(
		"INSERT INTO orders (%s) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)",
		orderInsertCols), orderArgs(o)...)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" { // unique_violation
		return domain.ErrConflict
	}
	return err
}

func (r *OrderRepo) UpdateOrder(ctx context.Context, o domain.Order) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders SET
			client_id=$1, order_date=$3, delivery_date=$4, carrier=$5, origin_city=$6,
			destination_city=$7, status=$8, sku=$9, product_category=$10, quantity=$11,
			unit_price_usd=$12, order_value_usd=$13, is_promo=$14, promo_discount_pct=$15,
			region=$16, warehouse=$17, transit_days=$18
		WHERE order_id=$2`, orderArgs(o)...)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (r *OrderRepo) DeleteOrder(ctx context.Context, orderID string) (bool, error) {
	res, err := r.db.ExecContext(ctx, "DELETE FROM orders WHERE order_id = $1", orderID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ImportOrders writes orders in a single transaction, upserting by order_id.
// When replace is true the table is truncated first (a clean re-init).
func (r *OrderRepo) ImportOrders(ctx context.Context, orders []domain.Order, replace bool) (int, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() //nolint: errcheck — no-op after a successful commit

	if replace {
		if _, err := tx.ExecContext(ctx, "TRUNCATE orders"); err != nil {
			return 0, err
		}
	}

	const upsert = `
		INSERT INTO orders
			(client_id, order_id, order_date, delivery_date, carrier, origin_city, destination_city,
			 status, sku, product_category, quantity, unit_price_usd, order_value_usd, is_promo,
			 promo_discount_pct, region, warehouse, transit_days)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		ON CONFLICT (order_id) DO UPDATE SET
			client_id=EXCLUDED.client_id, order_date=EXCLUDED.order_date,
			delivery_date=EXCLUDED.delivery_date, carrier=EXCLUDED.carrier,
			origin_city=EXCLUDED.origin_city, destination_city=EXCLUDED.destination_city,
			status=EXCLUDED.status, sku=EXCLUDED.sku, product_category=EXCLUDED.product_category,
			quantity=EXCLUDED.quantity, unit_price_usd=EXCLUDED.unit_price_usd,
			order_value_usd=EXCLUDED.order_value_usd, is_promo=EXCLUDED.is_promo,
			promo_discount_pct=EXCLUDED.promo_discount_pct, region=EXCLUDED.region,
			warehouse=EXCLUDED.warehouse, transit_days=EXCLUDED.transit_days`
	stmt, err := tx.PreparexContext(ctx, upsert)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	count := 0
	for _, o := range orders {
		if _, err := stmt.ExecContext(ctx,
			o.ClientID, o.OrderID, o.OrderDate, o.DeliveryDate, o.Carrier, o.OriginCity, o.DestinationCity,
			o.Status, o.SKU, o.Category, o.Quantity, o.UnitPrice, o.OrderValue, o.IsPromo,
			o.PromoDiscount, o.Region, o.Warehouse, o.TransitDays,
		); err != nil {
			return 0, fmt.Errorf("insert order %s: %w", o.OrderID, err)
		}
		count++
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *OrderRepo) MonthlyUnits(ctx context.Context, category string) ([]domain.MonthUnits, error) {
	q := `SELECT to_char(date_trunc('month', order_date), 'YYYY-MM') AS bucket, SUM(quantity) AS units
	      FROM orders`
	var args []interface{}
	if category != "" {
		q += " WHERE product_category = $1"
		args = append(args, category)
	}
	q += " GROUP BY 1 ORDER BY 1"
	var out []domain.MonthUnits
	if err := r.db.SelectContext(ctx, &out, q, args...); err != nil {
		return nil, err
	}
	return out, nil
}
