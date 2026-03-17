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
		// Public: Register new user
		users.POST("/register", handler.Register)

		// Admin only: Register admin user
		if adminMiddleware != nil {
			users.POST("/register-admin", adminMiddleware, handler.RegisterAdmin)
		} else {
			users.POST("/register-admin", handler.RegisterAdmin)
		}

		// Admin only: List all users
		if adminMiddleware != nil {
			users.GET("", adminMiddleware, handler.List)
		} else {
			users.GET("", handler.List)
		}

		// Authenticated: Get user by ID (user themselves or admin)
		if authMiddleware != nil {
			users.GET("/:id", authMiddleware, handler.GetByID)
		} else {
			users.GET("/:id", handler.GetByID)
		}

		// Authenticated: Update user (user themselves or admin)
		if authMiddleware != nil {
			users.PUT("/:id", authMiddleware, handler.Update)
		} else {
			users.PUT("/:id", handler.Update)
		}

		// Authenticated: Update avatar (user themselves or admin)
		if authMiddleware != nil {
			users.PUT("/:id/avatar", authMiddleware, handler.UpdateAvatar)
		} else {
			users.PUT("/:id/avatar", handler.UpdateAvatar)
		}

		// Admin only: Delete user
		if adminMiddleware != nil {
			users.DELETE("/:id", adminMiddleware, handler.Delete)
		} else {
			users.DELETE("/:id", handler.Delete)
		}
	}
}
