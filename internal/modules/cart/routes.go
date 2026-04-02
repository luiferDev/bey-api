package cart

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.RouterGroup, cartRepo CartRepository, variantRepo VariantFinder, orderRepo OrderCreator, variantStockReserver VariantStockReserver, inventoryReserver InventoryReserver) {
	SetupRoutesWithDeps(router, cartRepo, variantRepo, orderRepo, variantStockReserver, inventoryReserver, nil)
}

func SetupRoutesWithDeps(router *gin.RouterGroup, cartRepo CartRepository, variantRepo VariantFinder, orderRepo OrderCreator, variantStockReserver VariantStockReserver, inventoryReserver InventoryReserver, authMiddleware gin.HandlerFunc) {
	cartService := NewCartService(cartRepo, variantRepo)
	handler := NewCartHandler(cartService, orderRepo, variantStockReserver, inventoryReserver)

	cart := router.Group("/cart")
	{
		if authMiddleware != nil {
			cart.Use(authMiddleware)
		}

		cart.GET("", handler.GetCart)
		cart.POST("/checkout", handler.Checkout)
		cart.POST("/items", handler.AddItem)
		cart.PUT("/items/:variant_id", handler.UpdateItem)
		cart.DELETE("/items/:variant_id", handler.RemoveItem)
		cart.DELETE("", handler.ClearCart)
	}
}
