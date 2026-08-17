package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"analytics-dashboard-be/internal/domain"
)

// orderInput is the create/update payload. Dates are plain YYYY-MM-DD strings so
// the frontend can use <input type="date"> directly; transit_days is derived.
type orderInput struct {
	ClientID         string  `json:"clientId"`
	OrderID          string  `json:"orderId"`
	OrderDate        string  `json:"orderDate"`
	DeliveryDate     string  `json:"deliveryDate"`
	Carrier          string  `json:"carrier"`
	OriginCity       string  `json:"originCity"`
	DestinationCity  string  `json:"destinationCity"`
	Status           string  `json:"status"`
	SKU              string  `json:"sku"`
	Category         string  `json:"category"`
	Quantity         int     `json:"quantity"`
	UnitPrice        float64 `json:"unitPrice"`
	OrderValue       float64 `json:"orderValue"`
	IsPromo          bool    `json:"isPromo"`
	PromoDiscountPct float64 `json:"promoDiscountPct"`
	Region           string  `json:"region"`
	Warehouse        string  `json:"warehouse"`
}

func (in orderInput) toOrder() (domain.Order, error) {
	od, err := time.Parse("2006-01-02", strings.TrimSpace(in.OrderDate))
	if err != nil {
		return domain.Order{}, domain.NewAPIError(http.StatusBadRequest, "VALIDATION_ERROR", "orderDate must be YYYY-MM-DD")
	}
	o := domain.Order{
		ClientID: in.ClientID, OrderID: strings.TrimSpace(in.OrderID), OrderDate: od,
		Carrier: in.Carrier, OriginCity: in.OriginCity, DestinationCity: in.DestinationCity,
		Status: domain.OrderStatus(in.Status), SKU: in.SKU, Category: in.Category,
		Quantity: in.Quantity, UnitPrice: in.UnitPrice, OrderValue: in.OrderValue,
		IsPromo: in.IsPromo, PromoDiscount: in.PromoDiscountPct, Region: in.Region, Warehouse: in.Warehouse,
	}
	if dd := strings.TrimSpace(in.DeliveryDate); dd != "" {
		parsed, err := time.Parse("2006-01-02", dd)
		if err != nil {
			return domain.Order{}, domain.NewAPIError(http.StatusBadRequest, "VALIDATION_ERROR", "deliveryDate must be YYYY-MM-DD")
		}
		o.DeliveryDate = &parsed
		days := int(parsed.Sub(od).Hours() / 24)
		o.TransitDays = &days
	}
	return o, nil
}

func (h *Handler) GetOrder(c *gin.Context) {
	o, err := h.analytics.GetOrder(c.Request.Context(), c.Param("orderId"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, o)
}

func (h *Handler) CreateOrder(c *gin.Context) {
	var in orderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, domain.ErrValidation)
		return
	}
	o, err := in.toOrder()
	if err != nil {
		fail(c, err)
		return
	}
	created, err := h.analytics.CreateOrder(c.Request.Context(), o)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) UpdateOrder(c *gin.Context) {
	var in orderInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, domain.ErrValidation)
		return
	}
	o, err := in.toOrder()
	if err != nil {
		fail(c, err)
		return
	}
	o.OrderID = c.Param("orderId") // the path is authoritative
	updated, err := h.analytics.UpdateOrder(c.Request.Context(), o)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) DeleteOrder(c *gin.Context) {
	if err := h.analytics.DeleteOrder(c.Request.Context(), c.Param("orderId")); err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": c.Param("orderId")})
}
