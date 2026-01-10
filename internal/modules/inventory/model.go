package inventory

import (
	"time"

	"gorm.io/gorm"
)

type Inventory struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	ProductID uint           `gorm:"uniqueIndex;index" json:"product_id"`
	Quantity  int            `gorm:"default:0" json:"quantity"`
	Reserved  int            `gorm:"default:0" json:"reserved"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type UpdateInventoryRequest struct {
	Quantity *int `json:"quantity"`
}

type InventoryResponse struct {
	ID        uint      `json:"id"`
	ProductID uint      `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Reserved  int       `json:"reserved"`
	Available int       `json:"available"`
	UpdatedAt time.Time `json:"updated_at"`
}
