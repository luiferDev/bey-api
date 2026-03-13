package inventory

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	handler := NewInventoryHandler(db)

	inventory := rg.Group("/inventory")
	{
		inventory.GET("/:product_id", handler.GetByProductID)
		inventory.PUT("/:product_id", handler.Update)
		inventory.POST("/:product_id/reserve", handler.Reserve)
		inventory.POST("/:product_id/release", handler.Release)
	}
}
