package cart

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"bey/internal/shared/response"
)

type CartHandler struct {
	cartService *CartService
	response    *response.ResponseHandler
}

func NewCartHandler(cartService *CartService) *CartHandler {
	return &CartHandler{
		cartService: cartService,
		response:    response.NewResponseHandler(),
	}
}

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
