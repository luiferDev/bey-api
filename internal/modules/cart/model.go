package cart

import "time"

type Cart struct {
	UserID    string     `json:"user_id"`
	Items     []CartItem `json:"items"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CartItem struct {
	VariantID string `json:"variant_id"`
	Quantity  int    `json:"quantity"`
}
