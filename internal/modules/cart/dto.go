package cart

import "time"

type AddToCartRequest struct {
	VariantID string `json:"variant_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,gt=0"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" binding:"required,gte=0"`
}

type CartResponse struct {
	UserID    string             `json:"user_id"`
	Items     []CartItemResponse `json:"items"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

type CartItemResponse struct {
	VariantID string `json:"variant_id"`
	Quantity  int    `json:"quantity"`
}

// CheckoutRequest - Request to convert cart to order
type CheckoutRequest struct {
	ShippingAddress string `json:"shipping_address" binding:"required,max=500"`
	Notes           string `json:"notes" binding:"max=1000"`
}

// CheckoutResponse - Response after creating order from cart
type CheckoutResponse struct {
	Message         string                 `json:"message"`
	OrderID         string                 `json:"order_id"`
	ShippingAddress string                 `json:"shipping_address"`
	Items           []CheckoutItemResponse `json:"items"`
	TotalPrice      float64                `json:"total_price"`
	CartCleared     bool                   `json:"cart_cleared"`
}

// CheckoutItemResponse - Individual item in checkout response
type CheckoutItemResponse struct {
	ProductID string  `json:"product_id"`
	VariantID *string `json:"variant_id,omitempty"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}
