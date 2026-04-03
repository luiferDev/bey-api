package products

import (
	"context"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"

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
		ID:        variant.ID.String(),
		ProductID: variant.ProductID.String(),
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

func productToResponse(product *Product) ProductResponse {
	resp := ProductResponse{
		ID:          product.ID.String(),
		CategoryID:  product.CategoryID.String(),
		Name:        product.Name,
		Slug:        product.Slug,
		Brand:       product.Brand,
		Description: product.Description,
		BasePrice:   product.BasePrice,
		IsActive:    product.IsActive,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}
	if product.Category.ID != uuid.Nil {
		catResp := toCategoryResponse(product.Category, "/"+product.Category.Slug, nil)
		resp.Category = &catResp
	}
	if product.Variants != nil {
		resp.Variants = make([]ProductVariantResponse, len(product.Variants))
		for i, v := range product.Variants {
			resp.Variants[i] = variantToResponse(&v)
		}
	}
	if product.Images != nil {
		resp.Images = make([]ProductImageResponse, len(product.Images))
		for i, img := range product.Images {
			resp.Images[i] = imageToResponse(&img)
		}
	}
	return resp
}

func imageToResponse(img *ProductImage) ProductImageResponse {
	resp := ProductImageResponse{
		ID:        img.ID.String(),
		ProductID: img.ProductID.String(),
		URLImage:  img.URLImage,
		IsMain:    img.IsMain,
		SortOrder: img.SortOrder,
	}
	if img.VariantID != nil {
		s := img.VariantID.String()
		resp.VariantID = &s
	}
	return resp
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

func (h *ProductHandler) CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationError(c, err.Error())
		return
	}

	category := &Category{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	}

	if req.ParentID != nil && *req.ParentID != "" {
		parentID, err := uuid.FromString(*req.ParentID)
		if err != nil {
			h.response.ValidationError(c, "Invalid parent_id format")
			return
		}
		category.ParentID = &parentID
	}

	if err := h.categoryRepo.Create(category); err != nil {
		h.response.InternalError(c, "Failed to create category")
		return
	}

	h.invalidateCategoryCache(category.ID)

	path := h.buildCategoryPath(category)
	counts, _ := h.categoryRepo.CountProductsByCategoryIDs([]uuid.UUID{category.ID})
	h.response.Created(c, toCategoryResponse(*category, path, counts))
}

func (h *ProductHandler) GetCategory(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.response.ValidationError(c, "Invalid category ID format")
		return
	}

	category, err := h.categoryRepo.FindByID(id)
	if err != nil {
		h.response.InternalError(c, "Failed to get category")
		return
	}
	if category == nil {
		h.response.NotFound(c, "Category not found")
		return
	}

	path := h.buildCategoryPath(category)
	counts, _ := h.categoryRepo.CountProductsByCategoryIDs([]uuid.UUID{category.ID})
	h.response.Success(c, toCategoryResponse(*category, path, counts))
}

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

	path := h.buildCategoryPath(category)
	counts, _ := h.categoryRepo.CountProductsByCategoryIDs([]uuid.UUID{category.ID})
	h.response.Success(c, toCategoryResponse(*category, path, counts))
}

func (h *ProductHandler) UpdateCategory(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.response.ValidationError(c, "Invalid category ID format")
		return
	}

	category, err := h.categoryRepo.FindByID(id)
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
		if *req.ParentID != "" {
			parentID, parseErr := uuid.FromString(*req.ParentID)
			if parseErr != nil {
				h.response.ValidationError(c, "Invalid parent_id format")
				return
			}
			category.ParentID = &parentID
		} else {
			category.ParentID = nil
		}
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

	h.invalidateCategoryCache(id)

	path := h.buildCategoryPath(category)
	counts, _ := h.categoryRepo.CountProductsByCategoryIDs([]uuid.UUID{category.ID})
	h.response.Success(c, toCategoryResponse(*category, path, counts))
}

func (h *ProductHandler) DeleteCategory(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.response.ValidationError(c, "Invalid category ID format")
		return
	}

	if err := h.categoryRepo.Delete(id); err != nil {
		h.response.InternalError(c, "Failed to delete category")
		return
	}

	h.invalidateCategoryCache(id)

	h.response.Success(c, gin.H{"message": "Category deleted successfully"})
}

func (h *ProductHandler) GetCategories(c *gin.Context) {
	categories, err := h.categoryRepo.FindTree()
	if err != nil {
		h.response.InternalError(c, "Failed to get categories")
		return
	}

	counts := h.getProductCounts(categories)
	h.response.Success(c, toCategoryResponseList(categories, "", counts))
}

func (h *ProductHandler) GetCategoryTree(c *gin.Context) {
	h.GetCategories(c)
}

func (h *ProductHandler) GetCategoryChildren(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.response.ValidationError(c, "invalid category ID format")
		return
	}

	exists, err := h.categoryRepo.Exists(id)
	if err != nil {
		h.response.InternalError(c, "failed to check category")
		return
	}
	if !exists {
		h.response.NotFound(c, "category not found")
		return
	}

	parent, err := h.categoryRepo.FindByID(id)
	if err != nil {
		h.response.InternalError(c, "failed to fetch parent category")
		return
	}
	basePath := h.buildCategoryPath(parent)

	children, err := h.categoryRepo.FindChildren(id)
	if err != nil {
		h.response.InternalError(c, "failed to fetch children")
		return
	}
	counts := h.getProductCounts(children)
	h.response.Success(c, toCategoryResponseList(children, basePath, counts))
}

func (h *ProductHandler) GetCategoryBreadcrumbs(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.response.ValidationError(c, "invalid category ID format")
		return
	}

	breadcrumbs, err := h.categoryRepo.FindBreadcrumbs(id)
	if err != nil {
		h.response.NotFound(c, "category not found")
		return
	}

	ids := make([]uuid.UUID, len(breadcrumbs))
	for i, bc := range breadcrumbs {
		ids[i] = bc.ID
	}
	counts, _ := h.categoryRepo.CountProductsByCategoryIDs(ids)

	// Build incremental path for each breadcrumb
	resp := make([]CategoryResponse, len(breadcrumbs))
	path := ""
	for i, bc := range breadcrumbs {
		path += "/" + bc.Slug
		resp[i] = toCategoryResponse(bc, path, counts)
	}
	h.response.Success(c, resp)
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationError(c, err.Error())
		return
	}

	categoryID, err := uuid.FromString(req.CategoryID)
	if err != nil {
		h.response.ValidationError(c, "Invalid category_id format")
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	product := &Product{
		CategoryID:  categoryID,
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

	h.response.Created(c, productToResponse(product))
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.response.ValidationError(c, "Invalid product ID format")
		return
	}

	product, err := h.productRepo.FindByID(id)
	if err != nil {
		h.response.InternalError(c, "Failed to get product")
		return
	}
	if product == nil {
		h.response.NotFound(c, "Product not found")
		return
	}

	h.response.Success(c, productToResponse(product))
}

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

	h.response.Success(c, productToResponse(product))
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.response.ValidationError(c, "Invalid product ID format")
		return
	}

	product, err := h.productRepo.FindByID(id)
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
		categoryID, parseErr := uuid.FromString(*req.CategoryID)
		if parseErr != nil {
			h.response.ValidationError(c, "Invalid category_id format")
			return
		}
		product.CategoryID = categoryID
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

	h.invalidateProductCache(id)

	h.response.Success(c, productToResponse(product))
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.response.ValidationError(c, "Invalid product ID format")
		return
	}

	if err := h.productRepo.Delete(id); err != nil {
		h.response.InternalError(c, "Failed to delete product")
		return
	}

	h.invalidateProductCache(id)

	h.response.Success(c, gin.H{"message": "Product deleted successfully"})
}

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
		catID, parseErr := uuid.FromString(categoryID)
		if parseErr != nil {
			h.response.ValidationError(c, "Invalid category ID format")
			return
		}
		products, err = h.productRepo.FindByCategoryID(catID, offset, limit)
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

	responses := make([]ProductResponse, len(products))
	for i := range products {
		responses[i] = productToResponse(&products[i])
	}
	h.response.Success(c, responses)
}

func (h *ProductHandler) CreateVariant(c *gin.Context) {
	var req CreateProductVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationError(c, err.Error())
		return
	}

	productID, err := uuid.FromString(req.ProductID)
	if err != nil {
		h.response.ValidationError(c, "Invalid product_id format")
		return
	}

	variant := &ProductVariant{
		ProductID: productID,
		SKU:       req.SKU,
		Price:     req.Price,
		Stock:     req.Stock,
		Reserved:  0,
	}

	if err := h.variantRepo.Create(variant); err != nil {
		h.response.InternalError(c, "Failed to create variant")
		return
	}

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

	if h.inventoryRepo != nil && req.Stock > 0 {
		inv, err := h.inventoryRepo.FindByProductID(productID)
		if err != nil {
			h.response.InternalError(c, "Failed to update inventory")
			return
		}

		if inv == nil {
			inv = &inventory.Inventory{
				ProductID: productID,
				Quantity:  req.Stock,
				Reserved:  0,
			}
			if err := h.inventoryRepo.Create(inv); err != nil {
				h.response.InternalError(c, "Failed to create inventory")
				return
			}
		} else {
			newQuantity := inv.Quantity + req.Stock
			if err := h.inventoryRepo.UpdateQuantity(productID, newQuantity); err != nil {
				h.response.InternalError(c, "Failed to update inventory")
				return
			}
		}
	}

	h.invalidateVariantCache(variant.ID, productID)

	h.response.Created(c, variantToResponse(variant))
}

func (h *ProductHandler) GetVariant(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.response.ValidationError(c, "Invalid variant ID format")
		return
	}

	variant, err := h.variantRepo.FindByID(id)
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

func (h *ProductHandler) UpdateVariant(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.response.ValidationError(c, "Invalid variant ID format")
		return
	}

	variant, err := h.variantRepo.FindByID(id)
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

	if req.Color != nil || req.Size != nil || req.Weight != nil {
		attribute, err := h.variantRepo.FindAttributeByVariantID(variant.ID)
		if err != nil {
			h.response.InternalError(c, "Failed to get variant attributes")
			return
		}

		if attribute == nil {
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

		if attribute.ID == uuid.Nil {
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

func (h *ProductHandler) DeleteVariant(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.response.ValidationError(c, "Invalid variant ID format")
		return
	}

	variant, err := h.variantRepo.FindByID(id)
	if err != nil {
		h.response.InternalError(c, "Failed to get variant")
		return
	}

	if err := h.variantRepo.Delete(id); err != nil {
		log.Printf("ERROR: Failed to delete variant %s: %v", id, err)
		h.response.InternalError(c, "Failed to delete variant")
		return
	}

	if variant != nil {
		h.invalidateVariantCache(variant.ID, variant.ProductID)
	}

	h.response.Success(c, gin.H{"message": "Variant deleted successfully"})
}

func (h *ProductHandler) GetVariantsByProduct(c *gin.Context) {
	productID, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.response.ValidationError(c, "Invalid product ID format")
		return
	}

	variants, err := h.variantRepo.FindByProductID(productID)
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

func (h *ProductHandler) CreateImage(c *gin.Context) {
	var req CreateProductImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationError(c, err.Error())
		return
	}

	productID, err := uuid.FromString(req.ProductID)
	if err != nil {
		h.response.ValidationError(c, "Invalid product_id format")
		return
	}

	isMain := false
	if req.IsMain != nil {
		isMain = *req.IsMain
	}

	image := &ProductImage{
		ProductID: productID,
		URLImage:  req.URLImage,
		IsMain:    isMain,
		SortOrder: req.SortOrder,
	}

	if req.VariantID != nil && *req.VariantID != "" {
		variantID, parseErr := uuid.FromString(*req.VariantID)
		if parseErr != nil {
			h.response.ValidationError(c, "Invalid variant_id format")
			return
		}
		image.VariantID = &variantID
	}

	if err := h.imageRepo.Create(image); err != nil {
		h.response.InternalError(c, "Failed to create image")
		return
	}

	h.invalidateImageCache(image.ID, productID)

	h.response.Created(c, imageToResponse(image))
}

func (h *ProductHandler) GetImage(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.response.ValidationError(c, "Invalid image ID format")
		return
	}

	image, err := h.imageRepo.FindByID(id)
	if err != nil {
		h.response.InternalError(c, "Failed to get image")
		return
	}
	if image == nil {
		h.response.NotFound(c, "Image not found")
		return
	}

	h.response.Success(c, imageToResponse(image))
}

func (h *ProductHandler) UpdateImage(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.response.ValidationError(c, "Invalid image ID format")
		return
	}

	image, err := h.imageRepo.FindByID(id)
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

	h.response.Success(c, imageToResponse(image))
}

func (h *ProductHandler) DeleteImage(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.response.ValidationError(c, "Invalid image ID format")
		return
	}

	image, err := h.imageRepo.FindByID(id)
	if err != nil {
		h.response.InternalError(c, "Failed to get image")
		return
	}

	if err := h.imageRepo.Delete(id); err != nil {
		h.response.InternalError(c, "Failed to delete image")
		return
	}

	if image != nil {
		h.invalidateImageCache(image.ID, image.ProductID)
	}

	h.response.Success(c, gin.H{"message": "Image deleted successfully"})
}

func (h *ProductHandler) GetImagesByProduct(c *gin.Context) {
	productID, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.response.ValidationError(c, "Invalid product ID format")
		return
	}

	images, err := h.imageRepo.FindByProductID(productID)
	if err != nil {
		h.response.InternalError(c, "Failed to get images")
		return
	}

	responses := make([]ProductImageResponse, len(images))
	for i, img := range images {
		responses[i] = imageToResponse(&img)
	}
	h.response.Success(c, responses)
}

func (h *ProductHandler) SetMainImage(c *gin.Context) {
	productID, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.response.ValidationError(c, "Invalid product ID format")
		return
	}

	imageID, err := uuid.FromString(c.Param("image_id"))
	if err != nil {
		h.response.ValidationError(c, "Invalid image ID format")
		return
	}

	if err := h.imageRepo.SetMainImage(productID, imageID); err != nil {
		h.response.InternalError(c, "Failed to set main image")
		return
	}

	h.invalidateProductCache(productID)

	h.response.Success(c, gin.H{"message": "Main image set successfully"})
}

func (h *ProductHandler) invalidateProductCache(productID uuid.UUID) {
	if h.cache == nil {
		return
	}
	ctx := context.Background()
	go func() {
		if err := h.cache.Delete(ctx, h.cache.Key("cache", "product", productID.String())); err != nil {
			log.Printf("Failed to invalidate product cache %s: %v", productID, err)
		}
		if err := h.cache.InvalidatePattern(ctx, "cache:product:list:*"); err != nil {
			log.Printf("Failed to invalidate product list cache: %v", err)
		}
		if err := h.cache.InvalidatePattern(ctx, "cache:product:search:*"); err != nil {
			log.Printf("Failed to invalidate product search cache: %v", err)
		}
	}()
}

func (h *ProductHandler) invalidateCategoryCache(categoryID uuid.UUID) {
	if h.cache == nil {
		return
	}
	ctx := context.Background()
	go func() {
		if err := h.cache.Delete(ctx, h.cache.Key("cache", "category", categoryID.String())); err != nil {
			log.Printf("Failed to invalidate category cache %s: %v", categoryID, err)
		}
		if err := h.cache.Delete(ctx, h.cache.Key("cache", "category", "list")); err != nil {
			log.Printf("Failed to invalidate category list cache: %v", err)
		}
		if err := h.cache.InvalidatePattern(ctx, "cache:product:list:*"); err != nil {
			log.Printf("Failed to invalidate product list cache after category mutation: %v", err)
		}
	}()
}

func (h *ProductHandler) invalidateVariantCache(variantID uuid.UUID, productID uuid.UUID) {
	if h.cache == nil {
		return
	}
	ctx := context.Background()
	go func() {
		if variantID != uuid.Nil {
			if err := h.cache.Delete(ctx, h.cache.Key("cache", "variant", variantID.String())); err != nil {
				log.Printf("Failed to invalidate variant cache %s: %v", variantID, err)
			}
		}
		if err := h.cache.Delete(ctx, h.cache.Key("cache", "variant", "product", productID.String())); err != nil {
			log.Printf("Failed to invalidate variant product cache %s: %v", productID, err)
		}
		if err := h.cache.Delete(ctx, h.cache.Key("cache", "product", productID.String())); err != nil {
			log.Printf("Failed to invalidate product cache %s: %v", productID, err)
		}
		if err := h.cache.InvalidatePattern(ctx, "cache:product:list:*"); err != nil {
			log.Printf("Failed to invalidate product list cache after variant mutation: %v", err)
		}
	}()
}

func (h *ProductHandler) invalidateImageCache(imageID uuid.UUID, productID uuid.UUID) {
	if h.cache == nil {
		return
	}
	ctx := context.Background()
	go func() {
		if imageID != uuid.Nil {
			if err := h.cache.Delete(ctx, h.cache.Key("cache", "image", imageID.String())); err != nil {
				log.Printf("Failed to invalidate image cache %s: %v", imageID, err)
			}
		}
		if err := h.cache.Delete(ctx, h.cache.Key("cache", "image", "product", productID.String())); err != nil {
			log.Printf("Failed to invalidate image product cache %s: %v", productID, err)
		}
		if err := h.cache.Delete(ctx, h.cache.Key("cache", "product", productID.String())); err != nil {
			log.Printf("Failed to invalidate product cache %s: %v", productID, err)
		}
	}()
}

func toCategoryResponse(cat Category, path string, productCounts map[uuid.UUID]int) CategoryResponse {
	resp := CategoryResponse{
		ID:           cat.ID.String(),
		Name:         cat.Name,
		Slug:         cat.Slug,
		Description:  cat.Description,
		Path:         path,
		Level:        cat.Level,
		IsActive:     cat.IsActive,
		SortOrder:    cat.SortOrder,
		ProductCount: productCounts[cat.ID],
		CreatedAt:    cat.CreatedAt,
		UpdatedAt:    cat.UpdatedAt,
	}
	if cat.ParentID != nil {
		s := cat.ParentID.String()
		resp.ParentID = &s
	}
	if cat.Subcategories != nil {
		resp.Subcategories = make([]CategoryResponse, len(cat.Subcategories))
		for i, sub := range cat.Subcategories {
			resp.Subcategories[i] = toCategoryResponse(sub, path+"/"+sub.Slug, productCounts)
		}
	}
	return resp
}

func toCategoryResponseList(categories []Category, basePath string, productCounts map[uuid.UUID]int) []CategoryResponse {
	responses := make([]CategoryResponse, len(categories))
	for i, cat := range categories {
		path := basePath + "/" + cat.Slug
		responses[i] = toCategoryResponse(cat, path, productCounts)
	}
	return responses
}

// buildCategoryPath constructs a URL-friendly path from root to the category.
// Example: /electronics/computers/laptops
func (h *ProductHandler) buildCategoryPath(category *Category) string {
	if category.ParentID == nil || *category.ParentID == uuid.Nil {
		return "/" + category.Slug
	}

	breadcrumbs, err := h.categoryRepo.FindBreadcrumbs(category.ID)
	if err != nil || len(breadcrumbs) == 0 {
		return "/" + category.Slug
	}

	path := ""
	for _, bc := range breadcrumbs {
		path += "/" + bc.Slug
	}
	return path
}

// collectCategoryIDs recursively collects all category IDs from a tree.
func collectCategoryIDs(categories []Category) []uuid.UUID {
	var ids []uuid.UUID
	for _, cat := range categories {
		ids = append(ids, cat.ID)
		ids = append(ids, collectCategoryIDs(cat.Subcategories)...)
	}
	return ids
}

// getProductCounts fetches product counts for all categories in the tree.
func (h *ProductHandler) getProductCounts(categories []Category) map[uuid.UUID]int {
	ids := collectCategoryIDs(categories)
	if len(ids) == 0 {
		return make(map[uuid.UUID]int)
	}
	counts, err := h.categoryRepo.CountProductsByCategoryIDs(ids)
	if err != nil {
		log.Printf("Failed to get product counts: %v", err)
		return make(map[uuid.UUID]int)
	}
	return counts
}
