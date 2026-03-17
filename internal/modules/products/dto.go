package products

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"gorm.io/datatypes"
	"time"
)

// Category DTOs
type CreateCategoryRequest struct {
	ParentID    *uint  `json:"parent_id"`
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug" binding:"required"`
	Description string `json:"description"`
}

type UpdateCategoryRequest struct {
	ParentID    *uint   `json:"parent_id"`
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
}

type CategoryResponse struct {
	ID            uint               `json:"id"`
	ParentID      *uint              `json:"parent_id"`
	Name          string             `json:"name"`
	Slug          string             `json:"slug"`
	Description   string             `json:"description"`
	CreatedAt     time.Time          `json:"created_at"`
	Subcategories []CategoryResponse `json:"subcategories,omitempty"`
}

// Product DTOs
type CreateProductRequest struct {
	CategoryID  uint    `json:"category_id" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Slug        string  `json:"slug" binding:"required"`
	Brand       string  `json:"brand"`
	Description string  `json:"description"`
	BasePrice   float64 `json:"base_price" binding:"required,gt=0"`
	IsActive    *bool   `json:"is_active"`
}

type UpdateProductRequest struct {
	CategoryID  *uint    `json:"category_id"`
	Name        *string  `json:"name"`
	Slug        *string  `json:"slug"`
	Brand       *string  `json:"brand"`
	Description *string  `json:"description"`
	BasePrice   *float64 `json:"base_price"`
	IsActive    *bool    `json:"is_active"`
}

type ProductResponse struct {
	ID          uint                     `json:"id"`
	CategoryID  uint                     `json:"category_id"`
	Name        string                   `json:"name"`
	Slug        string                   `json:"slug"`
	Brand       string                   `json:"brand"`
	Description string                   `json:"description"`
	BasePrice   float64                  `json:"base_price"`
	IsActive    bool                     `json:"is_active"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
	Category    *CategoryResponse        `json:"category,omitempty"`
	Variants    []ProductVariantResponse `json:"variants,omitempty"`
	Images      []ProductImageResponse   `json:"images,omitempty"`
}

// ProductVariant DTOs
type CreateProductVariantRequest struct {
	ProductID uint    `json:"product_id" binding:"required"`
	SKU       string  `json:"sku" binding:"required"`
	Price     float64 `json:"price" binding:"required,gt=0"`
	Stock     int     `json:"stock"`
	Color     string  `json:"color" binding:"required"`
	Size      string  `json:"size" binding:"required"`
	Weight    string  `json:"weight" binding:"required"`
}

type UpdateProductVariantRequest struct {
	SKU    *string  `json:"sku"`
	Price  *float64 `json:"price"`
	Stock  *int     `json:"stock"`
	Color  *string  `json:"color"`
	Size   *string  `json:"size"`
	Weight *string  `json:"weight"`
}

type ProductVariantAttributeResponse struct {
	Color  string `json:"color"`
	Size   string `json:"size"`
	Weight string `json:"weight"`
}

type ProductVariantResponse struct {
	ID        uint                             `json:"id"`
	ProductID uint                             `json:"product_id"`
	SKU       string                           `json:"sku"`
	Price     float64                          `json:"price"`
	Stock     int                              `json:"stock"`
	Reserved  int                              `json:"reserved"`
	Available int                              `json:"available"`
	Attribute *ProductVariantAttributeResponse `json:"attribute,omitempty"`
	CreatedAt time.Time                        `json:"created_at"`
	Images    []ProductImageResponse           `json:"images,omitempty"`
}

// ProductImage DTOs
type CreateProductImageRequest struct {
	ProductID uint   `json:"product_id" binding:"required"`
	VariantID *uint  `json:"variant_id"`
	URLImage  string `json:"url_image" binding:"required"`
	IsMain    *bool  `json:"is_main"`
	SortOrder int    `json:"sort_order"`
}

type UpdateProductImageRequest struct {
	URLImage  *string `json:"url_image"`
	IsMain    *bool   `json:"is_main"`
	SortOrder *int    `json:"sort_order"`
}

type ProductImageResponse struct {
	ID        uint   `json:"id"`
	ProductID uint   `json:"product_id"`
	VariantID *uint  `json:"variant_id"`
	URLImage  string `json:"url_image"`
	IsMain    bool   `json:"is_main"`
	SortOrder int    `json:"sort_order"`
}

var validAttributeKeys = map[string]bool{
	"color":  true,
	"size":   true,
	"weight": true,
}

func validateAttributes(fl validator.FieldLevel) bool {
	attrs, ok := fl.Field().Interface().(datatypes.JSONMap)
	if !ok {
		return false
	}

	for key := range attrs {
		if !validAttributeKeys[key] {
			return false
		}
	}
	return true
}

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("validAttributes", validateAttributes)
	}
}
