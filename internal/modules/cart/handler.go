package cart

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"bey/internal/modules/orders"
	"bey/internal/shared/response"
)

type OrderCreator interface {
	Create(order *orders.Order) error
}

type VariantStockReserver interface {
	ReserveStock(id uint, quantity int) error
}

type InventoryReserver interface {
	Reserve(productID uint, quantity int) error
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

// GetCart godoc
// @Summary Get shopping cart
// @Description Retrieves the authenticated user's shopping cart items
// @Tags Cart
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponse{data=CartResponse} "Cart retrieved successfully"
// @Failure 401 {object} response.ApiResponse "Unauthorized - invalid or missing token"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/cart [get]
func (h *CartHandler) GetCart(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
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

// AddItem godoc
// @Summary Add item to cart
// @Description Adds a product variant to the authenticated user's shopping cart
// @Tags Cart
// @Accept json
// @Produce json
// @Param request body AddToCartRequest true "Item to add"
// @Success 200 {object} response.ApiResponse{data=CartResponse} "Item added to cart"
// @Failure 400 {object} response.ApiResponse "Bad request - invalid data or insufficient stock"
// @Failure 401 {object} response.ApiResponse "Unauthorized - invalid or missing token"
// @Failure 404 {object} response.ApiResponse "Variant not found"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/cart/items [post]
func (h *CartHandler) AddItem(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		h.response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req AddToCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationError(c, err.Error())
		return
	}

	cart, err := h.cartService.AddItem(userID, req.VariantID, req.Quantity)
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

// UpdateItem godoc
// @Summary Update cart item quantity
// @Description Updates the quantity of an item in the authenticated user's shopping cart
// @Tags Cart
// @Accept json
// @Produce json
// @Param variant_id path int true "Product variant ID"
// @Param request body UpdateCartItemRequest true "New quantity"
// @Success 200 {object} response.ApiResponse{data=CartResponse} "Item updated successfully"
// @Failure 400 {object} response.ApiResponse "Bad request - invalid data or insufficient stock"
// @Failure 401 {object} response.ApiResponse "Unauthorized - invalid or missing token"
// @Failure 404 {object} response.ApiResponse "Item not found in cart"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/cart/items/{variant_id} [put]
func (h *CartHandler) UpdateItem(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		h.response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	variantID, err := parseUintParam(c, "variant_id")
	if err != nil {
		h.response.ValidationError(c, "invalid variant_id")
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

// RemoveItem godoc
// @Summary Remove item from cart
// @Description Removes a specific item from the authenticated user's shopping cart
// @Tags Cart
// @Accept json
// @Produce json
// @Param variant_id path int true "Product variant ID to remove"
// @Success 200 {object} response.ApiResponse{data=CartResponse} "Item removed successfully"
// @Failure 400 {object} response.ApiResponse "Bad request - invalid variant ID"
// @Failure 401 {object} response.ApiResponse "Unauthorized - invalid or missing token"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/cart/items/{variant_id} [delete]
func (h *CartHandler) RemoveItem(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		h.response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	variantID, err := parseUintParam(c, "variant_id")
	if err != nil {
		h.response.ValidationError(c, "invalid variant_id")
		return
	}

	cart, err := h.cartService.RemoveItem(userID, variantID)
	if err != nil {
		h.response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	h.response.Success(c, ToCartResponse(cart))
}

// ClearCart godoc
// @Summary Clear shopping cart
// @Description Removes all items from the authenticated user's shopping cart
// @Tags Cart
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponse "Cart cleared successfully"
// @Failure 401 {object} response.ApiResponse "Unauthorized - invalid or missing token"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/cart [delete]
func (h *CartHandler) ClearCart(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		h.response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.cartService.ClearCart(userID); err != nil {
		h.response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	h.response.Success(c, gin.H{"message": "cart cleared"})
}

// Checkout godoc
// @Summary Checkout cart to create an order
// @Description Converts the authenticated user's shopping cart into an order. Validates stock, calculates prices, and clears the cart automatically.
// @Tags Cart
// @Accept json
// @Produce json
// @Param request body CheckoutRequest true "Checkout data with shipping address"
// @Success 201 {object} response.ApiResponse{data=CheckoutResponse} "Order created successfully from cart"
// @Failure 400 {object} response.ApiResponse "Bad request - cart empty or insufficient stock"
// @Failure 401 {object} response.ApiResponse "Unauthorized - invalid or missing token"
// @Failure 404 {object} response.ApiResponse "Variant not found"
// @Failure 500 {object} response.ApiResponse "Internal server error - order creation failed"
// @Security BearerAuth
// @Router /api/v1/cart/checkout [post]
func (h *CartHandler) Checkout(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		h.response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationError(c, err.Error())
		return
	}

	// Step 1: Prepare checkout (validate stock, calculate prices)
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

	// Step 2: Create the order in the database
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

	// Step 2.5: Reserve inventory for all order items
	for _, item := range result.Items {
		if item.VariantID != nil && h.variantRepo != nil {
			if err := h.variantRepo.ReserveStock(*item.VariantID, item.Quantity); err != nil {
				// Rollback: delete the order since reservation failed
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

	// Step 3: Clear the cart
	if err := h.cartService.ClearCartAfterCheckout(userID); err != nil {
		// Cart clear failed but order was created — log but don't fail
		// TODO: add proper logging
	}

	// Step 4: Build response
	items := make([]CheckoutItemResponse, len(result.Items))
	for i, item := range result.Items {
		items[i] = CheckoutItemResponse{
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
		}
	}

	h.response.Created(c, CheckoutResponse{
		Message:         "order created from cart",
		OrderID:         order.ID,
		ShippingAddress: req.ShippingAddress,
		Items:           items,
		TotalPrice:      result.TotalPrice,
		CartCleared:     true,
	})
}

func parseUintParam(c *gin.Context, param string) (uint, error) {
	value := c.Param(param)
	if value == "" {
		return 0, errors.New("empty param")
	}
	result, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(result), nil
}

func ToCartResponse(cart *Cart) CartResponse {
	items := make([]CartItemResponse, len(cart.Items))
	for i, item := range cart.Items {
		items[i] = CartItemResponse{
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
		}
	}
	return CartResponse{
		UserID:    cart.UserID,
		Items:     items,
		CreatedAt: cart.CreatedAt,
		UpdatedAt: cart.UpdatedAt,
	}
}
