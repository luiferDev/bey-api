package orders

import (
	"time"

	"bey/internal/shared/uuidutil"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type Order struct {
	ID                   uuid.UUID      `gorm:"type:uuid;primarykey" json:"id"`
	UserID               uuid.UUID      `gorm:"index" json:"user_id"`
	Status               string         `gorm:"size:50;default:pending" json:"status"`
	TotalPrice           float64        `gorm:"precision:10;scale:2" json:"total_price"`
	ShippingAddress      string         `gorm:"type:text" json:"shipping_address"`
	Notes                string         `gorm:"type:text" json:"notes"`
	PaymentTransactionID string         `gorm:"size:255" json:"payment_transaction_id"`
	PaymentLinkID        string         `gorm:"size:255" json:"payment_link_id"`
	PaymentStatus        string         `gorm:"size:50;default:pending" json:"payment_status"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
	Items                []OrderItem    `gorm:"foreignKey:OrderID" json:"items"`
}

func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuidutil.GenerateV7()
	}
	return nil
}

type OrderItem struct {
	ID        uuid.UUID  `gorm:"type:uuid;primarykey" json:"id"`
	OrderID   uuid.UUID  `gorm:"index" json:"order_id"`
	ProductID uuid.UUID  `gorm:"index" json:"product_id"`
	VariantID *uuid.UUID `gorm:"index" json:"variant_id"`
	Quantity  int        `json:"quantity"`
	UnitPrice float64    `gorm:"precision:10;scale:2" json:"unit_price"`
	CreatedAt time.Time  `json:"created_at"`
}

func (oi *OrderItem) BeforeCreate(tx *gorm.DB) error {
	if oi.ID == uuid.Nil {
		oi.ID = uuidutil.GenerateV7()
	}
	return nil
}

type CreateOrderRequest struct {
	ShippingAddress string                   `json:"shipping_address" binding:"required,max=500"`
	Notes           string                   `json:"notes" binding:"max=1000"`
	Items           []CreateOrderItemRequest `json:"items" binding:"required,min=1,max=50"`
}

type CreateOrderItemRequest struct {
	ProductID string  `json:"product_id" binding:"required"`
	VariantID *string `json:"variant_id"`
	Quantity  int     `json:"quantity" binding:"required,gt=0"`
}

type OrderResponse struct {
	ID              string              `json:"id"`
	UserID          string              `json:"user_id"`
	Status          string              `json:"status"`
	TotalPrice      float64             `json:"total_price"`
	ShippingAddress string              `json:"shipping_address"`
	Notes           string              `json:"notes"`
	Items           []OrderItemResponse `json:"items"`
	CreatedAt       time.Time           `json:"created_at"`
}

type OrderItemResponse struct {
	ID        string  `json:"id"`
	ProductID string  `json:"product_id"`
	VariantID *string `json:"variant_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}
