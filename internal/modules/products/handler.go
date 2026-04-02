package products

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"

	"bey/internal/modules/inventory"
	"bey/internal/shared/cache"
	"bey/internal/shared/response"
)

func variantToResponse(variant *ProductVariant) ProductVariantResponse {
	available := variant.Stock - variant.Reserved
	if available < 0 {
		available = 0
	}

	var attrResponse *ProductVariantAttributeResponse
	if variant.Attribute != nil {
		attrResponse = &ProductVariantAttributeResponse{
			Color:  variant.Attribute.Color,
			Size:   variant.Attribute.Size,
			Weight: variant.Attribute.Weight,
		}
	}

	return ProductVariantResponse{
		ID:        variant.ID,
		ProductID: variant.ProductID,
		SKU:       variant.SKU,
		Price:     variant.Price,
		Stock:     variant.Stock,
		Reserved:  variant.Reserved,
		Available: available,
		Attribute: attrResponse,
		CreatedAt: variant.CreatedAt,
		Images:    nil,
	}
}

type ProductHandler struct {
	categoryRepo  *CategoryRepository
	productRepo   *ProductRepository
	variantRepo   *ProductVariantRepository
	imageRepo     *ProductImageRepository
	inventoryRepo *inventory.InventoryRepository
	response      *response.ResponseHandler
	cache         *cache.CacheService
}

func NewProductHandler(
	categoryRepo *CategoryRepository,
	productRepo *ProductRepository,
	variantRepo *ProductVariantRepository,
	imageRepo *ProductImageRepository,
) *ProductHandler {
	return &ProductHandler{
		categoryRepo: categoryRepo,
		productRepo:  productRepo,
		variantRepo:  variantRepo,
		imageRepo:    imageRepo,
		response:     response.NewResponseHandler(),
	}
}

func NewProductHandlerWithInventory(
	categoryRepo *CategoryRepository,
	productRepo *ProductRepository,
	variantRepo *ProductVariantRepository,
	imageRepo *ProductImageRepository,
	inventoryRepo *inventory.InventoryRepository,
) *ProductHandler {
	return &ProductHandler{
		categoryRepo:  categoryRepo,
		productRepo:   productRepo,
		variantRepo:   variantRepo,
		imageRepo:     imageRepo,
		inventoryRepo: inventoryRepo,
		response:      response.NewResponseHandler(),
	}
}

func NewProductHandlerWithCache(
	categoryRepo *CategoryRepository,
	productRepo *ProductRepository,
	variantRepo *ProductVariantRepository,
	imageRepo *ProductImageRepository,
	cacheSvc *cache.CacheService,
) *ProductHandler {
	return &ProductHandler{
		categoryRepo: categoryRepo,
		productRepo:  productRepo,
		variantRepo:  variantRepo,
		imageRepo:    imageRepo,
		response:     response.NewResponseHandler(),
		cache:        cacheSvc,
	}
}

func NewProductHandlerWithInventoryAndCache(
	categoryRepo *CategoryRepository,
	productRepo *ProductRepository,
	variantRepo *ProductVariantRepository,
	imageRepo *ProductImageRepository,
	inventoryRepo *inventory.InventoryRepository,
	cacheSvc *cache.CacheService,
) *ProductHandler {
	return &ProductHandler{
		categoryRepo:  categoryRepo,
		productRepo:   productRepo,
		variantRepo:   variantRepo,
		imageRepo:     imageRepo,
		inventoryRepo: inventoryRepo,
		response:      response.NewResponseHandler(),
		cache:         cacheSvc,
	}
}

// @Summary Create a new category
// @Description Creates a new product category
// @Tags Categories
// @Accept json
// @Produce json
// @Param category body CreateCategoryRequest true "Category data"
// @Success 201 {object} Category
// @Router /api/v1/categories [post]
func (h *ProductHandler) CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationError(c, err.Error())
		return
	}

	category := &Category{
		ParentID:    req.ParentID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	}

	if err := h.categoryRepo.Create(category); err != nil {
		h.response.InternalError(c, "Failed to create category")
		return
	}

	h.invalidateCategoryCache(category.ID)

	h.response.Created(c, category)
}

// @Summary Get category by ID
// @Description Retrieves a category by its ID
// @Tags Categories
// @Accept json
// @Produce json
// @Param id path int true "Category ID"
// @Success 200 {object} Category
// @Router /api/v1/categories/{id} [get]
func (h *ProductHandler) GetCategory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "Invalid category ID")
		return
	}

	category, err := h.categoryRepo.FindByID(uint(id))
	if err != nil {
		h.response.InternalError(c, "Failed to get category")
		return
	}
	if category == nil {
		h.response.NotFound(c, "Category not found")
		return
	}

	h.response.Success(c, category)
}

// @Summary Get category by slug
// @Description Retrieves a category by its slug
// @Tags Categories
// @Accept json
// @Produce json
// @Param slug path string true "Category slug"
// @Success 200 {object} Category
// @Router /api/v1/categories/slug/{slug} [get]
func (h *ProductHandler) GetCategoryBySlug(c *gin.Context) {
	slug := c.Param("slug")

	category, err := h.categoryRepo.FindBySlug(slug)
	if err != nil {
		h.response.InternalError(c, "Failed to get category")
		return
	}
	if category == nil {
		h.response.NotFound(c, "Category not found")
		return
	}

	h.response.Success(c, category)
}

// @Summary Update a category
// @Description Updates an existing category
// @Tags Categories
// @Accept json
// @Produce json
// @Param id path int true "Category ID"
// @Param category body UpdateCategoryRequest true "Category data"
// @Success 200 {object} Category
// @Router /api/v1/categories/{id} [put]
func (h *ProductHandler) UpdateCategory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "Invalid category ID")
		return
	}

	category, err := h.categoryRepo.FindByID(uint(id))
	if err != nil {
		h.response.InternalError(c, "Failed to get category")
		return
	}
	if category == nil {
		h.response.NotFound(c, "Category not found")
		return
	}

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationError(c, err.Error())
		return
	}

	if req.ParentID != nil {
		category.ParentID = req.ParentID
	}
	if req.Name != nil {
		category.Name = *req.Name
	}
	if req.Slug != nil {
		category.Slug = *req.Slug
	}
	if req.Description != nil {
		category.Description = *req.Description
	}

	if err := h.categoryRepo.Update(category); err != nil {
		h.response.InternalError(c, "Failed to update category")
		return
	}

	h.invalidateCategoryCache(uint(id))

	h.response.Success(c, category)
}

// @Summary Delete a category
// @Description Deletes a category by ID
// @Tags Categories
// @Accept json
// @Produce json
// @Param id path int true "Category ID"
// @Success 200
// @Router /api/v1/categories/{id} [delete]
func (h *ProductHandler) DeleteCategory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "Invalid category ID")
		return
	}

	if err := h.categoryRepo.Delete(uint(id)); err != nil {
		h.response.InternalError(c, "Failed to delete category")
		return
	}

	h.invalidateCategoryCache(uint(id))

	h.response.Success(c, gin.H{"message": "Category deleted successfully"})
}

// @Summary Get all categories
// @Description Retrieves a list of all categories
// @Tags Categories
// @Summary Get all categories as nested tree
// @Description Returns the complete category tree with parent-child relationships
// @Tags Categories
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponse{data=[]CategoryResponse}
// @Router /api/v1/categories [get]
func (h *ProductHandler) GetCategories(c *gin.Context) {
	categories, err := h.categoryRepo.FindTree()
	if err != nil {
		h.response.InternalError(c, "Failed to get categories")
		return
	}

	h.response.Success(c, toCategoryResponseList(categories))
}

// GetCategoryTree godoc
// @Summary Get full category tree (alias for GET /categories)
// @Description Returns the complete nested category tree with all levels
// @Tags Categories
// @Produce json
// @Success 200 {object} response.ApiResponse{data=[]CategoryResponse}
// @Router /api/v1/categories/tree [get]
func (h *ProductHandler) GetCategoryTree(c *gin.Context) {
	h.GetCategories(c)
}

// GetCategoryChildren godoc
// @Summary Get direct children of a category
// @Description Returns immediate subcategories of the specified category
// @Tags Categories
// @Produce json
// @Param id path int true "Category ID"
// @Success 200 {object} response.ApiResponse{data=[]CategoryResponse}
// @Failure 400 {object} response.ApiResponse "Invalid category ID"
// @Failure 404 {object} response.ApiResponse "Category not found"
// @Router /api/v1/categories/{id}/children [get]
func (h *ProductHandler) GetCategoryChildren(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "invalid category ID")
		return
	}

	exists, err := h.categoryRepo.Exists(uint(id))
	if err != nil {
		h.response.InternalError(c, "failed to check category")
		return
	}
	if !exists {
		h.response.NotFound(c, "category not found")
		return
	}

	children, err := h.categoryRepo.FindChildren(uint(id))
	if err != nil {
		h.response.InternalError(c, "failed to fetch children")
		return
	}
	h.response.Success(c, toCategoryResponseList(children))
}

// GetCategoryBreadcrumbs godoc
// @Summary Get breadcrumbs for a category
// @Description Returns the path from root category to the specified category
// @Tags Categories
// @Produce json
// @Param id path int true "Category ID"
// @Success 200 {object} response.ApiResponse{data=[]CategoryResponse}
// @Failure 400 {object} response.ApiResponse "Invalid category ID"
// @Failure 404 {object} response.ApiResponse "Category not found"
// @Router /api/v1/categories/{id}/breadcrumbs [get]
func (h *ProductHandler) GetCategoryBreadcrumbs(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "invalid category ID")
		return
	}

	breadcrumbs, err := h.categoryRepo.FindBreadcrumbs(uint(id))
	if err != nil {
		h.response.NotFound(c, "category not found")
		return
	}
	h.response.Success(c, toCategoryResponseList(breadcrumbs))
}

// Product Handlers
// @Summary Create a new product
// @Description Creates a new product
// @Tags Products
// @Accept json
// @Produce json
// @Param product body CreateProductRequest true "Product data"
// @Success 201 {object} Product
// @Router /api/v1/products [post]
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationError(c, err.Error())
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	product := &Product{
		CategoryID:  req.CategoryID,
		Name:        req.Name,
		Slug:        req.Slug,
		Brand:       req.Brand,
		Description: req.Description,
		BasePrice:   req.BasePrice,
		IsActive:    isActive,
	}

	if err := h.productRepo.Create(product); err != nil {
		h.response.InternalError(c, "Failed to create product")
		return
	}

	h.invalidateProductCache(product.ID)

	h.response.Created(c, product)
}

// @Summary Get product by ID
// @Description Retrieves a product by its ID
// @Tags Products
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Success 200 {object} Product
// @Router /api/v1/products/{id} [get]
func (h *ProductHandler) GetProduct(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "Invalid product ID")
		return
	}

	product, err := h.productRepo.FindByID(uint(id))
	if err != nil {
		h.response.InternalError(c, "Failed to get product")
		return
	}
	if product == nil {
		h.response.NotFound(c, "Product not found")
		return
	}

	h.response.Success(c, product)
}

// @Summary Get product by slug
// @Description Retrieves a product by its slug
// @Tags Products
// @Accept json
// @Produce json
// @Param slug path string true "Product slug"
// @Success 200 {object} Product
// @Router /api/v1/products/slug/{slug} [get]
func (h *ProductHandler) GetProductBySlug(c *gin.Context) {
	slug := c.Param("slug")

	product, err := h.productRepo.FindBySlug(slug)
	if err != nil {
		h.response.InternalError(c, "Failed to get product")
		return
	}
	if product == nil {
		h.response.NotFound(c, "Product not found")
		return
	}

	h.response.Success(c, product)
}

// @Summary Update a product
// @Description Updates an existing product
// @Tags Products
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Param product body UpdateProductRequest true "Product data"
// @Success 200 {object} Product
// @Router /api/v1/products/{id} [put]
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "Invalid product ID")
		return
	}

	product, err := h.productRepo.FindByID(uint(id))
	if err != nil {
		h.response.InternalError(c, "Failed to get product")
		return
	}
	if product == nil {
		h.response.NotFound(c, "Product not found")
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationError(c, err.Error())
		return
	}

	if req.CategoryID != nil {
		product.CategoryID = *req.CategoryID
	}
	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Slug != nil {
		product.Slug = *req.Slug
	}
	if req.Brand != nil {
		product.Brand = *req.Brand
	}
	if req.Description != nil {
		product.Description = *req.Description
	}
	if req.BasePrice != nil {
		product.BasePrice = *req.BasePrice
	}
	if req.IsActive != nil {
		product.IsActive = *req.IsActive
	}

	if err := h.productRepo.Update(product); err != nil {
		h.response.InternalError(c, "Failed to update product")
		return
	}

	h.invalidateProductCache(uint(id))

	h.response.Success(c, product)
}

// @Summary Delete a product
// @Description Deletes a product by ID
// @Tags Products
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Success 200
// @Router /api/v1/products/{id} [delete]
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "Invalid product ID")
		return
	}

	if err := h.productRepo.Delete(uint(id)); err != nil {
		h.response.InternalError(c, "Failed to delete product")
		return
	}

	h.invalidateProductCache(uint(id))

	h.response.Success(c, gin.H{"message": "Product deleted successfully"})
}

// @Summary Get all products
// @Description Retrieves a list of products with optional pagination and filtering
// @Tags Products
// @Accept json
// @Produce json
// @Param offset query int false "Offset for pagination"
// @Param limit query int false "Limit for pagination"
// @Param category_id query int false "Filter by category ID"
// @Param active query bool false "Filter by active status"
// @Success 200 {array} Product
// @Router /api/v1/products [get]
func (h *ProductHandler) GetProducts(c *gin.Context) {
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "10")

	offset, offsetErr := strconv.Atoi(offsetStr)
	if offsetErr != nil {
		h.response.ValidationError(c, "Invalid offset: must be a number")
		return
	}

	limit, limitErr := strconv.Atoi(limitStr)
	if limitErr != nil {
		h.response.ValidationError(c, "Invalid limit: must be a number")
		return
	}

	if offset < 0 {
		h.response.ValidationError(c, "Invalid offset: must be >= 0")
		return
	}
	if limit <= 0 {
		h.response.ValidationError(c, "Invalid limit: must be > 0")
		return
	}

	categoryID := c.Query("category_id")
	active := c.Query("active")

	var products []Product
	var err error

	if categoryID != "" {
		catID, parseErr := strconv.ParseUint(categoryID, 10, 32)
		if parseErr != nil {
			h.response.ValidationError(c, "Invalid category ID")
			return
		}
		products, err = h.productRepo.FindByCategoryID(uint(catID), offset, limit)
	} else if active != "" {
		isActive, parseErr := strconv.ParseBool(active)
		if parseErr != nil {
			h.response.ValidationError(c, "Invalid active value")
			return
		}
		products, err = h.productRepo.FindByActive(isActive, offset, limit)
	} else {
		products, err = h.productRepo.FindAll(offset, limit)
	}

	if err != nil {
		h.response.InternalError(c, "Failed to get products")
		return
	}

	h.response.Success(c, products)
}

// ProductVariant Handlers
// @Summary Create a product variant
// @Description Creates a new variant for a product and updates inventory
// @Tags Variants
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Param variant body CreateProductVariantRequest true "Variant data"
// @Success 201 {object} ProductVariantResponse
// @Router /api/v1/products/{id}/variants [post]
func (h *ProductHandler) CreateVariant(c *gin.Context) {
	var req CreateProductVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationError(c, err.Error())
		return
	}

	variant := &ProductVariant{
		ProductID: req.ProductID,
		SKU:       req.SKU,
		Price:     req.Price,
		Stock:     req.Stock,
		Reserved:  0,
	}

	if err := h.variantRepo.Create(variant); err != nil {
		h.response.InternalError(c, "Failed to create variant")
		return
	}

	// Create variant attributes
	attribute := &ProductVariantAttribute{
		VariantID: variant.ID,
		Color:     req.Color,
		Size:      req.Size,
		Weight:    req.Weight,
	}
	if err := h.variantRepo.CreateAttribute(attribute); err != nil {
		h.response.InternalError(c, "Failed to create variant attributes")
		return
	}
	variant.Attribute = attribute

	// Update inventory with variant stock
	if h.inventoryRepo != nil && req.Stock > 0 {
		// Find existing inventory or create new one
		inv, err := h.inventoryRepo.FindByProductID(req.ProductID)
		if err != nil {
			h.response.InternalError(c, "Failed to update inventory")
			return
		}

		if inv == nil {
			// Create new inventory
			inv = &inventory.Inventory{
				ProductID: req.ProductID,
				Quantity:  req.Stock,
				Reserved:  0,
			}
			if err := h.inventoryRepo.Create(inv); err != nil {
				h.response.InternalError(c, "Failed to create inventory")
				return
			}
		} else {
			// Update existing inventory (add stock)
			newQuantity := inv.Quantity + req.Stock
			if err := h.inventoryRepo.UpdateQuantity(req.ProductID, newQuantity); err != nil {
				h.response.InternalError(c, "Failed to update inventory")
				return
			}
		}
	}

	h.invalidateVariantCache(0, req.ProductID)

	h.response.Created(c, variantToResponse(variant))
}

// @Summary Get variant by ID
// @Description Retrieves a variant by its ID
// @Tags Variants
// @Accept json
// @Produce json
// @Param id path int true "Variant ID"
// @Success 200 {object} ProductVariantResponse
// @Router /api/v1/variants/{id} [get]
func (h *ProductHandler) GetVariant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "Invalid variant ID")
		return
	}

	variant, err := h.variantRepo.FindByID(uint(id))
	if err != nil {
		h.response.InternalError(c, "Failed to get variant")
		return
	}
	if variant == nil {
		h.response.NotFound(c, "Variant not found")
		return
	}

	h.response.Success(c, variantToResponse(variant))
}

// @Summary Update a variant
// @Description Updates an existing variant
// @Tags Variants
// @Accept json
// @Produce json
// @Param id path int true "Variant ID"
// @Param variant body UpdateProductVariantRequest true "Variant data"
// @Success 200 {object} ProductVariantResponse
// @Router /api/v1/variants/{id} [put]
func (h *ProductHandler) UpdateVariant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "Invalid variant ID")
		return
	}

	variant, err := h.variantRepo.FindByID(uint(id))
	if err != nil {
		h.response.InternalError(c, "Failed to get variant")
		return
	}
	if variant == nil {
		h.response.NotFound(c, "Variant not found")
		return
	}

	var req UpdateProductVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationError(c, err.Error())
		return
	}

	if req.SKU != nil {
		variant.SKU = *req.SKU
	}
	if req.Price != nil {
		variant.Price = *req.Price
	}
	if req.Stock != nil {
		variant.Stock = *req.Stock
	}

	if err := h.variantRepo.Update(variant); err != nil {
		h.response.InternalError(c, "Failed to update variant")
		return
	}

	// Update attributes if provided
	if req.Color != nil || req.Size != nil || req.Weight != nil {
		attribute, err := h.variantRepo.FindAttributeByVariantID(variant.ID)
		if err != nil {
			h.response.InternalError(c, "Failed to get variant attributes")
			return
		}

		if attribute == nil {
			// Create new attribute if doesn't exist
			attribute = &ProductVariantAttribute{
				VariantID: variant.ID,
			}
		}

		if req.Color != nil {
			attribute.Color = *req.Color
		}
		if req.Size != nil {
			attribute.Size = *req.Size
		}
		if req.Weight != nil {
			attribute.Weight = *req.Weight
		}

		if attribute.ID == 0 {
			if err := h.variantRepo.CreateAttribute(attribute); err != nil {
				h.response.InternalError(c, "Failed to create variant attributes")
				return
			}
		} else {
			if err := h.variantRepo.UpdateAttribute(attribute); err != nil {
				h.response.InternalError(c, "Failed to update variant attributes")
				return
			}
		}
		variant.Attribute = attribute
	}

	h.invalidateVariantCache(variant.ID, variant.ProductID)

	h.response.Success(c, variantToResponse(variant))
}

// @Summary Delete a variant
// @Description Deletes a variant by ID
// @Tags Variants
// @Accept json
// @Produce json
// @Param id path int true "Variant ID"
// @Success 200
// @Router /api/v1/variants/{id} [delete]
func (h *ProductHandler) DeleteVariant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "Invalid variant ID")
		return
	}

	variant, err := h.variantRepo.FindByID(uint(id))
	if err != nil {
		h.response.InternalError(c, "Failed to get variant")
		return
	}

	if err := h.variantRepo.Delete(uint(id)); err != nil {
		h.response.InternalError(c, "Failed to delete variant")
		return
	}

	if variant != nil {
		h.invalidateVariantCache(variant.ID, variant.ProductID)
	}

	h.response.Success(c, gin.H{"message": "Variant deleted successfully"})
}

// @Summary Get variants by product
// @Description Retrieves all variants for a specific product
// @Tags Variants
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Success 200 {array} ProductVariantResponse
// @Router /api/v1/products/{id}/variants [get]
func (h *ProductHandler) GetVariantsByProduct(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "Invalid product ID")
		return
	}

	variants, err := h.variantRepo.FindByProductID(uint(productID))
	if err != nil {
		h.response.InternalError(c, "Failed to get variants")
		return
	}

	responses := make([]ProductVariantResponse, len(variants))
	for i, v := range variants {
		responses[i] = variantToResponse(&v)
	}

	h.response.Success(c, responses)
}

// ProductImage Handlers
// @Summary Create a product image
// @Description Creates a new image for a product
// @Tags Images
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Param image body CreateProductImageRequest true "Image data"
// @Success 201 {object} ProductImage
// @Router /api/v1/products/{id}/images [post]
func (h *ProductHandler) CreateImage(c *gin.Context) {
	var req CreateProductImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationError(c, err.Error())
		return
	}

	isMain := false
	if req.IsMain != nil {
		isMain = *req.IsMain
	}

	image := &ProductImage{
		ProductID: req.ProductID,
		VariantID: req.VariantID,
		URLImage:  req.URLImage,
		IsMain:    isMain,
		SortOrder: req.SortOrder,
	}

	if err := h.imageRepo.Create(image); err != nil {
		h.response.InternalError(c, "Failed to create image")
		return
	}

	h.invalidateImageCache(0, req.ProductID)

	h.response.Created(c, image)
}

// @Summary Get image by ID
// @Description Retrieves an image by its ID
// @Tags Images
// @Accept json
// @Produce json
// @Param id path int true "Image ID"
// @Success 200 {object} ProductImage
// @Router /api/v1/images/{id} [get]
func (h *ProductHandler) GetImage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "Invalid image ID")
		return
	}

	image, err := h.imageRepo.FindByID(uint(id))
	if err != nil {
		h.response.InternalError(c, "Failed to get image")
		return
	}
	if image == nil {
		h.response.NotFound(c, "Image not found")
		return
	}

	h.response.Success(c, image)
}

// @Summary Update an image
// @Description Updates an existing image
// @Tags Images
// @Accept json
// @Produce json
// @Param id path int true "Image ID"
// @Param image body UpdateProductImageRequest true "Image data"
// @Success 200 {object} ProductImage
// @Router /api/v1/images/{id} [put]
func (h *ProductHandler) UpdateImage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "Invalid image ID")
		return
	}

	image, err := h.imageRepo.FindByID(uint(id))
	if err != nil {
		h.response.InternalError(c, "Failed to get image")
		return
	}
	if image == nil {
		h.response.NotFound(c, "Image not found")
		return
	}

	var req UpdateProductImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationError(c, err.Error())
		return
	}

	if req.URLImage != nil {
		image.URLImage = *req.URLImage
	}
	if req.IsMain != nil {
		image.IsMain = *req.IsMain
	}
	if req.SortOrder != nil {
		image.SortOrder = *req.SortOrder
	}

	if err := h.imageRepo.Update(image); err != nil {
		h.response.InternalError(c, "Failed to update image")
		return
	}

	h.invalidateImageCache(image.ID, image.ProductID)

	h.response.Success(c, image)
}

// @Summary Delete an image
// @Description Deletes an image by ID
// @Tags Images
// @Accept json
// @Produce json
// @Param id path int true "Image ID"
// @Success 200
// @Router /api/v1/images/{id} [delete]
func (h *ProductHandler) DeleteImage(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "Invalid image ID")
		return
	}

	image, err := h.imageRepo.FindByID(uint(id))
	if err != nil {
		h.response.InternalError(c, "Failed to get image")
		return
	}

	if err := h.imageRepo.Delete(uint(id)); err != nil {
		h.response.InternalError(c, "Failed to delete image")
		return
	}

	if image != nil {
		h.invalidateImageCache(image.ID, image.ProductID)
	}

	h.response.Success(c, gin.H{"message": "Image deleted successfully"})
}

// @Summary Get images by product
// @Description Retrieves all images for a specific product
// @Tags Images
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Success 200 {array} ProductImage
// @Router /api/v1/products/{id}/images [get]
func (h *ProductHandler) GetImagesByProduct(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "Invalid product ID")
		return
	}

	images, err := h.imageRepo.FindByProductID(uint(productID))
	if err != nil {
		h.response.InternalError(c, "Failed to get images")
		return
	}

	h.response.Success(c, images)
}

// @Summary Set main image
// @Description Sets a product image as the main image
// @Tags Images
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Param image_id path int true "Image ID"
// @Success 200
// @Router /api/v1/products/{id}/images/{image_id}/main [put]
func (h *ProductHandler) SetMainImage(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "Invalid product ID")
		return
	}

	imageID, err := strconv.ParseUint(c.Param("image_id"), 10, 32)
	if err != nil {
		h.response.ValidationError(c, "Invalid image ID")
		return
	}

	if err := h.imageRepo.SetMainImage(uint(productID), uint(imageID)); err != nil {
		h.response.InternalError(c, "Failed to set main image")
		return
	}

	h.invalidateProductCache(uint(productID))

	h.response.Success(c, gin.H{"message": "Main image set successfully"})
}

// invalidateProductCache invalidates all product-related cache entries (non-blocking)
func (h *ProductHandler) invalidateProductCache(productID uint) {
	if h.cache == nil {
		return
	}
	ctx := context.Background()
	go func() {
		if err := h.cache.Delete(ctx, h.cache.Key("cache", "product", fmt.Sprintf("%d", productID))); err != nil {
			log.Printf("Failed to invalidate product cache %d: %v", productID, err)
		}
		if err := h.cache.InvalidatePattern(ctx, "cache:product:list:*"); err != nil {
			log.Printf("Failed to invalidate product list cache: %v", err)
		}
		if err := h.cache.InvalidatePattern(ctx, "cache:product:search:*"); err != nil {
			log.Printf("Failed to invalidate product search cache: %v", err)
		}
	}()
}

// invalidateCategoryCache invalidates all category-related cache entries (non-blocking)
func (h *ProductHandler) invalidateCategoryCache(categoryID uint) {
	if h.cache == nil {
		return
	}
	ctx := context.Background()
	go func() {
		if err := h.cache.Delete(ctx, h.cache.Key("cache", "category", fmt.Sprintf("%d", categoryID))); err != nil {
			log.Printf("Failed to invalidate category cache %d: %v", categoryID, err)
		}
		if err := h.cache.Delete(ctx, h.cache.Key("cache", "category", "list")); err != nil {
			log.Printf("Failed to invalidate category list cache: %v", err)
		}
		if err := h.cache.InvalidatePattern(ctx, "cache:product:list:*"); err != nil {
			log.Printf("Failed to invalidate product list cache after category mutation: %v", err)
		}
	}()
}

// invalidateVariantCache invalidates variant-related cache entries (non-blocking)
func (h *ProductHandler) invalidateVariantCache(variantID uint, productID uint) {
	if h.cache == nil {
		return
	}
	ctx := context.Background()
	go func() {
		if variantID > 0 {
			if err := h.cache.Delete(ctx, h.cache.Key("cache", "variant", fmt.Sprintf("%d", variantID))); err != nil {
				log.Printf("Failed to invalidate variant cache %d: %v", variantID, err)
			}
		}
		if err := h.cache.Delete(ctx, h.cache.Key("cache", "variant", "product", fmt.Sprintf("%d", productID))); err != nil {
			log.Printf("Failed to invalidate variant product cache %d: %v", productID, err)
		}
		if err := h.cache.Delete(ctx, h.cache.Key("cache", "product", fmt.Sprintf("%d", productID))); err != nil {
			log.Printf("Failed to invalidate product cache %d: %v", productID, err)
		}
		if err := h.cache.InvalidatePattern(ctx, "cache:product:list:*"); err != nil {
			log.Printf("Failed to invalidate product list cache after variant mutation: %v", err)
		}
	}()
}

// invalidateImageCache invalidates image-related cache entries (non-blocking)
func (h *ProductHandler) invalidateImageCache(imageID uint, productID uint) {
	if h.cache == nil {
		return
	}
	ctx := context.Background()
	go func() {
		if imageID > 0 {
			if err := h.cache.Delete(ctx, h.cache.Key("cache", "image", fmt.Sprintf("%d", imageID))); err != nil {
				log.Printf("Failed to invalidate image cache %d: %v", imageID, err)
			}
		}
		if err := h.cache.Delete(ctx, h.cache.Key("cache", "image", "product", fmt.Sprintf("%d", productID))); err != nil {
			log.Printf("Failed to invalidate image product cache %d: %v", productID, err)
		}
		if err := h.cache.Delete(ctx, h.cache.Key("cache", "product", fmt.Sprintf("%d", productID))); err != nil {
			log.Printf("Failed to invalidate product cache %d: %v", productID, err)
		}
	}()
}

// toCategoryResponse converts a Category to CategoryResponse
func toCategoryResponse(cat Category) CategoryResponse {
	resp := CategoryResponse{
		ID:          cat.ID,
		Name:        cat.Name,
		Slug:        cat.Slug,
		Description: cat.Description,
		ParentID:    cat.ParentID,
		Path:        cat.Path,
		Level:       cat.Level,
		IsActive:    cat.IsActive,
		SortOrder:   cat.SortOrder,
		CreatedAt:   cat.CreatedAt,
		UpdatedAt:   cat.UpdatedAt,
	}
	if cat.Subcategories != nil {
		resp.Subcategories = make([]CategoryResponse, len(cat.Subcategories))
		for i, sub := range cat.Subcategories {
			resp.Subcategories[i] = toCategoryResponse(sub)
		}
	}
	return resp
}

// toCategoryResponseList converts a slice of Categories to CategoryResponses
func toCategoryResponseList(categories []Category) []CategoryResponse {
	responses := make([]CategoryResponse, len(categories))
	for i, cat := range categories {
		responses[i] = toCategoryResponse(cat)
	}
	return responses
}
