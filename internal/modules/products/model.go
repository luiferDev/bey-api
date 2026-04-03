package products

import (
	"time"

	"bey/internal/shared/uuidutil"

	"github.com/gofrs/uuid/v5"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type JSONMap = datatypes.JSONMap

type Category struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name          string         `gorm:"size:255;not null" json:"name"`
	Slug          string         `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Description   string         `gorm:"type:text" json:"description"`
	ParentID      *uuid.UUID     `gorm:"index" json:"parent_id"`
	Level         int            `gorm:"default:0;index" json:"level"`
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	SortOrder     int            `gorm:"default:0" json:"sort_order"`
	Subcategories []Category     `gorm:"foreignKey:ParentID" json:"subcategories,omitempty"`
	Products      []Product      `gorm:"foreignKey:CategoryID" json:"products,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (c *Category) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuidutil.GenerateV7()
	}
	return nil
}

type Product struct {
	ID          uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	CategoryID  uuid.UUID        `json:"category_id"`
	Name        string           `gorm:"size:255;not null" json:"name"`
	Slug        string           `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Brand       string           `gorm:"size:100" json:"brand"`
	Description string           `json:"description"`
	BasePrice   float64          `gorm:"type:decimal(12,2);not null" json:"base_price"`
	IsActive    bool             `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Category    Category         `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Variants    []ProductVariant `gorm:"foreignKey:ProductID" json:"variants,omitempty"`
	Images      []ProductImage   `gorm:"foreignKey:ProductID" json:"images,omitempty"`
}

func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuidutil.GenerateV7()
	}
	return nil
}

type ProductVariant struct {
	ID        uuid.UUID                `gorm:"type:uuid;primaryKey" json:"id"`
	ProductID uuid.UUID                `json:"product_id"`
	SKU       string                   `gorm:"size:100;uniqueIndex;not null" json:"sku"`
	Price     float64                  `gorm:"type:decimal(12,2);not null" json:"price"`
	Stock     int                      `gorm:"default:0" json:"stock"`
	Reserved  int                      `gorm:"default:0" json:"reserved"`
	CreatedAt time.Time                `json:"created_at"`
	Attribute *ProductVariantAttribute `gorm:"foreignKey:VariantID" json:"attribute,omitempty"`
	Images    []ProductImage           `gorm:"foreignKey:VariantID" json:"images,omitempty"`
}

func (pv *ProductVariant) BeforeCreate(tx *gorm.DB) error {
	if pv.ID == uuid.Nil {
		pv.ID = uuidutil.GenerateV7()
	}
	return nil
}

type ProductVariantAttribute struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	VariantID uuid.UUID `gorm:"uniqueIndex" json:"variant_id"`
	Color     string    `gorm:"size:50;not null" json:"color"`
	Size      string    `gorm:"size:20;not null" json:"size"`
	Weight    string    `gorm:"size:50;not null" json:"weight"`
}

func (pva *ProductVariantAttribute) BeforeCreate(tx *gorm.DB) error {
	if pva.ID == uuid.Nil {
		pva.ID = uuidutil.GenerateV7()
	}
	return nil
}

type ProductImage struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	ProductID uuid.UUID  `json:"product_id"`
	VariantID *uuid.UUID `json:"variant_id"`
	URLImage  string     `gorm:"not null" json:"url_image"`
	IsMain    bool       `gorm:"default:false" json:"is_main"`
	SortOrder int        `gorm:"default:0" json:"sort_order"`
}

func (pi *ProductImage) BeforeCreate(tx *gorm.DB) error {
	if pi.ID == uuid.Nil {
		pi.ID = uuidutil.GenerateV7()
	}
	return nil
}
