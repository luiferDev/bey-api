package main

import (
	"fmt"
	"log"

	"bey/internal/config"
	"bey/internal/database"
	"bey/internal/modules/inventory"
	"bey/internal/modules/orders"
	"bey/internal/modules/products"
	"bey/internal/modules/users"
	"bey/internal/shared/middleware"
	"bey/internal/shared/response"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.AutoMigrate(
		&users.User{},
		&products.Category{},
		&products.Product{},
		&products.ProductVariant{},
		&products.ProductImage{},
		&products.Product{},
		&orders.Order{},
		&orders.OrderItem{},
		&inventory.Inventory{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	router := gin.Default()

	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.LoggerMiddleware())

	responseHandler := response.NewResponseHandler()

	api := router.Group("/api/v1")
	{
		users.RegisterRoutes(api, db.GetDB())
		products.SetupRoutes(api, db.GetDB())
		orders.RegisterRoutes(api, db.GetDB())
		inventory.RegisterRoutes(api, db.GetDB())
	}

	router.GET("/health", func(c *gin.Context) {
		responseHandler.Success(c, gin.H{"status": "healthy"})
	})

	addr := fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port)
	log.Printf("Server starting on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
