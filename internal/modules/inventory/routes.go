package inventory

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	RegisterRoutesWithAuth(rg, db, nil, nil)
}

func RegisterRoutesWithAuth(rg *gin.RouterGroup, db *gorm.DB, authMiddleware gin.HandlerFunc, adminMiddleware gin.HandlerFunc) {
	handler := NewInventoryHandler(db)

	inventory := rg.Group("/inventory")
	{
		if authMiddleware != nil {
			inventory.GET("/:product_id", authMiddleware, handler.GetByProductID)
		} else {
			inventory.GET("/:product_id", handler.GetByProductID)
		}

		if adminMiddleware != nil {
			inventory.PUT("/:product_id", adminMiddleware, handler.Update)
			inventory.POST("/:product_id/reserve", adminMiddleware, handler.Reserve)
			inventory.POST("/:product_id/release", adminMiddleware, handler.Release)
		} else {
			inventory.PUT("/:product_id", handler.Update)
			inventory.POST("/:product_id/reserve", handler.Reserve)
			inventory.POST("/:product_id/release", handler.Release)
		}
	}
}
