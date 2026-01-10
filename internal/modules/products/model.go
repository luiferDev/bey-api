package products

import (
	"time"

	"gorm.io/datatypes"
)

// Category - Tabla recursiva
type Category struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	ParentID    *uint      `json:"parent_id"`
	Name        string     `gorm:"size:100;not null" json:"name"`
	Slug        string     `gorm:"size:150;uniqueIndex;not null" json:"slug"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	// Relaciones
	Subcategories []Category `gorm:"foreignKey:ParentID" json:"subcategories,omitempty"`
	Products      []Product  `gorm:"foreignKey:CategoryID" json:"products,omitempty"`
}
// commit
// Product - Información General
type Product struct {
	ID          uint             `gorm:"primaryKey" json:"id"`
	CategoryID  uint             `json:"category_id"`
	Name        string           `gorm:"size:255;not null" json:"name"`
	Slug        string           `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Brand       string           `gorm:"size:100" json:"brand"`
	Description string           `json:"description"`
	BasePrice   float64          `gorm:"type:decimal(12,2);not null" json:"base_price"`
	IsActive    bool             `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	// Relaciones
	Category    Category         `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Variants    []ProductVariant `gorm:"foreignKey:ProductID" json:"variants,omitempty"`
	Images      []ProductImage   `gorm:"foreignKey:ProductID" json:"images,omitempty"`
}

// ProductVariant - Inventario y SKU
type ProductVariant struct {
	ID         uint               `gorm:"primaryKey" json:"id"`
	ProductID  uint               `json:"product_id"`
	SKU        string             `gorm:"size:100;uniqueIndex;not null" json:"sku"`
	Price      float64            `gorm:"type:decimal(12,2);not null" json:"price"`
	Stock      int                `gorm:"default:0" json:"stock"`
	Attributes datatypes.JSONMap  `gorm:"type:jsonb;not null" json:"attributes"`
	CreatedAt  time.Time          `json:"created_at"`
	// Relaciones
	Images     []ProductImage     `gorm:"foreignKey:VariantID" json:"images,omitempty"`
}

// ProductImage - Imágenes de productos/variantes
type ProductImage struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ProductID uint   `json:"product_id"`
	VariantID *uint  `json:"variant_id"` // Puntero para permitir nulos
	URLImage  string `gorm:"not null" json:"url_image"`
	IsMain    bool   `gorm:"default:false" json:"is_main"`
	SortOrder int    `gorm:"default:0" json:"sort_order"`
}
