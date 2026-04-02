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
	RegisterRoutesWithAllDeps(rg, db, nil, nil, nil, nil, nil, nil)
}

func RegisterRoutesWithService(rg *gin.RouterGroup, db *gorm.DB, orderService *OrderService) {
	RegisterRoutesWithAllDeps(rg, db, orderService, nil, nil, nil, nil, nil)
}

func RegisterRoutesWithServiceAndProductRepo(rg *gin.RouterGroup, db *gorm.DB, orderService *OrderService, productRepo ProductPriceFinder) {
	RegisterRoutesWithAllDeps(rg, db, orderService, productRepo, nil, nil, nil, nil)
}

func RegisterRoutesWithAllDeps(rg *gin.RouterGroup, db *gorm.DB, orderService *OrderService, productRepo ProductPriceFinder, variantRepo VariantStockHandler, inventoryRepo InventoryHandler, authMiddleware gin.HandlerFunc, adminMiddleware gin.HandlerFunc) {
	handler := NewOrderHandlerWithAllDeps(db, orderService, productRepo, variantRepo, inventoryRepo)

	orders := rg.Group("/orders")
	{
		// User: Create order (authenticated)
		if authMiddleware != nil {
			orders.POST("", authMiddleware, handler.Create)
		} else {
			orders.POST("", handler.Create)
		}

		// Admin only: List all orders
		if adminMiddleware != nil {
			orders.GET("", adminMiddleware, handler.List)
		} else {
			orders.GET("", handler.List)
		}

		// Authenticated: Get own order, confirm, cancel, task status
		if authMiddleware != nil {
			orders.GET("/:id", authMiddleware, handler.GetByID)
			orders.POST("/:id/confirm", authMiddleware, handler.Confirm)
			orders.POST("/:id/cancel", authMiddleware, handler.Cancel)
			orders.GET("/tasks/:task_id", authMiddleware, handler.GetTaskStatus)
		} else {
			orders.GET("/:id", handler.GetByID)
			orders.POST("/:id/confirm", handler.Confirm)
			orders.POST("/:id/cancel", handler.Cancel)
			orders.GET("/tasks/:task_id", handler.GetTaskStatus)
		}

		// Admin only: Update order status
		if adminMiddleware != nil {
			orders.PATCH("/:id/status", adminMiddleware, handler.UpdateStatus)
		} else {
			orders.PATCH("/:id/status", handler.UpdateStatus)
		}
	}
}

func RegisterRoutesWithProductAndVariant(rg *gin.RouterGroup, db *gorm.DB, orderService *OrderService, productRepo ProductPriceFinder, variantRepo *products.ProductVariantRepository, inventoryRepo InventoryHandler) {
	RegisterRoutesWithAllDeps(rg, db, orderService, productRepo, variantRepo, inventoryRepo, nil, nil)
}
