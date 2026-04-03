package cart

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

type Cart struct {
	UserID    uuid.UUID  `json:"user_id"`
	Items     []CartItem `json:"items"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CartItem struct {
	VariantID string `json:"variant_id"`
	Quantity  int    `json:"quantity"`
}
