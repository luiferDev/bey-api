package products

import (
	"errors"

	"gorm.io/gorm"
)

// CategoryRepository
type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(category *Category) error {
	return r.db.Create(category).Error
}

func (r *CategoryRepository) FindByID(id uint) (*Category, error) {
	var category Category
	if err := r.db.Preload("Subcategories").First(&category, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &category, nil
}

func (r *CategoryRepository) FindBySlug(slug string) (*Category, error) {
	var category Category
	if err := r.db.Preload("Subcategories").Where("slug = ?", slug).First(&category).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &category, nil
}

func (r *CategoryRepository) Update(category *Category) error {
	return r.db.Save(category).Error
}

func (r *CategoryRepository) Delete(id uint) error {
	return r.db.Delete(&Category{}, id).Error
}

func (r *CategoryRepository) FindAll() ([]Category, error) {
	var categories []Category
	if err := r.db.Where("parent_id IS NULL").Preload("Subcategories").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *CategoryRepository) FindByParentID(parentID uint) ([]Category, error) {
	var categories []Category
	if err := r.db.Where("parent_id = ?", parentID).Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

// ProductRepository
type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(product *Product) error {
	return r.db.Create(product).Error
}

func (r *ProductRepository) FindByID(id uint) (*Product, error) {
	var product Product
	if err := r.db.Preload("Category").Preload("Variants").Preload("Images").First(&product, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepository) FindBySlug(slug string) (*Product, error) {
	var product Product
	if err := r.db.Preload("Category").Preload("Variants").Preload("Images").Where("slug = ?", slug).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepository) Update(product *Product) error {
	return r.db.Save(product).Error
}

func (r *ProductRepository) Delete(id uint) error {
	return r.db.Delete(&Product{}, id).Error
}

func (r *ProductRepository) FindAll(offset, limit int) ([]Product, error) {
	var products []Product
	query := r.db.Preload("Category").Preload("Images")
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}
	if err := query.Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

func (r *ProductRepository) FindByCategoryID(categoryID uint, offset, limit int) ([]Product, error) {
	var products []Product
	query := r.db.Where("category_id = ?", categoryID).Preload("Category").Preload("Images")
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}
	if err := query.Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

func (r *ProductRepository) FindByActive(isActive bool, offset, limit int) ([]Product, error) {
	var products []Product
	query := r.db.Where("is_active = ?", isActive).Preload("Category").Preload("Images")
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}
	if err := query.Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// ProductVariantRepository
type ProductVariantRepository struct {
	db *gorm.DB
}

func NewProductVariantRepository(db *gorm.DB) *ProductVariantRepository {
	return &ProductVariantRepository{db: db}
}

func (r *ProductVariantRepository) Create(variant *ProductVariant) error {
	return r.db.Create(variant).Error
}

func (r *ProductVariantRepository) FindByID(id uint) (*ProductVariant, error) {
	var variant ProductVariant
	if err := r.db.Preload("Images").First(&variant, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &variant, nil
}

func (r *ProductVariantRepository) FindBySKU(sku string) (*ProductVariant, error) {
	var variant ProductVariant
	if err := r.db.Preload("Images").Where("sku = ?", sku).First(&variant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &variant, nil
}

func (r *ProductVariantRepository) Update(variant *ProductVariant) error {
	return r.db.Save(variant).Error
}

func (r *ProductVariantRepository) Delete(id uint) error {
	return r.db.Delete(&ProductVariant{}, id).Error
}

func (r *ProductVariantRepository) FindByProductID(productID uint) ([]ProductVariant, error) {
	var variants []ProductVariant
	if err := r.db.Where("product_id = ?", productID).Preload("Images").Find(&variants).Error; err != nil {
		return nil, err
	}
	return variants, nil
}

func (r *ProductVariantRepository) UpdateStock(id uint, stock int) error {
	return r.db.Model(&ProductVariant{}).Where("id = ?", id).Update("stock", stock).Error
}

// ProductImageRepository
type ProductImageRepository struct {
	db *gorm.DB
}

func NewProductImageRepository(db *gorm.DB) *ProductImageRepository {
	return &ProductImageRepository{db: db}
}

func (r *ProductImageRepository) Create(image *ProductImage) error {
	return r.db.Create(image).Error
}

func (r *ProductImageRepository) FindByID(id uint) (*ProductImage, error) {
	var image ProductImage
	if err := r.db.First(&image, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &image, nil
}

func (r *ProductImageRepository) Update(image *ProductImage) error {
	return r.db.Save(image).Error
}

func (r *ProductImageRepository) Delete(id uint) error {
	return r.db.Delete(&ProductImage{}, id).Error
}

func (r *ProductImageRepository) FindByProductID(productID uint) ([]ProductImage, error) {
	var images []ProductImage
	if err := r.db.Where("product_id = ?", productID).Order("sort_order ASC").Find(&images).Error; err != nil {
		return nil, err
	}
	return images, nil
}

func (r *ProductImageRepository) FindByVariantID(variantID uint) ([]ProductImage, error) {
	var images []ProductImage
	if err := r.db.Where("variant_id = ?", variantID).Order("sort_order ASC").Find(&images).Error; err != nil {
		return nil, err
	}
	return images, nil
}

func (r *ProductImageRepository) SetMainImage(productID uint, imageID uint) error {
	tx := r.db.Begin()
	
	// Desmarcar todas las imágenes como principales
	if err := tx.Model(&ProductImage{}).Where("product_id = ?", productID).Update("is_main", false).Error; err != nil {
		tx.Rollback()
		return err
	}
	
	// Marcar la imagen específica como principal
	if err := tx.Model(&ProductImage{}).Where("id = ? AND product_id = ?", imageID, productID).Update("is_main", true).Error; err != nil {
		tx.Rollback()
		return err
	}
	
	return tx.Commit().Error
}
