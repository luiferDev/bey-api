package orders

import (
	"bey/internal/shared/response"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type VariantStockHandler interface {
	ConfirmSale(id uint, quantity int) error
	ReleaseStock(id uint, quantity int) error
}

type OrderHandler struct {
	repo         *OrderRepository
	orderService *OrderService
	productRepo  interface {
		GetPriceByID(id uint) (float64, error)
	}
	variantRepo VariantStockHandler
	resp        *response.ResponseHandler
}

func NewOrderHandler(db *gorm.DB) *OrderHandler {
	return &OrderHandler{
		repo: NewOrderRepository(db),
		resp: response.NewResponseHandler(),
	}
}

func NewOrderHandlerWithAllDeps(db *gorm.DB, orderService *OrderService, productRepo interface {
	GetPriceByID(id uint) (float64, error)
}, variantRepo VariantStockHandler) *OrderHandler {
	return &OrderHandler{
		repo:         NewOrderRepository(db),
		orderService: orderService,
		productRepo:  productRepo,
		variantRepo:  variantRepo,
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
		unitPrice := float64(0)
		if h.productRepo != nil {
			price, err := h.productRepo.GetPriceByID(item.ProductID)
			if err != nil {
				h.resp.InternalError(c, "failed to get product price")
				return
			}
			if price == 0 {
				h.resp.ValidationError(c, "product not found")
				return
			}
			unitPrice = price
		}

		items[i] = OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: unitPrice,
		}
		totalPrice += float64(item.Quantity) * unitPrice
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
// @Description Retrieves an order by its ID (owner or admin)
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path int true "Order ID"
// @Success 200 {object} OrderResponse
// @Router /api/v1/orders/{id} [get]
func (h *OrderHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.resp.ValidationError(c, "invalid order_id")
		return
	}

	order, err := h.repo.FindByID(uint(id))
	if err != nil {
		h.resp.InternalError(c, "failed to get order")
		return
	}
	if order == nil {
		h.resp.NotFound(c, "order not found")
		return
	}

	// Check if user is owner or admin
	currentUserID := c.GetUint("user_id")
	userRole := c.GetString("user_role")

	if userRole != "admin" && order.UserID != currentUserID {
		h.resp.Error(c, 403, "you can only view your own orders")
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
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.resp.ValidationError(c, "invalid order_id")
		return
	}

	order, err := h.repo.FindByID(uint(id))
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

// @Summary Confirm order sale
// @Description Confirms a sale after payment (finalizes the reservation)
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path int true "Order ID"
// @Success 200 {object} OrderResponse
// @Router /api/v1/orders/{id}/confirm [post]
func (h *OrderHandler) Confirm(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.resp.ValidationError(c, "invalid order_id")
		return
	}

	order, err := h.repo.FindByID(uint(id))
	if err != nil {
		h.resp.InternalError(c, "failed to get order")
		return
	}
	if order == nil {
		h.resp.NotFound(c, "order not found")
		return
	}

	// Check if user is owner or admin
	currentUserID := c.GetUint("user_id")
	userRole := c.GetString("user_role")

	if userRole != "admin" && order.UserID != currentUserID {
		h.resp.Error(c, 403, "you can only confirm your own orders")
		return
	}

	// Only confirm pending or confirmed orders
	if order.Status != "pending" && order.Status != "confirmed" {
		h.resp.Error(c, 400, "order cannot be confirmed in current status")
		return
	}

	// Confirm stock for each item
	if h.variantRepo != nil {
		for _, item := range order.Items {
			if item.VariantID != nil {
				// Confirm variant sale (just reduces reserved)
				if err := h.variantRepo.ConfirmSale(*item.VariantID, item.Quantity); err != nil {
					h.resp.InternalError(c, "failed to confirm variant stock")
					return
				}
			}
			// TODO: Also confirm inventory if no variant
		}
	}

	order.Status = "confirmed"
	if err := h.repo.Update(order); err != nil {
		h.resp.InternalError(c, "failed to update order")
		return
	}

	h.resp.Success(c, toOrderResponse(order))
}

// @Summary Cancel order
// @Description Cancels an order and releases reserved inventory
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path int true "Order ID"
// @Success 200 {object} OrderResponse
// @Router /api/v1/orders/{id}/cancel [post]
func (h *OrderHandler) Cancel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.resp.ValidationError(c, "invalid order_id")
		return
	}

	order, err := h.repo.FindByID(uint(id))
	if err != nil {
		h.resp.InternalError(c, "failed to get order")
		return
	}
	if order == nil {
		h.resp.NotFound(c, "order not found")
		return
	}

	// Check if user is owner or admin
	currentUserID := c.GetUint("user_id")
	userRole := c.GetString("user_role")

	if userRole != "admin" && order.UserID != currentUserID {
		h.resp.Error(c, 403, "you can only cancel your own orders")
		return
	}

	// Only cancel pending orders
	if order.Status != "pending" {
		h.resp.Error(c, 400, "order cannot be cancelled in current status")
		return
	}

	// Release stock for each item
	if h.variantRepo != nil {
		for _, item := range order.Items {
			if item.VariantID != nil {
				// Release variant stock (returns to stock, reduces reserved)
				if err := h.variantRepo.ReleaseStock(*item.VariantID, item.Quantity); err != nil {
					h.resp.InternalError(c, "failed to release variant stock")
					return
				}
			}
			// TODO: Also release inventory if no variant
		}
	}

	order.Status = "cancelled"
	if err := h.repo.Update(order); err != nil {
		h.resp.InternalError(c, "failed to update order")
		return
	}

	h.resp.Success(c, toOrderResponse(order))
}

func toOrderResponse(order *Order) OrderResponse {
	items := make([]OrderItemResponse, len(order.Items))
	for i := range order.Items {
		items[i] = OrderItemResponse{
			ID:        order.Items[i].ID,
			ProductID: order.Items[i].ProductID,
			VariantID: order.Items[i].VariantID,
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
