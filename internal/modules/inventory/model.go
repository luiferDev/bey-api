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
	Quantity *int `json:"quantity"`
}

type InventoryResponse struct {
	ID        string    `json:"id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Reserved  int       `json:"reserved"`
	Available int       `json:"available"`
	UpdatedAt time.Time `json:"updated_at"`
}
