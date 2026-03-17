package cart

import "time"

type AddToCartRequest struct {
	VariantID uint `json:"variant_id" binding:"required"`
	Quantity  int  `json:"quantity" binding:"required,gt=0"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" binding:"required,gte=0"`
}

type CartResponse struct {
	UserID    uint               `json:"user_id"`
	Items     []CartItemResponse `json:"items"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

type CartItemResponse struct {
	VariantID uint `json:"variant_id"`
	Quantity  int  `json:"quantity"`
}
