package products

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	categoryRepo *CategoryRepository
	productRepo  *ProductRepository
	variantRepo  *ProductVariantRepository
	imageRepo    *ProductImageRepository
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category := &Category{
		ParentID:    req.ParentID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	}

	if err := h.categoryRepo.Create(category); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	c.JSON(http.StatusCreated, category)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	category, err := h.categoryRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get category"})
		return
	}
	if category == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	c.JSON(http.StatusOK, category)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get category"})
		return
	}
	if category == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	c.JSON(http.StatusOK, category)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	category, err := h.categoryRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get category"})
		return
	}
	if category == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category"})
		return
	}

	c.JSON(http.StatusOK, category)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	if err := h.categoryRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category deleted successfully"})
}

// @Summary Get all categories
// @Description Retrieves a list of all categories
// @Tags Categories
// @Accept json
// @Produce json
// @Success 200 {array} Category
// @Router /api/v1/categories [get]
func (h *ProductHandler) GetCategories(c *gin.Context) {
	categories, err := h.categoryRepo.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get categories"})
		return
	}

	c.JSON(http.StatusOK, categories)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
		return
	}

	c.JSON(http.StatusCreated, product)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	product, err := h.productRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get product"})
		return
	}
	if product == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, product)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get product"})
		return
	}
	if product == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, product)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	product, err := h.productRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get product"})
		return
	}
	if product == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
		return
	}

	c.JSON(http.StatusOK, product)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	if err := h.productRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
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
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset: must be >= 0"})
		return
	}
	if limit <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit: must be > 0"})
		return
	}

	categoryID := c.Query("category_id")
	active := c.Query("active")

	var products []Product
	var err error

	if categoryID != "" {
		catID, parseErr := strconv.ParseUint(categoryID, 10, 32)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
			return
		}
		products, err = h.productRepo.FindByCategoryID(uint(catID), offset, limit)
	} else if active != "" {
		isActive, parseErr := strconv.ParseBool(active)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid active value"})
			return
		}
		products, err = h.productRepo.FindByActive(isActive, offset, limit)
	} else {
		products, err = h.productRepo.FindAll(offset, limit)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get products"})
		return
	}

	c.JSON(http.StatusOK, products)
}

// ProductVariant Handlers
// @Summary Create a product variant
// @Description Creates a new variant for a product
// @Tags Variants
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Param variant body CreateProductVariantRequest true "Variant data"
// @Success 201 {object} ProductVariant
// @Router /api/v1/products/{id}/variants [post]
func (h *ProductHandler) CreateVariant(c *gin.Context) {
	var req CreateProductVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	variant := &ProductVariant{
		ProductID:  req.ProductID,
		SKU:        req.SKU,
		Price:      req.Price,
		Stock:      req.Stock,
		Attributes: req.Attributes,
	}

	if err := h.variantRepo.Create(variant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create variant"})
		return
	}

	c.JSON(http.StatusCreated, variant)
}

// @Summary Get variant by ID
// @Description Retrieves a variant by its ID
// @Tags Variants
// @Accept json
// @Produce json
// @Param id path int true "Variant ID"
// @Success 200 {object} ProductVariant
// @Router /api/v1/variants/{id} [get]
func (h *ProductHandler) GetVariant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid variant ID"})
		return
	}

	variant, err := h.variantRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get variant"})
		return
	}
	if variant == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Variant not found"})
		return
	}

	c.JSON(http.StatusOK, variant)
}

// @Summary Update a variant
// @Description Updates an existing variant
// @Tags Variants
// @Accept json
// @Produce json
// @Param id path int true "Variant ID"
// @Param variant body UpdateProductVariantRequest true "Variant data"
// @Success 200 {object} ProductVariant
// @Router /api/v1/variants/{id} [put]
func (h *ProductHandler) UpdateVariant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid variant ID"})
		return
	}

	variant, err := h.variantRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get variant"})
		return
	}
	if variant == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Variant not found"})
		return
	}

	var req UpdateProductVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	if req.Attributes != nil {
		variant.Attributes = *req.Attributes
	}

	if err := h.variantRepo.Update(variant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update variant"})
		return
	}

	c.JSON(http.StatusOK, variant)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid variant ID"})
		return
	}

	if err := h.variantRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete variant"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Variant deleted successfully"})
}

// @Summary Get variants by product
// @Description Retrieves all variants for a specific product
// @Tags Variants
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Success 200 {array} ProductVariant
// @Router /api/v1/products/{id}/variants [get]
func (h *ProductHandler) GetVariantsByProduct(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	variants, err := h.variantRepo.FindByProductID(uint(productID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get variants"})
		return
	}

	c.JSON(http.StatusOK, variants)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create image"})
		return
	}

	c.JSON(http.StatusCreated, image)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}

	image, err := h.imageRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get image"})
		return
	}
	if image == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	c.JSON(http.StatusOK, image)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}

	image, err := h.imageRepo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get image"})
		return
	}
	if image == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	var req UpdateProductImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update image"})
		return
	}

	c.JSON(http.StatusOK, image)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}

	if err := h.imageRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete image"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Image deleted successfully"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	images, err := h.imageRepo.FindByProductID(uint(productID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get images"})
		return
	}

	c.JSON(http.StatusOK, images)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	imageID, err := strconv.ParseUint(c.Param("image_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}

	if err := h.imageRepo.SetMainImage(uint(productID), uint(imageID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set main image"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Main image set successfully"})
}
