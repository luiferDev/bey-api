package payments

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.RouterGroup, db *gorm.DB, service *PaymentService, authMiddleware gin.HandlerFunc) {
	handler := NewPaymentHandler(service)

	payments := router.Group("/payments")
	{
		payments.Use(authMiddleware)
		payments.POST("", handler.CreatePayment)
		payments.GET("/:id", handler.GetPayment)
		payments.GET("/wompi/:wompi_id/status", handler.GetPaymentStatus)
		payments.POST("/:id/void", handler.VoidPayment)
	}

	paymentLinks := router.Group("/payments/links")
	{
		paymentLinks.Use(authMiddleware)
		paymentLinks.POST("", handler.CreatePaymentLink)
		paymentLinks.GET("/:id", handler.GetPaymentLink)
		paymentLinks.PATCH("/:id/activate", handler.ActivatePaymentLink)
		paymentLinks.PATCH("/:id/deactivate", handler.DeactivatePaymentLink)
	}

	router.POST("/payments/webhook", handler.Webhook)
}
