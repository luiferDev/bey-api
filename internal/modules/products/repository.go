package products

import (
	"errors"
	"log"

	"golang.org/x/sync/errgroup"
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
		log.Printf("ERROR: Failed to find category by id %d: %v", id, err)
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
		log.Printf("ERROR: Failed to find category by slug %s: %v", slug, err)
		return nil, err
	}
	return &category, nil
}

func (r *CategoryRepository) Update(category *Category) error {
	if err := r.db.Save(category).Error; err != nil {
		log.Printf("ERROR: Failed to update category %d: %v", category.ID, err)
		return err
	}
	return nil
}

func (r *CategoryRepository) Delete(id uint) error {
	if err := r.db.Delete(&Category{}, id).Error; err != nil {
		log.Printf("ERROR: Failed to delete category %d: %v", id, err)
		return err
	}
	return nil
}

func (r *CategoryRepository) FindAll() ([]Category, error) {
	var categories []Category
	if err := r.db.Where("parent_id IS NULL").Preload("Subcategories").Find(&categories).Error; err != nil {
		log.Printf("ERROR: Failed to find all categories: %v", err)
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
	db          *gorm.DB
	variantRepo *ProductVariantRepository
	imageRepo   *ProductImageRepository
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func NewProductRepositoryWithRelations(db *gorm.DB, variantRepo *ProductVariantRepository, imageRepo *ProductImageRepository) *ProductRepository {
	return &ProductRepository{
		db:          db,
		variantRepo: variantRepo,
		imageRepo:   imageRepo,
	}
}

func (r *ProductRepository) Create(product *Product) error {
	if err := r.db.Create(product).Error; err != nil {
		log.Printf("ERROR: Failed to create product %s: %v", product.Name, err)
		return err
	}
	return nil
}

func (r *ProductRepository) FindByID(id uint) (*Product, error) {
	var product Product
	if err := r.db.Preload("Category").Preload("Variants.Attribute").Preload("Variants.Images").Preload("Images").First(&product, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find product by id %d: %v", id, err)
		return nil, err
	}
	return &product, nil
}

// GetPriceByID returns only the BasePrice of a product (optimized for order creation)
func (r *ProductRepository) GetPriceByID(id uint) (float64, error) {
	var product Product
	if err := r.db.Model(&Product{}).Select("base_price").First(&product, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return product.BasePrice, nil
}

func (r *ProductRepository) FindBySlug(slug string) (*Product, error) {
	var product Product
	if err := r.db.Preload("Category").Preload("Variants.Attribute").Preload("Variants.Images").Preload("Images").Where("slug = ?", slug).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find product by slug %s: %v", slug, err)
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepository) Update(product *Product) error {
	if err := r.db.Save(product).Error; err != nil {
		log.Printf("ERROR: Failed to update product %d: %v", product.ID, err)
		return err
	}
	return nil
}

func (r *ProductRepository) Delete(id uint) error {
	if err := r.db.Delete(&Product{}, id).Error; err != nil {
		log.Printf("ERROR: Failed to delete product %d: %v", id, err)
		return err
	}
	return nil
}

func (r *ProductRepository) FindAll(offset, limit int) ([]Product, error) {
	var products []Product
	query := r.db.Preload("Category").Preload("Images")
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}
	if err := query.Find(&products).Error; err != nil {
		log.Printf("ERROR: Failed to find all products: %v", err)
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
		log.Printf("ERROR: Failed to find products by category %d: %v", categoryID, err)
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
		log.Printf("ERROR: Failed to find active products: %v", err)
		return nil, err
	}
	return products, nil
}

type ProductWithRelations struct {
	Product  *Product
	Variants []ProductVariant
	Images   []ProductImage
}

func (r *ProductRepository) FindByIDWithRelationsParallel(id uint) (*ProductWithRelations, error) {
	if r.variantRepo == nil || r.imageRepo == nil {
		return nil, errors.New("variant and image repositories are required for parallel fetch")
	}

	result := &ProductWithRelations{}

	var eg errgroup.Group

	eg.Go(func() error {
		product, err := r.FindByID(id)
		result.Product = product
		return err
	})

	eg.Go(func() error {
		variants, err := r.variantRepo.FindByProductID(id)
		result.Variants = variants
		return err
	})

	eg.Go(func() error {
		images, err := r.imageRepo.FindByProductID(id)
		result.Images = images
		return err
	})

	if err := eg.Wait(); err != nil {
		if result.Product == nil {
			return nil, nil
		}
		return nil, err
	}

	if result.Product == nil {
		return nil, nil
	}

	return result, nil
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
	if err := r.db.Preload("Attribute").Preload("Images").First(&variant, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &variant, nil
}

func (r *ProductVariantRepository) FindBySKU(sku string) (*ProductVariant, error) {
	var variant ProductVariant
	if err := r.db.Preload("Attribute").Preload("Images").Where("sku = ?", sku).First(&variant).Error; err != nil {
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
	if err := r.db.Where("product_id = ?", productID).Preload("Attribute").Preload("Images").Find(&variants).Error; err != nil {
		return nil, err
	}
	return variants, nil
}

func (r *ProductVariantRepository) UpdateStock(id uint, stock int) error {
	return r.db.Model(&ProductVariant{}).Where("id = ?", id).Update("stock", stock).Error
}

// GetPriceAndStock returns price and available stock for a variant
func (r *ProductVariantRepository) GetPriceAndStock(id uint) (float64, int, int, error) {
	var variant ProductVariant
	if err := r.db.Select("price", "stock", "reserved").First(&variant, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, 0, 0, nil
		}
		return 0, 0, 0, err
	}
	return variant.Price, variant.Stock, variant.Reserved, nil
}

// ReserveStock reserves stock for a variant (used in orders)
// Decrements stock, increments reserved
func (r *ProductVariantRepository) ReserveStock(id uint, quantity int) error {
	result := r.db.Model(&ProductVariant{}).
		Where("id = ? AND (stock - reserved) >= ?", id, quantity).
		Updates(map[string]interface{}{
			"stock":    gorm.Expr("stock - ?", quantity),
			"reserved": gorm.Expr("reserved + ?", quantity),
		})
	if result.RowsAffected == 0 {
		return errors.New("insufficient stock")
	}
	return result.Error
}

// ReleaseStock releases reserved stock (e.g., on order cancellation)
// Increments stock, decrements reserved
func (r *ProductVariantRepository) ReleaseStock(id uint, quantity int) error {
	result := r.db.Model(&ProductVariant{}).
		Where("id = ? AND reserved >= ?", id, quantity).
		Updates(map[string]interface{}{
			"stock":    gorm.Expr("stock + ?", quantity),
			"reserved": gorm.Expr("reserved - ?", quantity),
		})
	if result.RowsAffected == 0 {
		return errors.New("insufficient reserved stock")
	}
	return result.Error
}

// ConfirmSale converts reserved stock to sold (e.g., after payment)
// Just decrements reserved (stock already decreased)
func (r *ProductVariantRepository) ConfirmSale(id uint, quantity int) error {
	return r.db.Model(&ProductVariant{}).
		Where("id = ? AND reserved >= ?", id, quantity).
		Update("reserved", gorm.Expr("reserved - ?", quantity)).Error
}

// ProductVariantAttribute CRUD
func (r *ProductVariantRepository) CreateAttribute(attribute *ProductVariantAttribute) error {
	return r.db.Create(attribute).Error
}

func (r *ProductVariantRepository) FindAttributeByVariantID(variantID uint) (*ProductVariantAttribute, error) {
	var attribute ProductVariantAttribute
	if err := r.db.Where("variant_id = ?", variantID).First(&attribute).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &attribute, nil
}

func (r *ProductVariantRepository) UpdateAttribute(attribute *ProductVariantAttribute) error {
	return r.db.Save(attribute).Error
}

func (r *ProductVariantRepository) DeleteAttribute(variantID uint) error {
	return r.db.Where("variant_id = ?", variantID).Delete(&ProductVariantAttribute{}).Error
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
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Desmarcar todas las imágenes como principales
		if err := tx.Model(&ProductImage{}).Where("product_id = ?", productID).Update("is_main", false).Error; err != nil {
			return err
		}

		// Marcar la imagen específica como principal
		if err := tx.Model(&ProductImage{}).Where("id = ? AND product_id = ?", imageID, productID).Update("is_main", true).Error; err != nil {
			return err
		}

		return nil
	})
}
