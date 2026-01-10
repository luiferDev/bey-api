package orders

import (
	"time"

	"gorm.io/gorm"
)

type Order struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	UserID          uint           `gorm:"index" json:"user_id"`
	Status          string         `gorm:"size:50;default:pending" json:"status"`
	TotalPrice      float64        `gorm:"precision:10;scale:2" json:"total_price"`
	ShippingAddress string         `gorm:"type:text" json:"shipping_address"`
	Notes           string         `gorm:"type:text" json:"notes"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	Items           []OrderItem    `gorm:"foreignKey:OrderID" json:"items"`
}

type OrderItem struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	OrderID   uint      `gorm:"index" json:"order_id"`
	ProductID uint      `gorm:"index" json:"product_id"`
	Quantity  int       `json:"quantity"`
	UnitPrice float64   `gorm:"precision:10;scale:2" json:"unit_price"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateOrderRequest struct {
	UserID          uint                     `json:"user_id" binding:"required"`
	ShippingAddress string                   `json:"shipping_address" binding:"required"`
	Notes           string                   `json:"notes"`
	Items           []CreateOrderItemRequest `json:"items" binding:"required,min=1"`
}

type CreateOrderItemRequest struct {
	ProductID uint `json:"product_id" binding:"required"`
	Quantity  int  `json:"quantity" binding:"required,gt=0"`
}

type OrderResponse struct {
	ID              uint                `json:"id"`
	UserID          uint                `json:"user_id"`
	Status          string              `json:"status"`
	TotalPrice      float64             `json:"total_price"`
	ShippingAddress string              `json:"shipping_address"`
	Notes           string              `json:"notes"`
	Items           []OrderItemResponse `json:"items"`
	CreatedAt       time.Time           `json:"created_at"`
}

type OrderItemResponse struct {
	ID        uint    `json:"id"`
	ProductID uint    `json:"product_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}
