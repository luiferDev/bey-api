package products

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// JSONMap is a type alias for JSON map that works with both GORM and swagger
type JSONMap = datatypes.JSONMap

// Category - Tabla recursiva
type Category struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Name          string         `gorm:"size:255;not null" json:"name"`
	Slug          string         `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Description   string         `gorm:"type:text" json:"description"`
	ParentID      *uint          `gorm:"index" json:"parent_id"`
	Path          string         `gorm:"size:500;index" json:"path"`
	Level         int            `gorm:"default:0;index" json:"level"`
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	SortOrder     int            `gorm:"default:0" json:"sort_order"`
	Subcategories []Category     `gorm:"foreignKey:ParentID" json:"subcategories,omitempty"`
	Products      []Product      `gorm:"foreignKey:CategoryID" json:"products,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// commit
// Product - Información General
type Product struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CategoryID  uint      `json:"category_id"`
	Name        string    `gorm:"size:255;not null" json:"name"`
	Slug        string    `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Brand       string    `gorm:"size:100" json:"brand"`
	Description string    `json:"description"`
	BasePrice   float64   `gorm:"type:decimal(12,2);not null" json:"base_price"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// Relaciones
	Category Category         `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Variants []ProductVariant `gorm:"foreignKey:ProductID" json:"variants,omitempty"`
	Images   []ProductImage   `gorm:"foreignKey:ProductID" json:"images,omitempty"`
}

// ProductVariant - Inventario y SKU
type ProductVariant struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProductID uint      `json:"product_id"`
	SKU       string    `gorm:"size:100;uniqueIndex;not null" json:"sku"`
	Price     float64   `gorm:"type:decimal(12,2);not null" json:"price"`
	Stock     int       `gorm:"default:0" json:"stock"`
	Reserved  int       `gorm:"default:0" json:"reserved"` // Reservado en compras/checkout
	CreatedAt time.Time `json:"created_at"`
	// Relaciones
	Attribute *ProductVariantAttribute `gorm:"foreignKey:VariantID" json:"attribute,omitempty"`
	Images    []ProductImage           `gorm:"foreignKey:VariantID" json:"images,omitempty"`
}

// ProductVariantAttribute - Tabla embebida para atributos específicos de variante
type ProductVariantAttribute struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	VariantID uint   `gorm:"uniqueIndex" json:"variant_id"`
	Color     string `gorm:"size:50;not null" json:"color"`
	Size      string `gorm:"size:20;not null" json:"size"`
	Weight    string `gorm:"size:50;not null" json:"weight"`
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
