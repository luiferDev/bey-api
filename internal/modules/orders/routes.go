package orders

import (
	products "bey/internal/modules/products"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProductPriceFinder interface {
	GetPriceByID(id uint) (float64, error)
}

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	RegisterRoutesWithAllDeps(rg, db, nil, nil, nil)
}

func RegisterRoutesWithService(rg *gin.RouterGroup, db *gorm.DB, orderService *OrderService) {
	RegisterRoutesWithAllDeps(rg, db, orderService, nil, nil)
}

func RegisterRoutesWithServiceAndProductRepo(rg *gin.RouterGroup, db *gorm.DB, orderService *OrderService, productRepo ProductPriceFinder) {
	RegisterRoutesWithAllDeps(rg, db, orderService, productRepo, nil)
}

func RegisterRoutesWithAllDeps(rg *gin.RouterGroup, db *gorm.DB, orderService *OrderService, productRepo ProductPriceFinder, variantRepo VariantStockHandler) {
	handler := NewOrderHandlerWithAllDeps(db, orderService, productRepo, variantRepo)

	orders := rg.Group("/orders")
	{
		orders.POST("", handler.Create)
		orders.GET("", handler.List)
		orders.GET("/:id", handler.GetByID)
		orders.PATCH("/:id/status", handler.UpdateStatus)
		orders.POST("/:id/confirm", handler.Confirm)
		orders.POST("/:id/cancel", handler.Cancel)
		orders.GET("/tasks/:task_id", handler.GetTaskStatus)
	}
}

// RegisterRoutesWithProductAndVariant registers routes with product and variant repos
func RegisterRoutesWithProductAndVariant(rg *gin.RouterGroup, db *gorm.DB, orderService *OrderService, productRepo ProductPriceFinder, variantRepo *products.ProductVariantRepository) {
	RegisterRoutesWithAllDeps(rg, db, orderService, productRepo, variantRepo)
}
