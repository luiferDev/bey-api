package users

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	RegisterRoutesWithAuth(rg, db, nil, nil)
}

func RegisterRoutesWithAuth(rg *gin.RouterGroup, db *gorm.DB, authMiddleware gin.HandlerFunc, adminMiddleware gin.HandlerFunc) {
	handler := NewUserHandler(db)

	users := rg.Group("/users")
	{
		users.GET("", handler.List)
		users.GET("/:id", handler.GetByID)

		if authMiddleware != nil {
			users.PUT("/:id", authMiddleware, handler.Update)
		} else {
			users.PUT("/:id", handler.Update)
		}

		if adminMiddleware != nil {
			users.POST("", adminMiddleware, handler.Create)
			users.DELETE("/:id", adminMiddleware, handler.Delete)
		} else {
			users.POST("", handler.Create)
			users.DELETE("/:id", handler.Delete)
		}
	}
}
