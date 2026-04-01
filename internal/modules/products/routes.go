package products

import (
	"bey/internal/modules/inventory"
	"bey/internal/shared/cache"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.RouterGroup, db *gorm.DB) {
	SetupRoutesWithService(router, db, nil, nil, nil)
}

func SetupRoutesWithService(router *gin.RouterGroup, db *gorm.DB, productService *ProductService, authMiddleware gin.HandlerFunc, adminMiddleware gin.HandlerFunc) {
	categoryRepo := NewCategoryRepository(db)
	productRepo := NewProductRepository(db)
	variantRepo := NewProductVariantRepository(db)
	imageRepo := NewProductImageRepository(db)
	inventoryRepo := inventory.NewInventoryRepository(db)

	handler := NewProductHandlerWithInventory(categoryRepo, productRepo, variantRepo, imageRepo, inventoryRepo)

	categories := router.Group("/categories")
	{
		categories.GET("", handler.GetCategories)
		categories.GET("/:id", handler.GetCategory)
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

	categories := router.Group("/categories")
	{
		categories.GET("", handler.GetCategories)
		categories.GET("/:id", handler.GetCategory)
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
		products.GET("", handler.GetProducts)
		products.GET("/slug/:slug", handler.GetProductBySlug)
		products.GET("/:id", handler.GetProduct)

		products.GET("/:id/variants", handler.GetVariantsByProduct)
		products.GET("/:id/images", handler.GetImagesByProduct)

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
