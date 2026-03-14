package admin

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, authMiddleware gin.HandlerFunc, adminMiddleware gin.HandlerFunc) {
	handler := NewAdminHandler(db)

	admin := rg.Group("/admin")
	{
		if adminMiddleware != nil {
			admin.POST("/users", adminMiddleware, handler.CreateUser)
		} else {
			admin.POST("/users", handler.CreateUser)
		}
	}
}
