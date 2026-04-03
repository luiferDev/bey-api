package orders

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"

	"bey/internal/shared/response"
)

type VariantStockHandler interface {
	ReserveStock(id uuid.UUID, quantity int) error
	ConfirmSale(id uuid.UUID, quantity int) error
	ReleaseStock(id uuid.UUID, quantity int) error
}

type InventoryHandler interface {
	Reserve(productID uuid.UUID, quantity int) error
	Release(productID uuid.UUID, quantity int) error
}

type OrderHandler struct {
	repo         *OrderRepository
	orderService *OrderService
	productRepo  interface {
		GetPriceByID(id uuid.UUID) (float64, error)
	}
	variantRepo   VariantStockHandler
	inventoryRepo InventoryHandler
	resp          *response.ResponseHandler
}

func NewOrderHandler(db *gorm.DB) *OrderHandler {
	return &OrderHandler{
		repo: NewOrderRepository(db),
		resp: response.NewResponseHandler(),
	}
}

func NewOrderHandlerWithAllDeps(db *gorm.DB, orderService *OrderService, productRepo interface {
	GetPriceByID(id uuid.UUID) (float64, error)
}, variantRepo VariantStockHandler, inventoryRepo InventoryHandler) *OrderHandler {
	return &OrderHandler{
		repo:          NewOrderRepository(db),
		orderService:  orderService,
		productRepo:   productRepo,
		variantRepo:   variantRepo,
		inventoryRepo: inventoryRepo,
		resp:          response.NewResponseHandler(),
	}
}

func (h *OrderHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.resp.ValidationError(c, err.Error())
		return
	}

	if h.orderService != nil {
		taskID, err := h.orderService.SubmitAsyncOrder(req, userID)
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
			productID, parseErr := uuid.FromString(item.ProductID)
			if parseErr != nil {
				h.resp.ValidationError(c, "invalid product_id format")
				return
			}
			price, priceErr := h.productRepo.GetPriceByID(productID)
			if priceErr != nil {
				h.resp.InternalError(c, "failed to get product price")
				return
			}
			if price == 0 {
				h.resp.ValidationError(c, "product not found")
				return
			}
			unitPrice = price
		}

		orderItem := OrderItem{
			Quantity:  item.Quantity,
			UnitPrice: unitPrice,
		}
		if item.VariantID != nil && *item.VariantID != "" {
			variantID, parseErr := uuid.FromString(*item.VariantID)
			if parseErr != nil {
				h.resp.ValidationError(c, "invalid variant_id format")
				return
			}
			orderItem.VariantID = &variantID
		}

		items[i] = orderItem
		totalPrice += float64(item.Quantity) * unitPrice
	}

	order := &Order{
		UserID:          uuid.Nil,
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

	for _, item := range order.Items {
		if item.VariantID != nil && h.variantRepo != nil {
			if err := h.variantRepo.ReserveStock(*item.VariantID, item.Quantity); err != nil {
				log.Printf("Warning: failed to reserve variant stock for variant %s: %v", *item.VariantID, err)
			}
		} else if h.inventoryRepo != nil {
			if err := h.inventoryRepo.Reserve(item.ProductID, item.Quantity); err != nil {
				log.Printf("Warning: failed to reserve inventory for product %s: %v", item.ProductID, err)
			}
		}
	}

	h.resp.Created(c, toOrderResponse(order))
}

func (h *OrderHandler) GetByID(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.resp.ValidationError(c, "invalid order_id format")
		return
	}

	order, err := h.repo.FindByID(id)
	if err != nil {
		h.resp.InternalError(c, "failed to get order")
		return
	}
	if order == nil {
		h.resp.NotFound(c, "order not found")
		return
	}

	currentUserID := c.GetString("user_id")
	userRole := c.GetString("user_role")

	if userRole != "admin" && order.UserID.String() != currentUserID {
		h.resp.Error(c, 403, "you can only view your own orders")
		return
	}

	h.resp.Success(c, toOrderResponse(order))
}

func (h *OrderHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.resp.ValidationError(c, "invalid order_id format")
		return
	}

	order, err := h.repo.FindByID(id)
	if err != nil {
		h.resp.InternalError(c, "failed to get order")
		return
	}
	if order == nil {
		h.resp.NotFound(c, "order not found")
		return
	}

	userID := c.GetString("user_id")
	userRole := c.GetString("user_role")
	if userRole != "admin" && order.UserID.String() != userID {
		h.resp.Error(c, 403, "you don't have access to this order")
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
	userID := c.GetUint("user_id")
	userRole := c.GetString("user_role")

	var orders []Order
	var err error
	if userRole == "admin" {
		orders, err = h.repo.FindAll(0, 100)
	} else {
		userUUID, parseErr := uuid.FromString(fmt.Sprintf("%d", userID))
		if parseErr != nil {
			h.resp.InternalError(c, "invalid user ID")
			return
		}
		orders, err = h.repo.FindByUserID(userUUID)
	}
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

	userRole := c.GetString("user_role")
	errorMsg := task.Error
	if errorMsg != "" && userRole != "admin" {
		errorMsg = "An error occurred while processing your order"
	}

	h.resp.Success(c, gin.H{
		"task_id":    task.ID,
		"type":       task.Type,
		"status":     task.Status,
		"result":     task.Result,
		"error":      errorMsg,
		"created_at": task.CreatedAt,
		"updated_at": task.UpdatedAt,
	})
}

func (h *OrderHandler) Confirm(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.resp.ValidationError(c, "invalid order_id format")
		return
	}

	order, err := h.repo.FindByID(id)
	if err != nil {
		h.resp.InternalError(c, "failed to get order")
		return
	}
	if order == nil {
		h.resp.NotFound(c, "order not found")
		return
	}

	currentUserID := c.GetString("user_id")
	userRole := c.GetString("user_role")

	if userRole != "admin" && order.UserID.String() != currentUserID {
		h.resp.Error(c, 403, "you can only confirm your own orders")
		return
	}

	if order.Status != "pending" && order.Status != "confirmed" {
		h.resp.Error(c, 400, "order cannot be confirmed in current status")
		return
	}

	for _, item := range order.Items {
		if item.VariantID != nil && h.variantRepo != nil {
			if err := h.variantRepo.ConfirmSale(*item.VariantID, item.Quantity); err != nil {
				h.resp.InternalError(c, "failed to confirm variant stock")
				return
			}
		} else if h.inventoryRepo != nil {
			if err := h.inventoryRepo.Release(item.ProductID, item.Quantity); err != nil {
				log.Printf("Warning: failed to confirm inventory for product %s: %v", item.ProductID, err)
			}
		}
	}

	order.Status = "confirmed"
	if err := h.repo.Update(order); err != nil {
		h.resp.InternalError(c, "failed to update order")
		return
	}

	h.resp.Success(c, toOrderResponse(order))
}

func (h *OrderHandler) Cancel(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.resp.ValidationError(c, "invalid order_id format")
		return
	}

	order, err := h.repo.FindByID(id)
	if err != nil {
		h.resp.InternalError(c, "failed to get order")
		return
	}
	if order == nil {
		h.resp.NotFound(c, "order not found")
		return
	}

	currentUserID := c.GetString("user_id")
	userRole := c.GetString("user_role")

	if userRole != "admin" && order.UserID.String() != currentUserID {
		h.resp.Error(c, 403, "you can only cancel your own orders")
		return
	}

	if order.Status != "pending" {
		h.resp.Error(c, 400, "order cannot be cancelled in current status")
		return
	}

	if h.variantRepo != nil {
		for _, item := range order.Items {
			if item.VariantID != nil {
				if err := h.variantRepo.ReleaseStock(*item.VariantID, item.Quantity); err != nil {
					h.resp.InternalError(c, "failed to release variant stock")
					return
				}
			}
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
		itemResp := OrderItemResponse{
			ID:        order.Items[i].ID.String(),
			ProductID: order.Items[i].ProductID.String(),
			Quantity:  order.Items[i].Quantity,
			UnitPrice: order.Items[i].UnitPrice,
		}
		if order.Items[i].VariantID != nil {
			s := order.Items[i].VariantID.String()
			itemResp.VariantID = &s
		}
		items[i] = itemResp
	}
	return OrderResponse{
		ID:              order.ID.String(),
		UserID:          order.UserID.String(),
		Status:          order.Status,
		TotalPrice:      order.TotalPrice,
		ShippingAddress: order.ShippingAddress,
		Notes:           order.Notes,
		Items:           items,
		CreatedAt:       order.CreatedAt,
	}
}
