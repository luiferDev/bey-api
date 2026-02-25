package orders

import (
	"bey/internal/shared/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderHandler struct {
	repo         *OrderRepository
	orderService *OrderService
	resp         *response.ResponseHandler
}

func NewOrderHandler(db *gorm.DB) *OrderHandler {
	return &OrderHandler{
		repo: NewOrderRepository(db),
		resp: response.NewResponseHandler(),
	}
}

func NewOrderHandlerWithService(db *gorm.DB, orderService *OrderService) *OrderHandler {
	return &OrderHandler{
		repo:         NewOrderRepository(db),
		orderService: orderService,
		resp:         response.NewResponseHandler(),
	}
}

// @Summary Create a new order
// @Description Creates a new order (async or sync based on configuration)
// @Tags Orders
// @Accept json
// @Produce json
// @Param order body CreateOrderRequest true "Order data"
// @Success 201 {object} OrderResponse
// @Success 202
// @Router /api/v1/orders [post]
func (h *OrderHandler) Create(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.resp.ValidationError(c, err.Error())
		return
	}

	if h.orderService != nil {
		taskID, err := h.orderService.SubmitAsyncOrder(req)
		if err != nil {
			h.resp.InternalError(c, "failed to submit order task")
			return
		}
		c.JSON(202, gin.H{
			"message": "Order submitted for processing",
			"task_id": taskID,
			"status":  "pending",
		})
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

// @Summary Get order by ID
// @Description Retrieves an order by its ID
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path int true "Order ID"
// @Success 200 {object} OrderResponse
// @Router /api/v1/orders/{id} [get]
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

// @Summary Update order status
// @Description Updates the status of an existing order
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path int true "Order ID"
// @Param status body object true "Order status"
// @Success 200 {object} OrderResponse
// @Router /api/v1/orders/{id}/status [put]
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

// @Summary List all orders
// @Description Retrieves a list of all orders
// @Tags Orders
// @Accept json
// @Produce json
// @Success 200 {array} OrderResponse
// @Router /api/v1/orders [get]
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

// @Summary Get task status
// @Description Retrieves the status of an async order processing task
// @Tags Orders
// @Accept json
// @Produce json
// @Param task_id path string true "Task ID"
// @Success 200
// @Router /api/v1/orders/tasks/{task_id} [get]
func (h *OrderHandler) GetTaskStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		h.resp.ValidationError(c, "task_id is required")
		return
	}

	if h.orderService == nil {
		h.resp.InternalError(c, "task service not configured")
		return
	}

	task, err := h.orderService.GetTaskStatus(taskID)
	if err != nil {
		h.resp.InternalError(c, "failed to get task status")
		return
	}
	if task == nil {
		h.resp.NotFound(c, "task not found")
		return
	}

	h.resp.Success(c, gin.H{
		"task_id":    task.ID,
		"type":       task.Type,
		"status":     task.Status,
		"result":     task.Result,
		"error":      task.Error,
		"created_at": task.CreatedAt,
		"updated_at": task.UpdatedAt,
	})
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
