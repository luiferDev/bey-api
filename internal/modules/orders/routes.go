package orders

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	RegisterRoutesWithService(rg, db, nil)
}

func RegisterRoutesWithService(rg *gin.RouterGroup, db *gorm.DB, orderService *OrderService) {
	var handler *OrderHandler
	if orderService != nil {
		handler = NewOrderHandlerWithService(db, orderService)
	} else {
		handler = NewOrderHandler(db)
	}

	orders := rg.Group("/orders")
	{
		orders.POST("", handler.Create)
		orders.GET("", handler.List)
		orders.GET("/:id", handler.GetByID)
		orders.PATCH("/:id/status", handler.UpdateStatus)
		orders.GET("/tasks/:task_id", handler.GetTaskStatus)
	}
}
