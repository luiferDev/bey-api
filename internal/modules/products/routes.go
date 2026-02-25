package products

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.RouterGroup, db *gorm.DB) {
	SetupRoutesWithService(router, db, nil)
}

func SetupRoutesWithService(router *gin.RouterGroup, db *gorm.DB, productService *ProductService) {
	// Inicializar repositorios
	categoryRepo := NewCategoryRepository(db)
	productRepo := NewProductRepository(db)
	variantRepo := NewProductVariantRepository(db)
	imageRepo := NewProductImageRepository(db)

	// Inicializar handler
	handler := NewProductHandler(categoryRepo, productRepo, variantRepo, imageRepo)

	// Rutas de categorías
	categories := router.Group("/categories")
	{
		categories.POST("", handler.CreateCategory)
		categories.GET("", handler.GetCategories)
		categories.GET("/:id", handler.GetCategory)
		categories.GET("/slug/:slug", handler.GetCategoryBySlug)
		categories.PUT("/:id", handler.UpdateCategory)
		categories.DELETE("/:id", handler.DeleteCategory)
	}

	// Rutas de productos
	products := router.Group("/products")
	{
		products.POST("", handler.CreateProduct)
		products.GET("", handler.GetProducts)
		products.GET("/slug/:slug", handler.GetProductBySlug)
		products.GET("/:id", handler.GetProduct)
		products.PUT("/:id", handler.UpdateProduct)
		products.DELETE("/:id", handler.DeleteProduct)

		// Rutas de variantes de productos
		products.POST("/:id/variants", handler.CreateVariant)
		products.GET("/:id/variants", handler.GetVariantsByProduct)

		// Rutas de imágenes de productos
		products.POST("/:id/images", handler.CreateImage)
		products.GET("/:id/images", handler.GetImagesByProduct)
		products.PUT("/:id/images/:image_id/main", handler.SetMainImage)
	}

	// Rutas de variantes (independientes)
	variants := router.Group("/variants")
	{
		variants.GET("/:id", handler.GetVariant)
		variants.PUT("/:id", handler.UpdateVariant)
		variants.DELETE("/:id", handler.DeleteVariant)
	}

	// Rutas de imágenes (independientes)
	images := router.Group("/images")
	{
		images.GET("/:id", handler.GetImage)
		images.PUT("/:id", handler.UpdateImage)
		images.DELETE("/:id", handler.DeleteImage)
	}
}
