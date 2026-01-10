package orders

import (
	"bey/internal/shared/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderHandler struct {
	repo *OrderRepository
	resp *response.ResponseHandler
}

func NewOrderHandler(db *gorm.DB) *OrderHandler {
	return &OrderHandler{
		repo: NewOrderRepository(db),
		resp: response.NewResponseHandler(),
	}
}

func (h *OrderHandler) Create(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.resp.ValidationError(c, err.Error())
		return
	}

	var totalPrice float64
	items := make([]OrderItem, len(req.Items))
	for i, item := range req.Items {
		items[i] = OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: 0,
		}
		totalPrice += float64(item.Quantity) * items[i].UnitPrice
	}

	order := &Order{
		UserID:          req.UserID,
		Status:          "pending",
		TotalPrice:      totalPrice,
		ShippingAddress: req.ShippingAddress,
		Notes:           req.Notes,
		Items:           items,
	}

	if err := h.repo.Create(order); err != nil {
		h.resp.InternalError(c, "failed to create order")
		return
	}

	h.resp.Created(c, toOrderResponse(order))
}

func (h *OrderHandler) GetByID(c *gin.Context) {
	id := c.GetUint("order_id")
	order, err := h.repo.FindByID(id)
	if err != nil {
		h.resp.InternalError(c, "failed to get order")
		return
	}
	if order == nil {
		h.resp.NotFound(c, "order not found")
		return
	}
	h.resp.Success(c, toOrderResponse(order))
}

func (h *OrderHandler) UpdateStatus(c *gin.Context) {
	id := c.GetUint("order_id")
	order, err := h.repo.FindByID(id)
	if err != nil {
		h.resp.InternalError(c, "failed to get order")
		return
	}
	if order == nil {
		h.resp.NotFound(c, "order not found")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.resp.ValidationError(c, err.Error())
		return
	}

	order.Status = req.Status
	if err := h.repo.Update(order); err != nil {
		h.resp.InternalError(c, "failed to update order")
		return
	}

	h.resp.Success(c, toOrderResponse(order))
}

func (h *OrderHandler) List(c *gin.Context) {
	orders, err := h.repo.FindAll(0, 100)
	if err != nil {
		h.resp.InternalError(c, "failed to list orders")
		return
	}

	responses := make([]OrderResponse, len(orders))
	for i := range orders {
		responses[i] = toOrderResponse(&orders[i])
	}
	h.resp.Success(c, responses)
}

func toOrderResponse(order *Order) OrderResponse {
	items := make([]OrderItemResponse, len(order.Items))
	for i := range order.Items {
		items[i] = OrderItemResponse{
			ID:        order.Items[i].ID,
			ProductID: order.Items[i].ProductID,
			Quantity:  order.Items[i].Quantity,
			UnitPrice: order.Items[i].UnitPrice,
		}
	}
	return OrderResponse{
		ID:              order.ID,
		UserID:          order.UserID,
		Status:          order.Status,
		TotalPrice:      order.TotalPrice,
		ShippingAddress: order.ShippingAddress,
		Notes:           order.Notes,
		Items:           items,
		CreatedAt:       order.CreatedAt,
	}
}
