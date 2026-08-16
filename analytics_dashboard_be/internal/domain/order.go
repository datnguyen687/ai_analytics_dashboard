package domain

import "time"

// OrderStatus enumerates the delivery lifecycle states present in the dataset.
type OrderStatus string

const (
	StatusDelivered OrderStatus = "delivered"
	StatusDelayed   OrderStatus = "delayed"
	StatusInTransit OrderStatus = "in_transit"
	StatusException OrderStatus = "exception"
	StatusCanceled  OrderStatus = "canceled"
)

// Order is the core entity: one row of the logistics dataset. Treated read-only.
type Order struct {
	ClientID        string      `db:"client_id" json:"clientId"`
	OrderID         string      `db:"order_id" json:"orderId"`
	OrderDate       time.Time   `db:"order_date" json:"orderDate"`
	DeliveryDate    *time.Time  `db:"delivery_date" json:"deliveryDate"`
	Carrier         string      `db:"carrier" json:"carrier"`
	OriginCity      string      `db:"origin_city" json:"originCity"`
	DestinationCity string      `db:"destination_city" json:"destinationCity"`
	Status          OrderStatus `db:"status" json:"status"`
	SKU             string      `db:"sku" json:"sku"`
	Category        string      `db:"product_category" json:"category"`
	Quantity        int         `db:"quantity" json:"quantity"`
	UnitPrice       float64     `db:"unit_price_usd" json:"unitPrice"`
	OrderValue      float64     `db:"order_value_usd" json:"orderValue"`
	IsPromo         bool        `db:"is_promo" json:"isPromo"`
	PromoDiscount   float64     `db:"promo_discount_pct" json:"promoDiscountPct"`
	Region          string      `db:"region" json:"region"`
	Warehouse       string      `db:"warehouse" json:"warehouse"`
	TransitDays     *int        `db:"transit_days" json:"transitDays"`
}
