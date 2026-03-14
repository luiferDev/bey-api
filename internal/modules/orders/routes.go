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
	RegisterRoutesWithAllDeps(rg, db, nil, nil, nil, nil, nil)
}

func RegisterRoutesWithService(rg *gin.RouterGroup, db *gorm.DB, orderService *OrderService) {
	RegisterRoutesWithAllDeps(rg, db, orderService, nil, nil, nil, nil)
}

func RegisterRoutesWithServiceAndProductRepo(rg *gin.RouterGroup, db *gorm.DB, orderService *OrderService, productRepo ProductPriceFinder) {
	RegisterRoutesWithAllDeps(rg, db, orderService, productRepo, nil, nil, nil)
}

func RegisterRoutesWithAllDeps(rg *gin.RouterGroup, db *gorm.DB, orderService *OrderService, productRepo ProductPriceFinder, variantRepo VariantStockHandler, authMiddleware gin.HandlerFunc, adminMiddleware gin.HandlerFunc) {
	handler := NewOrderHandlerWithAllDeps(db, orderService, productRepo, variantRepo)

	orders := rg.Group("/orders")
	{
		if authMiddleware != nil {
			orders.POST("", authMiddleware, handler.Create)
			orders.GET("", authMiddleware, handler.List)
			orders.GET("/:id", authMiddleware, handler.GetByID)
			orders.PATCH("/:id/status", authMiddleware, handler.UpdateStatus)
			orders.POST("/:id/confirm", authMiddleware, handler.Confirm)
			orders.POST("/:id/cancel", authMiddleware, handler.Cancel)
			orders.GET("/tasks/:task_id", authMiddleware, handler.GetTaskStatus)
		} else {
			orders.POST("", handler.Create)
			orders.GET("", handler.List)
			orders.GET("/:id", handler.GetByID)
			orders.PATCH("/:id/status", handler.UpdateStatus)
			orders.POST("/:id/confirm", handler.Confirm)
			orders.POST("/:id/cancel", handler.Cancel)
			orders.GET("/tasks/:task_id", handler.GetTaskStatus)
		}
	}
}

func RegisterRoutesWithProductAndVariant(rg *gin.RouterGroup, db *gorm.DB, orderService *OrderService, productRepo ProductPriceFinder, variantRepo *products.ProductVariantRepository) {
	RegisterRoutesWithAllDeps(rg, db, orderService, productRepo, variantRepo, nil, nil)
}
