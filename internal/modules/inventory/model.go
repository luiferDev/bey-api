package inventory

import (
	"time"

	"bey/internal/shared/uuidutil"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type Inventory struct {
	ID        uuid.UUID      `gorm:"type:uuid;primarykey" json:"id"`
	ProductID uuid.UUID      `gorm:"uniqueIndex;index" json:"product_id"`
	Quantity  int            `gorm:"default:0" json:"quantity"`
	Reserved  int            `gorm:"default:0" json:"reserved"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (i *Inventory) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuidutil.GenerateV7()
	}
	return nil
}

type UpdateInventoryRequest struct {
	Quantity  int        `json:"quantity" binding:"required"`
	VariantID *uuid.UUID `json:"variant_id"`
}

type ReserveReleaseRequest struct {
	Quantity  int        `json:"quantity" binding:"required,gt=0"`
	VariantID *uuid.UUID `json:"variant_id"`
}

type VariantStockInfo struct {
	VariantID string `json:"variant_id"`
	SKU       string `json:"sku"`
	Stock     int    `json:"stock"`
	Reserved  int    `json:"reserved"`
	Available int    `json:"available"`
}

type InventoryResponse struct {
	ProductID      string             `json:"product_id"`
	TotalStock     int                `json:"total_stock"`
	TotalReserved  int                `json:"total_reserved"`
	TotalAvailable int                `json:"total_available"`
	Variants       []VariantStockInfo `json:"variants,omitempty"`
}
