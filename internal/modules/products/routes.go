package products

import (
	"bey/internal/modules/inventory"
	"bey/internal/shared/cache"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRoutes is a convenience wrapper for simple setups without cache.
// For production, prefer SetupRoutesWithCache.
func SetupRoutes(router *gin.RouterGroup, db *gorm.DB) {
	SetupRoutesWithService(router, db, nil, nil, nil)
}

// SetupRoutesWithService sets up routes with a custom ProductService.
// Deprecated: Use SetupRoutesWithCache for cache support.
func SetupRoutesWithService(router *gin.RouterGroup, db *gorm.DB, productService *ProductService, authMiddleware gin.HandlerFunc, adminMiddleware gin.HandlerFunc) {
	categoryRepo := NewCategoryRepository(db)
	productRepo := NewProductRepository(db)
	variantRepo := NewProductVariantRepository(db)
	imageRepo := NewProductImageRepository(db)
	inventoryRepo := inventory.NewInventoryRepository(db)

	handler := NewProductHandlerWithInventory(categoryRepo, productRepo, variantRepo, imageRepo, inventoryRepo)

	registerRoutes(router, handler, authMiddleware, adminMiddleware)
}

// SetupRoutesWithCache sets up routes with cache-enabled repositories.
func SetupRoutesWithCache(
	router *gin.RouterGroup,
	categoryRepo *CategoryRepository,
	productRepo *ProductRepository,
	variantRepo *ProductVariantRepository,
	imageRepo *ProductImageRepository,
	cacheSvc *cache.CacheService,
	authMiddleware gin.HandlerFunc,
	adminMiddleware gin.HandlerFunc,
) {
	handler := NewProductHandlerWithCache(categoryRepo, productRepo, variantRepo, imageRepo, cacheSvc)

	registerRoutes(router, handler, authMiddleware, adminMiddleware)
}

// registerRoutes registers all product module routes for the given handler.
// This is the single source of truth for route definitions — both
// SetupRoutesWithService and SetupRoutesWithCache delegate here.
func registerRoutes(
	router *gin.RouterGroup,
	handler *ProductHandler,
	authMiddleware gin.HandlerFunc,
	adminMiddleware gin.HandlerFunc,
) {
	categories := router.Group("/categories")
	{
		// Public: GET categories (specific routes before param routes)
		categories.GET("/tree", handler.GetCategoryTree)
		categories.GET("", handler.GetCategories)
		categories.GET("/:id", handler.GetCategory)
		categories.GET("/:id/children", handler.GetCategoryChildren)
		categories.GET("/:id/breadcrumbs", handler.GetCategoryBreadcrumbs)
		categories.GET("/slug/:slug", handler.GetCategoryBySlug)

		if adminMiddleware != nil {
			categories.POST("", adminMiddleware, handler.CreateCategory)
			categories.PUT("/:id", adminMiddleware, handler.UpdateCategory)
			categories.DELETE("/:id", adminMiddleware, handler.DeleteCategory)
		} else {
			categories.POST("", handler.CreateCategory)
			categories.PUT("/:id", handler.UpdateCategory)
			categories.DELETE("/:id", handler.DeleteCategory)
		}
	}

	products := router.Group("/products")
	{
		// Public: GET products
		products.GET("", handler.GetProducts)
		products.GET("/slug/:slug", handler.GetProductBySlug)
		products.GET("/:id", handler.GetProduct)

		// Public: GET variants and images
		products.GET("/:id/variants", handler.GetVariantsByProduct)
		products.GET("/:id/images", handler.GetImagesByProduct)

		// Admin only: CRUD products
		if adminMiddleware != nil {
			products.POST("", adminMiddleware, handler.CreateProduct)
			products.PUT("/:id", adminMiddleware, handler.UpdateProduct)
			products.DELETE("/:id", adminMiddleware, handler.DeleteProduct)
			products.POST("/:id/variants", adminMiddleware, handler.CreateVariant)
			products.POST("/:id/images", adminMiddleware, handler.CreateImage)
			products.PUT("/:id/images/:image_id/main", adminMiddleware, handler.SetMainImage)
		} else {
			products.POST("", handler.CreateProduct)
			products.PUT("/:id", handler.UpdateProduct)
			products.DELETE("/:id", handler.DeleteProduct)
			products.POST("/:id/variants", handler.CreateVariant)
			products.POST("/:id/images", handler.CreateImage)
			products.PUT("/:id/images/:image_id/main", handler.SetMainImage)
		}
	}

	variants := router.Group("/variants")
	{
		variants.GET("/:id", handler.GetVariant)

		if adminMiddleware != nil {
			variants.PUT("/:id", adminMiddleware, handler.UpdateVariant)
			variants.DELETE("/:id", adminMiddleware, handler.DeleteVariant)
		} else {
			variants.PUT("/:id", handler.UpdateVariant)
			variants.DELETE("/:id", handler.DeleteVariant)
		}
	}

	images := router.Group("/images")
	{
		images.GET("/:id", handler.GetImage)

		if adminMiddleware != nil {
			images.PUT("/:id", adminMiddleware, handler.UpdateImage)
			images.DELETE("/:id", adminMiddleware, handler.DeleteImage)
		} else {
			images.PUT("/:id", handler.UpdateImage)
			images.DELETE("/:id", handler.DeleteImage)
		}
	}
}
