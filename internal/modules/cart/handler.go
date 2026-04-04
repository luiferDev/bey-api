package cart

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"

	"bey/internal/modules/orders"
	"bey/internal/shared/response"
)

type OrderCreator interface {
	Create(order *orders.Order) error
}

type VariantStockReserver interface {
	ReserveStock(id uuid.UUID, quantity int) error
}

type InventoryReserver interface {
	Reserve(productID uuid.UUID, quantity int) error
}

type CartHandler struct {
	cartService   *CartService
	orderRepo     OrderCreator
	variantRepo   VariantStockReserver
	inventoryRepo InventoryReserver
	response      *response.ResponseHandler
}

func NewCartHandler(cartService *CartService, orderRepo OrderCreator, variantRepo VariantStockReserver, inventoryRepo InventoryReserver) *CartHandler {
	return &CartHandler{
		cartService:   cartService,
		orderRepo:     orderRepo,
		variantRepo:   variantRepo,
		inventoryRepo: inventoryRepo,
		response:      response.NewResponseHandler(),
	}
}

func (h *CartHandler) GetCart(c *gin.Context) {
	userID, err := h.parseUserID(c)
	if err != nil {
		h.response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	cart, err := h.cartService.GetCart(userID)
	if err != nil {
		h.response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	h.response.Success(c, ToCartResponse(cart))
}

func (h *CartHandler) AddItem(c *gin.Context) {
	userID, err := h.parseUserID(c)
	if err != nil {
		h.response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req AddToCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationError(c, err.Error())
		return
	}

	variantID, parseErr := uuid.FromString(req.VariantID)
	if parseErr != nil {
		h.response.ValidationError(c, "invalid variant_id format")
		return
	}

	cart, err := h.cartService.AddItem(userID, variantID, req.Quantity)
	if err != nil {
		if errors.Is(err, ErrInsufficientStock) {
			h.response.Error(c, http.StatusBadRequest, "insufficient stock")
			return
		}
		if errors.Is(err, ErrVariantNotFound) {
			h.response.Error(c, http.StatusNotFound, "variant not found")
			return
		}
		h.response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	h.response.Success(c, ToCartResponse(cart))
}

func (h *CartHandler) UpdateItem(c *gin.Context) {
	userID, err := h.parseUserID(c)
	if err != nil {
		h.response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	variantID, err := parseUUIDParam(c, "variant_id")
	if err != nil {
		h.response.ValidationError(c, "invalid variant_id format")
		return
	}

	var req UpdateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationError(c, err.Error())
		return
	}

	cart, err := h.cartService.UpdateQuantity(userID, variantID, req.Quantity)
	if err != nil {
		if errors.Is(err, ErrInsufficientStock) {
			h.response.Error(c, http.StatusBadRequest, "insufficient stock")
			return
		}
		if errors.Is(err, ErrVariantNotFound) {
			h.response.Error(c, http.StatusNotFound, "item not found in cart")
			return
		}
		h.response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	h.response.Success(c, ToCartResponse(cart))
}

func (h *CartHandler) RemoveItem(c *gin.Context) {
	userID, err := h.parseUserID(c)
	if err != nil {
		h.response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	variantID, err := parseUUIDParam(c, "variant_id")
	if err != nil {
		h.response.ValidationError(c, "invalid variant_id format")
		return
	}

	cart, err := h.cartService.RemoveItem(userID, variantID)
	if err != nil {
		h.response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	h.response.Success(c, ToCartResponse(cart))
}

func (h *CartHandler) ClearCart(c *gin.Context) {
	userID, err := h.parseUserID(c)
	if err != nil {
		h.response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.cartService.ClearCart(userID); err != nil {
		h.response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	h.response.Success(c, gin.H{"message": "cart cleared"})
}

func (h *CartHandler) Checkout(c *gin.Context) {
	userID, err := h.parseUserID(c)
	if err != nil {
		h.response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationError(c, err.Error())
		return
	}

	result, err := h.cartService.PrepareCheckout(userID, req.ShippingAddress, req.Notes)
	if err != nil {
		if errors.Is(err, ErrCartEmpty) {
			h.response.Error(c, http.StatusBadRequest, "cart is empty")
			return
		}
		if errors.Is(err, ErrInsufficientStock) {
			h.response.Error(c, http.StatusBadRequest, "insufficient stock for one or more items")
			return
		}
		if errors.Is(err, ErrVariantNotFound) {
			h.response.Error(c, http.StatusNotFound, "one or more variants no longer exist")
			return
		}
		h.response.Error(c, http.StatusInternalServerError, "failed to process cart")
		return
	}

	orderItems := make([]orders.OrderItem, len(result.Items))
	for i, item := range result.Items {
		orderItems[i] = orders.OrderItem{
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
		}
	}

	order := &orders.Order{
		UserID:          result.UserID,
		Status:          "pending",
		TotalPrice:      result.TotalPrice,
		ShippingAddress: result.ShippingAddress,
		Notes:           result.Notes,
		Items:           orderItems,
	}

	if err := h.orderRepo.Create(order); err != nil {
		h.response.Error(c, http.StatusInternalServerError, "failed to create order")
		return
	}

	for _, item := range result.Items {
		if item.VariantID != nil && h.variantRepo != nil {
			if err := h.variantRepo.ReserveStock(*item.VariantID, item.Quantity); err != nil {
				h.response.Error(c, http.StatusInternalServerError, "failed to reserve stock")
				return
			}
		} else if h.inventoryRepo != nil {
			if err := h.inventoryRepo.Reserve(item.ProductID, item.Quantity); err != nil {
				h.response.Error(c, http.StatusInternalServerError, "failed to reserve inventory")
				return
			}
		}
	}

	if err := h.cartService.ClearCartAfterCheckout(userID); err != nil {
	}

	items := make([]CheckoutItemResponse, len(result.Items))
	for i, item := range result.Items {
		itemResp := CheckoutItemResponse{
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
		}
		if item.VariantID != nil {
			s := item.VariantID.String()
			itemResp.VariantID = &s
		}
		items[i] = itemResp
	}

	h.response.Created(c, CheckoutResponse{
		Message:         "order created from cart",
		OrderID:         order.ID.String(),
		ShippingAddress: req.ShippingAddress,
		Items:           items,
		TotalPrice:      result.TotalPrice,
		CartCleared:     true,
	})
}

func parseUUIDParam(c *gin.Context, param string) (uuid.UUID, error) {
	value := c.Param(param)
	if value == "" {
		return uuid.Nil, errors.New("empty param")
	}
	result, err := uuid.FromString(value)
	if err != nil {
		return uuid.Nil, err
	}
	return result, nil
}

func ToCartResponse(cart *Cart) CartResponse {
	items := make([]CartItemResponse, len(cart.Items))
	for i, item := range cart.Items {
		items[i] = CartItemResponse(item)
	}
	return CartResponse{
		UserID:    cart.UserID.String(),
		Items:     items,
		CreatedAt: cart.CreatedAt,
		UpdatedAt: cart.UpdatedAt,
	}
}

func (h *CartHandler) parseUserID(c *gin.Context) (uuid.UUID, error) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		return uuid.Nil, errors.New("unauthorized")
	}
	return uuid.FromString(userIDStr)
}
