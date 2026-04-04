package products

import (
	"context"
	"errors"
	"log"
	"time"

	"bey/internal/concurrency"
	"bey/internal/shared/cache"

	"github.com/gofrs/uuid/v5"
)

type ProductService struct {
	categoryRepo *CategoryRepository
	productRepo  *ProductRepository
	variantRepo  *ProductVariantRepository
	imageRepo    *ProductImageRepository
	taskQueue    concurrency.TaskQueue
	cache        *cache.CacheService
}

func NewProductService(
	categoryRepo *CategoryRepository,
	productRepo *ProductRepository,
	variantRepo *ProductVariantRepository,
	imageRepo *ProductImageRepository,
) *ProductService {
	return &ProductService{
		categoryRepo: categoryRepo,
		productRepo:  productRepo,
		variantRepo:  variantRepo,
		imageRepo:    imageRepo,
	}
}

func NewProductServiceWithTaskQueue(
	categoryRepo *CategoryRepository,
	productRepo *ProductRepository,
	variantRepo *ProductVariantRepository,
	imageRepo *ProductImageRepository,
	taskQueue concurrency.TaskQueue,
) *ProductService {
	return &ProductService{
		categoryRepo: categoryRepo,
		productRepo:  productRepo,
		variantRepo:  variantRepo,
		imageRepo:    imageRepo,
		taskQueue:    taskQueue,
	}
}

func NewProductServiceWithCache(
	categoryRepo *CategoryRepository,
	productRepo *ProductRepository,
	variantRepo *ProductVariantRepository,
	imageRepo *ProductImageRepository,
	cacheSvc *cache.CacheService,
) *ProductService {
	return &ProductService{
		categoryRepo: categoryRepo,
		productRepo:  productRepo,
		variantRepo:  variantRepo,
		imageRepo:    imageRepo,
		cache:        cacheSvc,
	}
}

func NewProductServiceWithAllDeps(
	categoryRepo *CategoryRepository,
	productRepo *ProductRepository,
	variantRepo *ProductVariantRepository,
	imageRepo *ProductImageRepository,
	taskQueue concurrency.TaskQueue,
	cacheSvc *cache.CacheService,
) *ProductService {
	return &ProductService{
		categoryRepo: categoryRepo,
		productRepo:  productRepo,
		variantRepo:  variantRepo,
		imageRepo:    imageRepo,
		taskQueue:    taskQueue,
		cache:        cacheSvc,
	}
}

// ValidateCategory verifica que una categoría existe y está activa
func (s *ProductService) ValidateCategory(categoryID uuid.UUID) error {
	category, err := s.categoryRepo.FindByID(categoryID)
	if err != nil {
		return err
	}
	if category == nil {
		return errors.New("category not found")
	}
	return nil
}

// CreateProductWithVariants crea un producto con sus variantes e imágenes
func (s *ProductService) CreateProductWithVariants(
	productReq CreateProductRequest,
	variants []CreateProductVariantRequest,
	images []CreateProductImageRequest,
) (*Product, error) {
	categoryID, err := uuid.FromString(productReq.CategoryID)
	if err != nil {
		return nil, errors.New("invalid category ID")
	}
	if err := s.ValidateCategory(categoryID); err != nil {
		return nil, err
	}

	// Crear producto
	isActive := true
	if productReq.IsActive != nil {
		isActive = *productReq.IsActive
	}

	product := &Product{
		CategoryID:  categoryID,
		Name:        productReq.Name,
		Slug:        productReq.Slug,
		Brand:       productReq.Brand,
		Description: productReq.Description,
		BasePrice:   productReq.BasePrice,
		IsActive:    isActive,
	}

	if err := s.productRepo.Create(product); err != nil {
		return nil, err
	}

	// Crear variantes si se proporcionan
	for _, variantReq := range variants {
		variant := &ProductVariant{
			ProductID: product.ID,
			SKU:       variantReq.SKU,
			Price:     variantReq.Price,
			Stock:     variantReq.Stock,
		}
		if err := s.variantRepo.Create(variant); err != nil {
			return nil, err
		}

		// Crear atributos de la variante
		attribute := &ProductVariantAttribute{
			VariantID: variant.ID,
			Color:     variantReq.Color,
			Size:      variantReq.Size,
			Weight:    variantReq.Weight,
		}
		if err := s.variantRepo.CreateAttribute(attribute); err != nil {
			return nil, err
		}
		variant.Attribute = attribute
	}

	// Crear imágenes si se proporcionan
	for i, imageReq := range images {
		isMain := false
		if imageReq.IsMain != nil {
			isMain = *imageReq.IsMain
		} else if i == 0 { // Primera imagen como principal por defecto
			isMain = true
		}

		image := &ProductImage{
			ProductID: product.ID,
			VariantID: parseVariantID(imageReq.VariantID),
			URLImage:  imageReq.URLImage,
			IsMain:    isMain,
			SortOrder: imageReq.SortOrder,
		}
		if err := s.imageRepo.Create(image); err != nil {
			return nil, err
		}
	}

	// Invalidate cache after successful creation
	s.invalidateProductCache()

	// Recargar producto con relaciones
	return s.productRepo.FindByID(product.ID)
}

// GetProductWithDetails obtiene un producto con todas sus relaciones
func (s *ProductService) GetProductWithDetails(productID uuid.UUID) (*Product, error) {
	return s.productRepo.FindByID(productID)
}

// UpdateProductStock actualiza el stock de una variante específica
func (s *ProductService) UpdateProductStock(variantID uuid.UUID, newStock int) error {
	variant, err := s.variantRepo.FindByID(variantID)
	if err != nil {
		return err
	}
	if variant == nil {
		return errors.New("variant not found")
	}

	return s.variantRepo.UpdateStock(variantID, newStock)
}

// CheckProductAvailability verifica si un producto tiene stock disponible
func (s *ProductService) CheckProductAvailability(productID uuid.UUID) (bool, int, error) {
	variants, err := s.variantRepo.FindByProductID(productID)
	if err != nil {
		return false, 0, err
	}

	totalStock := 0
	for _, variant := range variants {
		totalStock += variant.Stock
	}

	return totalStock > 0, totalStock, nil
}

// GetProductsByCategory obtiene productos de una categoría específica con paginación
func (s *ProductService) GetProductsByCategory(categoryID uuid.UUID, offset, limit int) ([]Product, error) {
	if err := s.ValidateCategory(categoryID); err != nil {
		return nil, err
	}

	return s.productRepo.FindByCategoryID(categoryID, offset, limit)
}

// SearchProducts busca productos por nombre o descripción
func (s *ProductService) SearchProducts(query string, offset, limit int) ([]Product, error) {
	// Esta funcionalidad requeriría agregar un método de búsqueda al repositorio
	// Por ahora retornamos todos los productos activos
	return s.productRepo.FindByActive(true, offset, limit)
}

// DeactivateProduct desactiva un producto y todas sus variantes
func (s *ProductService) DeactivateProduct(productID uuid.UUID) error {
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return err
	}
	if product == nil {
		return errors.New("product not found")
	}

	product.IsActive = false
	return s.productRepo.Update(product)
}

// GetCategoryHierarchy obtiene la jerarquía completa de categorías
func (s *ProductService) GetCategoryHierarchy() ([]Category, error) {
	return s.categoryRepo.FindAll()
}

// ValidateProductSlug verifica que un slug de producto sea único
func (s *ProductService) ValidateProductSlug(slug string, excludeID *uuid.UUID) error {
	product, err := s.productRepo.FindBySlug(slug)
	if err != nil {
		return err
	}
	if product != nil && (excludeID == nil || product.ID != *excludeID) {
		return errors.New("product slug already exists")
	}
	return nil
}

// ValidateCategorySlug verifica que un slug de categoría sea único
func (s *ProductService) ValidateCategorySlug(slug string, excludeID *uuid.UUID) error {
	category, err := s.categoryRepo.FindBySlug(slug)
	if err != nil {
		return err
	}
	if category != nil && (excludeID == nil || category.ID != *excludeID) {
		return errors.New("category slug already exists")
	}
	return nil
}

// ValidateVariantSKU verifica que un SKU de variante sea único
func (s *ProductService) ValidateVariantSKU(sku string, excludeID *uuid.UUID) error {
	variant, err := s.variantRepo.FindBySKU(sku)
	if err != nil {
		return err
	}
	if variant != nil && (excludeID == nil || variant.ID != *excludeID) {
		return errors.New("variant SKU already exists")
	}
	return nil
}

// GetProductStats obtiene estadísticas de un producto
func (s *ProductService) GetProductStats(productID uuid.UUID) (map[string]interface{}, error) {
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, errors.New("product not found")
	}

	variants, err := s.variantRepo.FindByProductID(productID)
	if err != nil {
		return nil, err
	}

	images, err := s.imageRepo.FindByProductID(productID)
	if err != nil {
		return nil, err
	}

	totalStock := 0
	variantCount := len(variants)
	for _, variant := range variants {
		totalStock += variant.Stock
	}

	stats := map[string]interface{}{
		"product_id":    productID,
		"variant_count": variantCount,
		"total_stock":   totalStock,
		"image_count":   len(images),
		"is_active":     product.IsActive,
		"has_stock":     totalStock > 0,
	}

	return stats, nil
}

type BulkUpdateProductsRequest struct {
	ProductIDs []uuid.UUID            `json:"product_ids" binding:"required"`
	Updates    []UpdateProductRequest `json:"updates" binding:"required"`
}

type BulkCreateProductsRequest struct {
	Products []CreateProductRequest `json:"products" binding:"required"`
}

type BulkDeleteProductsRequest struct {
	ProductIDs []uuid.UUID `json:"product_ids" binding:"required"`
}

type BulkTaskResult struct {
	TotalProcessed int      `json:"total_processed"`
	Successful     int      `json:"successful"`
	Failed         int      `json:"failed"`
	Errors         []string `json:"errors,omitempty"`
}

func (s *ProductService) SubmitBulkUpdateTask(req BulkUpdateProductsRequest) (string, error) {
	if s.taskQueue == nil {
		return "", errors.New("task queue not configured")
	}

	task := &concurrency.Task{
		Type:    concurrency.TaskTypeBulkUpdate,
		Status:  concurrency.TaskStatusPending,
		Payload: req,
	}

	taskID, err := s.taskQueue.Submit(task)
	if err != nil {
		return "", err
	}

	go s.processBulkUpdateTask(task)

	return taskID, nil
}

func (s *ProductService) processBulkUpdateTask(task *concurrency.Task) {
	task.SetStatus(concurrency.TaskStatusRunning)
	task.SetUpdatedAt(time.Now())

	req, ok := task.Payload.(BulkUpdateProductsRequest)
	if !ok {
		task.SetStatus(concurrency.TaskStatusFailed)
		task.SetError("invalid payload type")
		task.SetUpdatedAt(time.Now())
		return
	}

	result := BulkTaskResult{
		TotalProcessed: len(req.ProductIDs),
	}

	for i, productID := range req.ProductIDs {
		if i < len(req.Updates) {
			update := req.Updates[i]
			err := s.applyProductUpdate(productID, update)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, err.Error())
			} else {
				result.Successful++
			}
		}
	}

	task.SetResult(result)
	task.SetStatus(concurrency.TaskStatusCompleted)
	task.SetUpdatedAt(time.Now())
}

func (s *ProductService) applyProductUpdate(productID uuid.UUID, update UpdateProductRequest) error {
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return err
	}
	if product == nil {
		return errors.New("product not found")
	}

	if update.CategoryID != nil {
		catID, _ := uuid.FromString(*update.CategoryID)
		product.CategoryID = catID
	}
	if update.Name != nil {
		product.Name = *update.Name
	}
	if update.Slug != nil {
		product.Slug = *update.Slug
	}
	if update.Brand != nil {
		product.Brand = *update.Brand
	}
	if update.Description != nil {
		product.Description = *update.Description
	}
	if update.BasePrice != nil {
		product.BasePrice = *update.BasePrice
	}
	if update.IsActive != nil {
		product.IsActive = *update.IsActive
	}

	return s.productRepo.Update(product)
}

func (s *ProductService) SubmitBulkCreateTask(req BulkCreateProductsRequest) (string, error) {
	if s.taskQueue == nil {
		return "", errors.New("task queue not configured")
	}

	task := &concurrency.Task{
		Type:    concurrency.TaskTypeBulkCreate,
		Status:  concurrency.TaskStatusPending,
		Payload: req,
	}

	taskID, err := s.taskQueue.Submit(task)
	if err != nil {
		return "", err
	}

	go s.processBulkCreateTask(task)

	return taskID, nil
}

func (s *ProductService) processBulkCreateTask(task *concurrency.Task) {
	task.SetStatus(concurrency.TaskStatusRunning)
	task.SetUpdatedAt(time.Now())

	req, ok := task.Payload.(BulkCreateProductsRequest)
	if !ok {
		task.SetStatus(concurrency.TaskStatusFailed)
		task.SetError("invalid payload type")
		task.SetUpdatedAt(time.Now())
		return
	}

	result := BulkTaskResult{
		TotalProcessed: len(req.Products),
	}

	for _, productReq := range req.Products {
		_, err := s.CreateProductWithVariants(productReq, nil, nil)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.Successful++
		}
	}

	task.SetResult(result)
	task.SetStatus(concurrency.TaskStatusCompleted)
	task.SetUpdatedAt(time.Now())
}

func (s *ProductService) SubmitBulkDeleteTask(req BulkDeleteProductsRequest) (string, error) {
	if s.taskQueue == nil {
		return "", errors.New("task queue not configured")
	}

	task := &concurrency.Task{
		Type:    concurrency.TaskTypeBulkDelete,
		Status:  concurrency.TaskStatusPending,
		Payload: req,
	}

	taskID, err := s.taskQueue.Submit(task)
	if err != nil {
		return "", err
	}

	go s.processBulkDeleteTask(task)

	return taskID, nil
}

func (s *ProductService) processBulkDeleteTask(task *concurrency.Task) {
	task.SetStatus(concurrency.TaskStatusRunning)
	task.SetUpdatedAt(time.Now())

	req, ok := task.Payload.(BulkDeleteProductsRequest)
	if !ok {
		task.SetStatus(concurrency.TaskStatusFailed)
		task.SetError("invalid payload type")
		task.SetUpdatedAt(time.Now())
		return
	}

	result := BulkTaskResult{
		TotalProcessed: len(req.ProductIDs),
	}

	for _, productID := range req.ProductIDs {
		err := s.productRepo.Delete(productID)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.Successful++
		}
	}

	task.SetResult(result)
	task.SetStatus(concurrency.TaskStatusCompleted)
	task.SetUpdatedAt(time.Now())
}

func (s *ProductService) GetTaskStatus(taskID string) (*concurrency.Task, error) {
	if s.taskQueue == nil {
		return nil, errors.New("task queue not configured")
	}

	return s.taskQueue.GetStatus(taskID)
}

// invalidateProductCache removes all product-related cache entries
func (s *ProductService) invalidateProductCache() {
	if s.cache == nil {
		return
	}

	ctx := context.Background()
	if err := s.cache.InvalidatePattern(ctx, "cache:product:*"); err != nil {
		log.Printf("Failed to invalidate product cache: %v", err)
	}
}

// invalidateCategoryCache removes all category-related cache entries
func (s *ProductService) invalidateCategoryCache() {
	if s.cache == nil {
		return
	}

	ctx := context.Background()
	if err := s.cache.InvalidatePattern(ctx, "cache:category:*"); err != nil {
		log.Printf("Failed to invalidate category cache: %v", err)
	}
}

// InvalidateProductCache public method for handlers to call after mutations
func (s *ProductService) InvalidateProductCache(productID uuid.UUID) {
	if s.cache == nil {
		return
	}

	ctx := context.Background()
	s.cache.Delete(ctx, s.cache.Key("cache", "product", productID.String()))
	s.invalidateProductCache()
}

// InvalidateCategoryCache public method for handlers to call after mutations
func (s *ProductService) InvalidateCategoryCache(categoryID uuid.UUID) {
	if s.cache == nil {
		return
	}

	ctx := context.Background()
	s.cache.Delete(ctx, s.cache.Key("cache", "category", categoryID.String()))
	s.invalidateCategoryCache()
}

func parseVariantID(s *string) *uuid.UUID {
	if s == nil || *s == "" {
		return nil
	}
	id, err := uuid.FromString(*s)
	if err != nil {
		return nil
	}
	return &id
}
