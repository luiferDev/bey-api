package products

import (
	"context"
	"errors"
	"fmt"
	"log"

	"bey/internal/shared/cache"

	"github.com/gofrs/uuid/v5"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// CategoryRepository
type CategoryRepository struct {
	db      *gorm.DB
	cache   *cache.CacheService
	metrics *cache.CacheMetrics
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func NewCategoryRepositoryWithCache(db *gorm.DB, cacheSvc *cache.CacheService, metrics *cache.CacheMetrics) *CategoryRepository {
	return &CategoryRepository{db: db, cache: cacheSvc, metrics: metrics}
}

func (r *CategoryRepository) Create(category *Category) error {
	if category.ParentID != nil && *category.ParentID != uuid.Nil {
		var parent Category
		if err := r.db.First(&parent, *category.ParentID).Error; err != nil {
			return fmt.Errorf("parent category not found: %w", err)
		}
		category.Level = parent.Level + 1
	} else {
		category.ParentID = nil
		category.Level = 0
	}

	return r.db.Create(category).Error
}

func (r *CategoryRepository) FindByID(id uuid.UUID) (*Category, error) {
	if r.cache != nil {
		key := r.cache.Key("cache", "category", id.String())
		var category Category
		hit, err := r.cache.Get(context.Background(), key, &category)
		if err != nil {
			log.Printf("Cache error: %v", err)
			r.metrics.Error()
		}
		if hit {
			r.metrics.Hit()
			return &category, nil
		}
		r.metrics.Miss()
	}

	var category Category
	if err := r.db.Preload("Subcategories").First(&category, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find category by id %s: %v", id.String(), err)
		return nil, err
	}

	if r.cache != nil {
		key := r.cache.Key("cache", "category", category.ID.String())
		if err := r.cache.Set(context.Background(), key, category); err != nil {
			log.Printf("Cache set error: %v", err)
		}
	}

	return &category, nil
}

func (r *CategoryRepository) FindBySlug(slug string) (*Category, error) {
	if r.cache != nil {
		key := r.cache.Key("cache", "category", "slug", slug)
		var category Category
		hit, err := r.cache.Get(context.Background(), key, &category)
		if err != nil {
			log.Printf("Cache error: %v", err)
			r.metrics.Error()
		}
		if hit {
			r.metrics.Hit()
			return &category, nil
		}
		r.metrics.Miss()
	}

	var category Category
	if err := r.db.Preload("Subcategories").Where("slug = ?", slug).First(&category).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find category by slug %s: %v", slug, err)
		return nil, err
	}

	if r.cache != nil {
		key := r.cache.Key("cache", "category", "slug", slug)
		if err := r.cache.Set(context.Background(), key, category); err != nil {
			log.Printf("Cache set error: %v", err)
		}
	}

	return &category, nil
}

func (r *CategoryRepository) Update(category *Category) error {
	var existing Category
	if err := r.db.First(&existing, category.ID).Error; err != nil {
		return err
	}

	oldParentID := existing.ParentID
	newParentID := category.ParentID

	if newParentID != nil && *newParentID != uuid.Nil {
		if r.isDescendant(*newParentID, category.ID) {
			return errors.New("circular reference detected: category cannot be its own descendant")
		}
	}

	if err := r.db.Model(category).Updates(category).Error; err != nil {
		return err
	}

	if (oldParentID == nil && newParentID != nil) ||
		(oldParentID != nil && newParentID == nil) ||
		(oldParentID != nil && newParentID != nil && *oldParentID != *newParentID) {
		var newLevel int
		if newParentID != nil && *newParentID != uuid.Nil {
			var parent Category
			if err := r.db.First(&parent, *newParentID).Error; err != nil {
				return err
			}
			newLevel = parent.Level + 1
		} else {
			newLevel = 0
		}
		if err := r.db.Model(&Category{}).Where("id = ?", category.ID).Update("level", newLevel).Error; err != nil {
			return err
		}
	}

	return nil
}

func (r *CategoryRepository) isDescendant(potentialDescendantID, ancestorID uuid.UUID) bool {
	var rows []uuid.UUID
	if err := r.db.Raw(`
		WITH RECURSIVE category_ancestors AS (
			SELECT id, parent_id FROM categories WHERE id = $1
			UNION ALL
			SELECT c.id, c.parent_id
			FROM categories c
			INNER JOIN category_ancestors ca ON ca.parent_id = c.id
		)
		SELECT id FROM category_ancestors
	`, potentialDescendantID).Scan(&rows).Error; err != nil {
		return false
	}
	for _, id := range rows {
		if id == ancestorID {
			return true
		}
	}
	return false
}

func (r *CategoryRepository) Delete(id uuid.UUID) error {
	var count int64
	r.db.Model(&Category{}).Where("parent_id = ?", id).Count(&count)
	if count > 0 {
		return errors.New("cannot delete category with children")
	}
	return r.db.Delete(&Category{}, id).Error
}

func (r *CategoryRepository) FindAll() ([]Category, error) {
	if r.cache != nil {
		key := r.cache.Key("cache", "category", "list")
		var categories []Category
		hit, err := r.cache.Get(context.Background(), key, &categories)
		if err != nil {
			log.Printf("Cache error: %v", err)
			r.metrics.Error()
		}
		if hit {
			r.metrics.Hit()
			return categories, nil
		}
		r.metrics.Miss()
	}

	var categories []Category
	if err := r.db.Where("parent_id IS NULL").Preload("Subcategories").Find(&categories).Error; err != nil {
		log.Printf("ERROR: Failed to find all categories: %v", err)
		return nil, err
	}

	if r.cache != nil {
		key := r.cache.Key("cache", "category", "list")
		if err := r.cache.Set(context.Background(), key, categories); err != nil {
			log.Printf("Cache set error: %v", err)
		}
	}

	return categories, nil
}

func (r *CategoryRepository) FindByParentID(parentID uuid.UUID) ([]Category, error) {
	var categories []Category
	if err := r.db.Where("parent_id = ?", parentID).Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *CategoryRepository) FindTree() ([]Category, error) {
	var categories []Category
	if err := r.db.Where("deleted_at IS NULL").Order("sort_order, name").Find(&categories).Error; err != nil {
		return nil, err
	}
	return r.buildTree(categories), nil
}

func (r *CategoryRepository) FindChildren(parentID uuid.UUID) ([]Category, error) {
	var children []Category
	if err := r.db.Raw(`
		WITH RECURSIVE category_tree AS (
			SELECT id, name, slug, parent_id, level FROM categories WHERE id = $1
			UNION ALL
			SELECT c.id, c.name, c.slug, c.parent_id, c.level
			FROM categories c
			INNER JOIN category_tree ct ON c.parent_id = ct.id
		)
		SELECT * FROM category_tree WHERE id != $1 ORDER BY level ASC
	`, parentID).Scan(&children).Error; err != nil {
		return nil, err
	}
	return children, nil
}

func (r *CategoryRepository) FindBreadcrumbs(categoryID uuid.UUID) ([]Category, error) {
	var breadcrumbs []Category
	if err := r.db.Raw(`
		WITH RECURSIVE category_path AS (
			SELECT id, name, slug, parent_id, level FROM categories WHERE id = $1
			UNION ALL
			SELECT c.id, c.name, c.slug, c.parent_id, c.level
			FROM categories c
			INNER JOIN category_path cp ON cp.parent_id = c.id
		)
		SELECT * FROM category_path ORDER BY level ASC
	`, categoryID).Scan(&breadcrumbs).Error; err != nil {
		return nil, err
	}
	return breadcrumbs, nil
}

func (r *CategoryRepository) buildTree(categories []Category) []Category {
	categoryMap := make(map[uuid.UUID]*Category)
	var rootIDs []uuid.UUID

	for i := range categories {
		categoryMap[categories[i].ID] = &categories[i]
	}

	for i := range categories {
		cat := &categories[i]
		if cat.ParentID != nil && *cat.ParentID != uuid.Nil {
			if parent, ok := categoryMap[*cat.ParentID]; ok {
				parent.Subcategories = append(parent.Subcategories, *cat)
			}
		} else {
			rootIDs = append(rootIDs, cat.ID)
		}
	}

	// Build roots AFTER all children have been assigned
	roots := make([]Category, len(rootIDs))
	for i, id := range rootIDs {
		roots[i] = *categoryMap[id]
	}

	return roots
}

func (r *CategoryRepository) Exists(id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&Category{}).Where("id = ? AND deleted_at IS NULL", id).Count(&count).Error
	return count > 0, err
}

// CountProductsByCategoryIDs returns a map of categoryID -> product count
// for all given category IDs in a single query.
func (r *CategoryRepository) CountProductsByCategoryIDs(categoryIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	if len(categoryIDs) == 0 {
		return make(map[uuid.UUID]int), nil
	}

	type categoryCount struct {
		CategoryID uuid.UUID
		Count      int
	}

	var results []categoryCount
	if err := r.db.Raw(`
		SELECT category_id, COUNT(*) as count
		FROM products
		WHERE category_id IN ? AND deleted_at IS NULL
		GROUP BY category_id
	`, categoryIDs).Scan(&results).Error; err != nil {
		return nil, err
	}

	counts := make(map[uuid.UUID]int)
	for _, r := range results {
		counts[r.CategoryID] = r.Count
	}
	// Ensure all requested IDs are present (zero if no products)
	for _, id := range categoryIDs {
		if _, ok := counts[id]; !ok {
			counts[id] = 0
		}
	}

	return counts, nil
}

// ProductRepository
type ProductRepository struct {
	db          *gorm.DB
	variantRepo *ProductVariantRepository
	imageRepo   *ProductImageRepository
	cache       *cache.CacheService
	metrics     *cache.CacheMetrics
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

func NewProductRepositoryWithCache(db *gorm.DB, variantRepo *ProductVariantRepository, imageRepo *ProductImageRepository, cacheSvc *cache.CacheService, metrics *cache.CacheMetrics) *ProductRepository {
	return &ProductRepository{
		db:          db,
		variantRepo: variantRepo,
		imageRepo:   imageRepo,
		cache:       cacheSvc,
		metrics:     metrics,
	}
}

func (r *ProductRepository) Create(product *Product) error {
	if err := r.db.Create(product).Error; err != nil {
		log.Printf("ERROR: Failed to create product %s: %v", product.Name, err)
		return err
	}
	return nil
}

func (r *ProductRepository) FindByID(id uuid.UUID) (*Product, error) {
	if r.cache != nil {
		key := r.cache.Key("cache", "product", id.String())
		var product Product
		hit, err := r.cache.Get(context.Background(), key, &product)
		if err != nil {
			log.Printf("Cache error: %v", err)
			r.metrics.Error()
		}
		if hit {
			r.metrics.Hit()
			return &product, nil
		}
		r.metrics.Miss()
	}

	var product Product
	if err := r.db.Preload("Category").Preload("Variants.Attribute").Preload("Variants.Images").Preload("Images").First(&product, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find product by id %s: %v", id.String(), err)
		return nil, err
	}

	if r.cache != nil {
		key := r.cache.Key("cache", "product", product.ID.String())
		if err := r.cache.Set(context.Background(), key, product); err != nil {
			log.Printf("Cache set error: %v", err)
		}
	}

	return &product, nil
}

// GetPriceByID returns only the BasePrice of a product (optimized for order creation)
func (r *ProductRepository) GetPriceByID(id uuid.UUID) (float64, error) {
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
	if r.cache != nil {
		key := r.cache.Key("cache", "product", "slug", slug)
		var product Product
		hit, err := r.cache.Get(context.Background(), key, &product)
		if err != nil {
			log.Printf("Cache error: %v", err)
			r.metrics.Error()
		}
		if hit {
			r.metrics.Hit()
			return &product, nil
		}
		r.metrics.Miss()
	}

	var product Product
	if err := r.db.Preload("Category").Preload("Variants.Attribute").Preload("Variants.Images").Preload("Images").Where("slug = ?", slug).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find product by slug %s: %v", slug, err)
		return nil, err
	}

	if r.cache != nil {
		key := r.cache.Key("cache", "product", "slug", slug)
		if err := r.cache.Set(context.Background(), key, product); err != nil {
			log.Printf("Cache set error: %v", err)
		}
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

func (r *ProductRepository) Delete(id uuid.UUID) error {
	if err := r.db.Delete(&Product{}, id).Error; err != nil {
		log.Printf("ERROR: Failed to delete product %s: %v", id.String(), err)
		return err
	}
	return nil
}

func (r *ProductRepository) FindAll(offset, limit int) ([]Product, error) {
	if r.cache != nil && offset == 0 && limit > 0 && limit <= 100 {
		key := r.cache.Key("cache", "product", "list", fmt.Sprintf("%d", offset), fmt.Sprintf("%d", limit))
		var products []Product
		hit, err := r.cache.Get(context.Background(), key, &products)
		if err != nil {
			log.Printf("Cache error: %v", err)
			r.metrics.Error()
		}
		if hit {
			r.metrics.Hit()
			return products, nil
		}
		r.metrics.Miss()
	}

	var products []Product
	query := r.db.Preload("Category").Preload("Images")
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}
	if err := query.Find(&products).Error; err != nil {
		log.Printf("ERROR: Failed to find all products: %v", err)
		return nil, err
	}

	if r.cache != nil && offset == 0 && limit > 0 && limit <= 100 {
		key := r.cache.Key("cache", "product", "list", fmt.Sprintf("%d", offset), fmt.Sprintf("%d", limit))
		if err := r.cache.Set(context.Background(), key, products); err != nil {
			log.Printf("Cache set error: %v", err)
		}
	}

	return products, nil
}

func (r *ProductRepository) FindByCategoryID(categoryID uuid.UUID, offset, limit int) ([]Product, error) {
	if r.cache != nil && offset == 0 && limit > 0 && limit <= 100 {
		key := r.cache.Key("cache", "product", "list", fmt.Sprintf("%d", offset), fmt.Sprintf("%d", limit), categoryID.String())
		var products []Product
		hit, err := r.cache.Get(context.Background(), key, &products)
		if err != nil {
			log.Printf("Cache error: %v", err)
			r.metrics.Error()
		}
		if hit {
			r.metrics.Hit()
			return products, nil
		}
		r.metrics.Miss()
	}

	var products []Product
	query := r.db.Where("category_id = ?", categoryID).Preload("Category").Preload("Images")
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}
	if err := query.Find(&products).Error; err != nil {
		log.Printf("ERROR: Failed to find products by category %s: %v", categoryID.String(), err)
		return nil, err
	}

	if r.cache != nil && offset == 0 && limit > 0 && limit <= 100 {
		key := r.cache.Key("cache", "product", "list", fmt.Sprintf("%d", offset), fmt.Sprintf("%d", limit), categoryID.String())
		if err := r.cache.Set(context.Background(), key, products); err != nil {
			log.Printf("Cache set error: %v", err)
		}
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

func (r *ProductRepository) FindByIDWithRelationsParallel(id uuid.UUID) (*ProductWithRelations, error) {
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
	db      *gorm.DB
	cache   *cache.CacheService
	metrics *cache.CacheMetrics
}

func NewProductVariantRepository(db *gorm.DB) *ProductVariantRepository {
	return &ProductVariantRepository{db: db}
}

func NewProductVariantRepositoryWithCache(db *gorm.DB, cacheSvc *cache.CacheService, metrics *cache.CacheMetrics) *ProductVariantRepository {
	return &ProductVariantRepository{db: db, cache: cacheSvc, metrics: metrics}
}

func (r *ProductVariantRepository) Create(variant *ProductVariant) error {
	return r.db.Create(variant).Error
}

func (r *ProductVariantRepository) FindByID(id uuid.UUID) (*ProductVariant, error) {
	if r.cache != nil {
		key := r.cache.Key("cache", "variant", id.String())
		var variant ProductVariant
		hit, err := r.cache.Get(context.Background(), key, &variant)
		if err != nil {
			log.Printf("Cache error: %v", err)
			r.metrics.Error()
		}
		if hit {
			r.metrics.Hit()
			return &variant, nil
		}
		r.metrics.Miss()
	}

	var variant ProductVariant
	if err := r.db.Preload("Attribute").Preload("Images").First(&variant, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find variant by id %s: %v", id.String(), err)
		return nil, err
	}

	if r.cache != nil {
		key := r.cache.Key("cache", "variant", variant.ID.String())
		if err := r.cache.Set(context.Background(), key, variant); err != nil {
			log.Printf("Cache set error: %v", err)
		}
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

func (r *ProductVariantRepository) Delete(id uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("variant_id = ?", id).Delete(&ProductVariantAttribute{}).Error; err != nil {
			return fmt.Errorf("failed to delete variant attributes: %w", err)
		}

		if err := tx.Where("variant_id = ?", id).Delete(&ProductImage{}).Error; err != nil {
			return fmt.Errorf("failed to delete variant images: %w", err)
		}

		if err := tx.Delete(&ProductVariant{}, id).Error; err != nil {
			return fmt.Errorf("failed to delete variant: %w", err)
		}

		return nil
	})
}

func (r *ProductVariantRepository) FindByProductID(productID uuid.UUID) ([]ProductVariant, error) {
	if r.cache != nil {
		key := r.cache.Key("cache", "variant", "product", productID.String())
		var variants []ProductVariant
		hit, err := r.cache.Get(context.Background(), key, &variants)
		if err != nil {
			log.Printf("Cache error: %v", err)
			r.metrics.Error()
		}
		if hit {
			r.metrics.Hit()
			return variants, nil
		}
		r.metrics.Miss()
	}

	var variants []ProductVariant
	if err := r.db.Where("product_id = ?", productID).Preload("Attribute").Preload("Images").Find(&variants).Error; err != nil {
		log.Printf("ERROR: Failed to find variants by product %s: %v", productID.String(), err)
		return nil, err
	}

	if r.cache != nil {
		key := r.cache.Key("cache", "variant", "product", productID.String())
		if err := r.cache.Set(context.Background(), key, variants); err != nil {
			log.Printf("Cache set error: %v", err)
		}
	}

	return variants, nil
}

func (r *ProductVariantRepository) FindAll() ([]ProductVariant, error) {
	var variants []ProductVariant
	if err := r.db.Preload("Attribute").Preload("Images").Find(&variants).Error; err != nil {
		return nil, err
	}
	return variants, nil
}

func (r *ProductVariantRepository) UpdateStock(id uuid.UUID, stock int) error {
	return r.db.Model(&ProductVariant{}).Where("id = ?", id).Update("stock", stock).Error
}

// GetPriceAndStock returns price and available stock for a variant
func (r *ProductVariantRepository) GetPriceAndStock(id uuid.UUID) (float64, int, int, error) {
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
func (r *ProductVariantRepository) ReserveStock(id uuid.UUID, quantity int) error {
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
func (r *ProductVariantRepository) ReleaseStock(id uuid.UUID, quantity int) error {
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
func (r *ProductVariantRepository) ConfirmSale(id uuid.UUID, quantity int) error {
	return r.db.Model(&ProductVariant{}).
		Where("id = ? AND reserved >= ?", id, quantity).
		Update("reserved", gorm.Expr("reserved - ?", quantity)).Error
}

// ProductVariantAttribute CRUD
func (r *ProductVariantRepository) CreateAttribute(attribute *ProductVariantAttribute) error {
	return r.db.Create(attribute).Error
}

func (r *ProductVariantRepository) FindAttributeByVariantID(variantID uuid.UUID) (*ProductVariantAttribute, error) {
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

func (r *ProductVariantRepository) DeleteAttribute(variantID uuid.UUID) error {
	return r.db.Where("variant_id = ?", variantID).Delete(&ProductVariantAttribute{}).Error
}

// ProductImageRepository
type ProductImageRepository struct {
	db      *gorm.DB
	cache   *cache.CacheService
	metrics *cache.CacheMetrics
}

func NewProductImageRepository(db *gorm.DB) *ProductImageRepository {
	return &ProductImageRepository{db: db}
}

func NewProductImageRepositoryWithCache(db *gorm.DB, cacheSvc *cache.CacheService, metrics *cache.CacheMetrics) *ProductImageRepository {
	return &ProductImageRepository{db: db, cache: cacheSvc, metrics: metrics}
}

func (r *ProductImageRepository) Create(image *ProductImage) error {
	return r.db.Create(image).Error
}

func (r *ProductImageRepository) FindByID(id uuid.UUID) (*ProductImage, error) {
	if r.cache != nil {
		key := r.cache.Key("cache", "image", id.String())
		var image ProductImage
		hit, err := r.cache.Get(context.Background(), key, &image)
		if err != nil {
			log.Printf("Cache error: %v", err)
			r.metrics.Error()
		}
		if hit {
			r.metrics.Hit()
			return &image, nil
		}
		r.metrics.Miss()
	}

	var image ProductImage
	if err := r.db.First(&image, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Printf("ERROR: Failed to find image by id %s: %v", id.String(), err)
		return nil, err
	}

	if r.cache != nil {
		key := r.cache.Key("cache", "image", image.ID.String())
		if err := r.cache.Set(context.Background(), key, image); err != nil {
			log.Printf("Cache set error: %v", err)
		}
	}

	return &image, nil
}

func (r *ProductImageRepository) Update(image *ProductImage) error {
	return r.db.Save(image).Error
}

func (r *ProductImageRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&ProductImage{}, id).Error
}

func (r *ProductImageRepository) FindByProductID(productID uuid.UUID) ([]ProductImage, error) {
	if r.cache != nil {
		key := r.cache.Key("cache", "image", "product", productID.String())
		var images []ProductImage
		hit, err := r.cache.Get(context.Background(), key, &images)
		if err != nil {
			log.Printf("Cache error: %v", err)
			r.metrics.Error()
		}
		if hit {
			r.metrics.Hit()
			return images, nil
		}
		r.metrics.Miss()
	}

	var images []ProductImage
	if err := r.db.Where("product_id = ?", productID).Order("sort_order ASC").Find(&images).Error; err != nil {
		log.Printf("ERROR: Failed to find images by product %s: %v", productID.String(), err)
		return nil, err
	}

	if r.cache != nil {
		key := r.cache.Key("cache", "image", "product", productID.String())
		if err := r.cache.Set(context.Background(), key, images); err != nil {
			log.Printf("Cache set error: %v", err)
		}
	}

	return images, nil
}

func (r *ProductImageRepository) FindByVariantID(variantID uuid.UUID) ([]ProductImage, error) {
	var images []ProductImage
	if err := r.db.Where("variant_id = ?", variantID).Order("sort_order ASC").Find(&images).Error; err != nil {
		return nil, err
	}
	return images, nil
}

func (r *ProductImageRepository) SetMainImage(productID uuid.UUID, imageID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&ProductImage{}).Where("product_id = ?", productID).Update("is_main", false).Error; err != nil {
			return err
		}

		if err := tx.Model(&ProductImage{}).Where("id = ? AND product_id = ?", imageID, productID).Update("is_main", true).Error; err != nil {
			return err
		}

		return nil
	})
}
