// @title Bey API
// @version 1.0
// @description E-commerce REST API with products, categories, orders, users, and inventory management
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.example.com/support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bey/internal/concurrency"
	"bey/internal/config"
	"bey/internal/database"
	"bey/internal/modules/inventory"
	"bey/internal/modules/orders"
	"bey/internal/modules/products"
	"bey/internal/modules/users"
	"bey/internal/shared/middleware"
	"bey/internal/shared/response"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"

	_ "bey/cmd/api/docs"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Concurrency.WorkerPool.WorkerPoolSize <= 0 {
		log.Fatalf("Invalid worker_pool_size: %d (must be > 0)", cfg.Concurrency.WorkerPool.WorkerPoolSize)
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
		&orders.Order{},
		&orders.OrderItem{},
		&inventory.Inventory{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	taskQueue := concurrency.NewInMemoryTaskQueue()

	workerPool := concurrency.NewWorkerPool(
		cfg.Concurrency.WorkerPool.WorkerPoolSize,
		cfg.Concurrency.WorkerPool.QueueDepthLimit,
		nil,
	)
	if err := workerPool.Start(); err != nil {
		log.Fatalf("Failed to start worker pool: %v", err)
	}
	log.Printf("Worker pool started with %d workers", cfg.Concurrency.WorkerPool.WorkerPoolSize)

	rateLimiter := middleware.NewRateLimiter(cfg.Concurrency.RateLimit)

	router := gin.Default()

	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.LoggerMiddleware())
	router.Use(middleware.RateLimitMiddleware(rateLimiter))

	responseHandler := response.NewResponseHandler()

	orderService := orders.NewOrderServiceWithTaskQueue(orders.NewOrderRepository(db.GetDB()), taskQueue)

	categoryRepo := products.NewCategoryRepository(db.GetDB())
	productRepo := products.NewProductRepository(db.GetDB())
	variantRepo := products.NewProductVariantRepository(db.GetDB())
	imageRepo := products.NewProductImageRepository(db.GetDB())
	productService := products.NewProductServiceWithTaskQueue(categoryRepo, productRepo, variantRepo, imageRepo, taskQueue)

	// Register API routes first (higher priority)
	api := router.Group("/api/v1")
	{
		users.RegisterRoutes(api, db.GetDB())
		products.SetupRoutesWithService(api, db.GetDB(), productService)
		orders.RegisterRoutesWithService(api, db.GetDB(), orderService)
		inventory.RegisterRoutes(api, db.GetDB())
	}

	// Register swagger UI
	if cfg.App.SwaggerEnabled {
		// Use ginSwagger.WrapHandler with embedded swagger files
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// Register static files under /dashboard prefix to avoid conflict with /api
	if cfg.App.StaticPath != "" {
		router.Static("/dashboard", cfg.App.StaticPath)
	}

	router.GET("/health", func(c *gin.Context) {
		responseHandler.Success(c, gin.H{"status": "healthy"})
	})

	go func() {
		log.Printf("pprof server starting on %s:%d", cfg.App.Host, cfg.App.Port+1)
		log.Println(http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port+1), nil))
	}()

	addr := fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("Server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Shutting down worker pool...")
	if err := workerPool.Shutdown(); err != nil {
		log.Fatalf("Error shutting down worker pool: %v", err)
	}

	log.Println("Server exited")
}
