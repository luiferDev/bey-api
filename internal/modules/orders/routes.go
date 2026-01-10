package orders

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	handler := NewOrderHandler(db)

	orders := rg.Group("/orders")
	{
		orders.POST("", handler.Create)
		orders.GET("", handler.List)
		orders.GET("/:id", handler.GetByID)
		orders.PATCH("/:id/status", handler.UpdateStatus)
	}
}
