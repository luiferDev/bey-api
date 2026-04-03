// @title Bey API
// @version 1.0
// @description E-commerce REST API with products, categories, orders, users, and inventory management
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@bey.com
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your Bearer token in the format: Bearer {token}

// @tag.name Auth
// @tag.description Authentication, authorization, 2FA, and OAuth2 endpoints
// @tag.name Users
// @tag.description User management and profile operations
// @tag.name Categories
// @tag.description Product category management
// @tag.name Products
// @tag.description Product catalog management
// @tag.name Variants
// @tag.description Product variant management (SKU, attributes)
// @tag.name Images
// @tag.description Product and variant image management
// @tag.name Orders
// @tag.description Order creation, management, and tracking
// @tag.name Inventory
// @tag.description Stock management and reservations
// @tag.name Cart
// @tag.description Shopping cart operations (Redis-backed)
// @tag.name Payments
// @tag.description Payment processing via Wompi gateway
// @tag.name Admin
// @tag.description Administrative operations
// @tag.name Health
// @tag.description Health check endpoint
// @tag.name Cache
// @tag.description Cache metrics and management endpoints

package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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
	"bey/internal/modules/auth"
	"bey/internal/modules/cart"
	"bey/internal/modules/email"
	"bey/internal/modules/inventory"
	"bey/internal/modules/orders"
	"bey/internal/modules/payments"
	"bey/internal/modules/products"
	"bey/internal/modules/users"
	"bey/internal/shared"
	"bey/internal/shared/cache"
	"bey/internal/shared/middleware"
	"bey/internal/shared/response"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	_ "bey/cmd/api/docs"
)

func generateRandomPassword(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate random password: %v", err))
	}
	return base64.URLEncoding.EncodeToString(b)[:length]
}

func seedAdminUser(db *gorm.DB, cfg *config.Config) error {
	adminEmail := cfg.GetAdminEmail()
	adminPassword := cfg.GetAdminPassword()

	if adminEmail == "" {
		adminEmail = "admin@bey.com"
	}
	if adminPassword == "" || adminPassword == "REPLACE_WITH_STRONG_RANDOM_PASSWORD" {
		log.Println("WARNING: Default admin password detected. Generating a strong random password.")
		adminPassword = generateRandomPassword(32)
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

	log.Printf("Admin user seeded successfully: %s", adminEmail)
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
		&products.ProductVariantAttribute{},
		&products.ProductImage{},
		&orders.Order{},
		&orders.OrderItem{},
		&inventory.Inventory{},
		&auth.RefreshToken{},
		&payments.Payment{},
		&payments.PaymentLink{},
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

	var redisPool *cache.RedisPool
	var cacheService *cache.CacheService
	var cacheMetrics *cache.CacheMetrics
	var cacheHandler *cache.Handler
	var cacheWarmer *cache.CacheWarmer
	if cfg.Cache.Enabled {
		redisAddr := fmt.Sprintf("%s:%d", cfg.RateLimit.Redis.Host, cfg.RateLimit.Redis.Port)
		redisPassword := cfg.RateLimit.Redis.Password
		if cfg.RateLimit.Redis.Enabled {
			redisAddr = fmt.Sprintf("%s:%d", cfg.RateLimit.Redis.Host, cfg.RateLimit.Redis.Port)
			redisPassword = cfg.RateLimit.Redis.Password
		} else if cfg.Cart.Enabled {
			redisAddr = fmt.Sprintf("%s:%d", cfg.Cart.Redis.Host, cfg.Cart.Redis.Port)
			redisPassword = cfg.Cart.Redis.Password
		}

		redisPool, err = cache.NewRedisPool(redisAddr, redisPassword, cfg.Cache.DB)
		if err != nil {
			log.Printf("Warning: Failed to initialize Redis pool: %v", err)
		} else {
			cacheMetrics = cache.NewCacheMetrics()
			cacheService = cache.NewCacheService(
				redisPool.GetClient(cfg.Cache.DB),
				time.Duration(cfg.Cache.DefaultTTL)*time.Second,
				cacheMetrics,
			)
			cacheHandler = cache.NewHandler(cacheService)
			log.Println("Redis cache pool initialized successfully")
		}
	}

	middleware.InitCORS(cfg.Security.GetAllowedOrigins())

	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20) // 10MB
		c.Next()
	})

	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.LoggerMiddleware())
	router.Use(middleware.RateLimitMiddleware(rateLimiter))

	responseHandler := response.NewResponseHandler()

	var variantRepo *products.ProductVariantRepository
	var imageRepo *products.ProductImageRepository
	if cacheService != nil && cacheMetrics != nil {
		variantRepo = products.NewProductVariantRepositoryWithCache(db.GetDB(), cacheService, cacheMetrics)
		imageRepo = products.NewProductImageRepositoryWithCache(db.GetDB(), cacheService, cacheMetrics)
		log.Println("Variant and image repositories initialized with cache")
	} else {
		variantRepo = products.NewProductVariantRepository(db.GetDB())
		imageRepo = products.NewProductImageRepository(db.GetDB())
	}

	var categoryRepo *products.CategoryRepository
	var productRepo *products.ProductRepository
	if cacheService != nil && cacheMetrics != nil {
		categoryRepo = products.NewCategoryRepositoryWithCache(db.GetDB(), cacheService, cacheMetrics)
		productRepo = products.NewProductRepositoryWithCache(db.GetDB(), variantRepo, imageRepo, cacheService, cacheMetrics)
		log.Println("Product and category repositories initialized with cache")
	} else {
		categoryRepo = products.NewCategoryRepository(db.GetDB())
		productRepo = products.NewProductRepository(db.GetDB())
	}

	inventoryRepo := inventory.NewInventoryRepository(db.GetDB())
	orderRepo := orders.NewOrderRepository(db.GetDB())
	orderService := orders.NewOrderServiceWithAllDepsAndVariant(
		orderRepo,
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

	if redisPool != nil && cacheService != nil {
		authService = auth.NewAuthServiceWithRedis(db.GetDB(), cfg, redisPool.GetClient(cfg.Cache.DB))
		log.Println("Auth service upgraded with Redis refresh tokens")
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
		users.RegisterRoutesWithAuth(api, db.GetDB(), authMiddleware.RequireAuth(), adminMiddleware)
		products.SetupRoutesWithCache(api, categoryRepo, productRepo, variantRepo, imageRepo, cacheService, authMiddleware.RequireAuth(), adminMiddleware)
		orders.RegisterRoutesWithAllDeps(api, db.GetDB(), orderService, productRepo, variantRepo, inventoryRepo, authMiddleware.RequireAuth(), adminMiddleware)
		inventory.RegisterRoutesWithAuth(api, db.GetDB(), authMiddleware.RequireAuth(), adminMiddleware)

		if cfg.Cart.Enabled {
			cartRepo, err := cart.NewRedisCartRepository(cfg.Cart)
			if err != nil {
				log.Printf("Warning: Failed to initialize cart repository: %v", err)
			} else {
				cart.SetupRoutesWithDeps(api, cartRepo, variantRepo, orderRepo, variantRepo, inventoryRepo, authMiddleware.RequireAuth())
				log.Println("Cart module initialized successfully")
			}
		}

		if cfg.Wompi.Enabled {
			paymentRepo := payments.NewPaymentRepository(db.GetDB())
			paymentLinkRepo := payments.NewPaymentLinkRepository(db.GetDB())
			paymentService := payments.NewPaymentService(&cfg.Wompi, paymentRepo, paymentLinkRepo, orderService)
			payments.SetupRoutes(api, db.GetDB(), paymentService, authMiddleware.RequireAuth())
			log.Println("Payments module initialized successfully")
		}
	}

	if cfg.App.SwaggerEnabled {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	if cfg.App.StaticPath != "" {
		router.Static("/dashboard", cfg.App.StaticPath)
	}

	if cacheHandler != nil {
		router.GET("/metrics/cache", cacheHandler.GetMetrics)
		router.POST("/metrics/cache/reset", cacheHandler.ResetMetrics)
	}

	if cacheService != nil && cfg.Cache.WarmingEnabled {
		cacheWarmer = cache.NewCacheWarmer(
			cacheService,
			products.NewCategoryWarmerAdapter(categoryRepo),
			products.NewProductWarmerAdapter(productRepo),
			products.NewVariantWarmerAdapter(variantRepo),
			cfg.Cache.WarmingProductLimit,
		)
	}

	// Health godoc
	// @Summary Health check
	// @Description Checks the health status of the API and its dependencies (database, worker pool, cache)
	// @Tags Health
	// @Accept json
	// @Produce json
	// @Success 200 {object} response.ApiResponse "API is healthy"
	// @Success 503 {object} response.ApiResponse "API is unhealthy - one or more dependencies failed"
	// @Router /health [get]
	router.GET("/health", func(c *gin.Context) {
		health := shared.PerformHealthCheck(
			db.GetDB(),
			cfg.Concurrency.WorkerPool.WorkerPoolSize,
			0,
			true,
			redisPool,
		)

		if health.Status == "unhealthy" {
			c.JSON(http.StatusServiceUnavailable, health)
			return
		}
		responseHandler.Success(c, health)
	})

	if cfg.App.Mode == "debug" {
		go func() {
			log.Printf("pprof server starting on 127.0.0.1:%d", cfg.App.Port+1)
			log.Println(http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", cfg.App.Port+1), nil))
		}()
	}

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

	if cacheWarmer != nil {
		go func() {
			time.Sleep(2 * time.Second)
			cacheWarmer.Warm(context.Background())
		}()
		log.Println("Cache warmer scheduled to start after 2 seconds")
	}

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
