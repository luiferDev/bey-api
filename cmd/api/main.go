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
	"bey/internal/modules/admin"
	"bey/internal/modules/auth"
	"bey/internal/modules/email"
	"bey/internal/modules/inventory"
	"bey/internal/modules/orders"
	"bey/internal/modules/products"
	"bey/internal/modules/users"
	"bey/internal/shared"
	"bey/internal/shared/middleware"
	"bey/internal/shared/response"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	_ "bey/cmd/api/docs"
)

func seedAdminUser(db *gorm.DB, cfg *config.Config) error {
	adminEmail := cfg.GetAdminEmail()
	adminPassword := cfg.GetAdminPassword()

	if adminEmail == "" {
		adminEmail = "admin@bey.com"
	}
	if adminPassword == "" {
		adminPassword = "admin123"
	}

	var count int64
	db.Model(&users.User{}).Where("email = ?", adminEmail).Count(&count)
	if count > 0 {
		log.Println("Admin user already exists, skipping seed")
		return nil
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	adminUser := &users.User{
		Email:     adminEmail,
		Password:  string(hashedPassword),
		FirstName: "Admin",
		LastName:  "User",
		Role:      "admin",
		Active:    true,
	}

	if err := db.Create(adminUser).Error; err != nil {
		return fmt.Errorf("failed to seed admin user: %w", err)
	}

	log.Printf("Admin user seeded successfully: %s / %s", adminEmail, adminPassword)
	return nil
}

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Concurrency.WorkerPool.WorkerPoolSize <= 0 {
		log.Fatalf("Invalid worker_pool_size: %d (must be > 0)", cfg.Concurrency.WorkerPool.WorkerPoolSize)
	}

	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	if err := db.AutoMigrate(
		&users.User{},
		&products.Category{},
		&products.Product{},
		&products.ProductVariant{},
		&products.ProductImage{},
		&orders.Order{},
		&orders.OrderItem{},
		&inventory.Inventory{},
		&auth.RefreshToken{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	if err := seedAdminUser(db.GetDB(), cfg); err != nil {
		log.Printf("Warning: Failed to seed admin user: %v", err)
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

	rateLimiterConfig := cfg.GetRateLimitConfig()
	rateLimiter := middleware.NewRateLimiterWithStorage(
		cfg.Concurrency.RateLimit,
		rateLimiterConfig,
		nil,
	)

	middleware.InitCORS(cfg.Security.GetAllowedOrigins())

	router := gin.Default()

	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.LoggerMiddleware())
	router.Use(middleware.RateLimitMiddleware(rateLimiter))

	responseHandler := response.NewResponseHandler()

	categoryRepo := products.NewCategoryRepository(db.GetDB())
	productRepo := products.NewProductRepository(db.GetDB())
	variantRepo := products.NewProductVariantRepository(db.GetDB())
	imageRepo := products.NewProductImageRepository(db.GetDB())
	productService := products.NewProductServiceWithTaskQueue(categoryRepo, productRepo, variantRepo, imageRepo, taskQueue)

	inventoryRepo := inventory.NewInventoryRepository(db.GetDB())
	orderService := orders.NewOrderServiceWithAllDepsAndVariant(
		orders.NewOrderRepository(db.GetDB()),
		taskQueue,
		productRepo,
		inventoryRepo,
		variantRepo,
	)

	authService := auth.NewAuthService(db.GetDB(), cfg)

	emailService, err := email.NewEmailService(cfg)
	if err != nil {
		log.Printf("Warning: Failed to initialize email service: %v", err)
	} else {
		authService = auth.NewAuthServiceWithEmail(db.GetDB(), cfg, emailService)
		log.Println("Email service initialized successfully")
	}
	authMiddleware := auth.NewAuthMiddleware(authService, cfg)
	adminMiddleware := func(c *gin.Context) {
		authMiddleware.RequireAuth()(c)
		if c.IsAborted() {
			return
		}
		middleware.RequireRole(middleware.RoleAdmin)(c)
	}

	api := router.Group("/api/v1")
	{
		auth.RegisterRoutes(router, authService, cfg)
		admin.RegisterRoutes(api, db.GetDB(), authMiddleware.RequireAuth(), adminMiddleware)
		users.RegisterRoutesWithAuth(api, db.GetDB(), authMiddleware.RequireAuth(), adminMiddleware)
		products.SetupRoutesWithService(api, db.GetDB(), productService, authMiddleware.RequireAuth(), adminMiddleware)
		orders.RegisterRoutesWithAllDeps(api, db.GetDB(), orderService, productRepo, variantRepo, authMiddleware.RequireAuth(), adminMiddleware)
		inventory.RegisterRoutesWithAuth(api, db.GetDB(), authMiddleware.RequireAuth(), adminMiddleware)
	}

	if cfg.App.SwaggerEnabled {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	if cfg.App.StaticPath != "" {
		router.Static("/dashboard", cfg.App.StaticPath)
	}

	router.GET("/health", func(c *gin.Context) {
		health := shared.PerformHealthCheck(
			db.GetDB(),
			cfg.Concurrency.WorkerPool.WorkerPoolSize,
			0,
			true,
		)

		if health.Status == "unhealthy" {
			c.JSON(http.StatusServiceUnavailable, health)
			return
		}
		responseHandler.Success(c, health)
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
