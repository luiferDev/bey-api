package cache

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Warmer interfaces to avoid circular dependencies
type CategoryWarmer interface {
	FindAll(ctx context.Context) ([]interface{}, error)
}

type ProductWarmer interface {
	FindActive(ctx context.Context, offset, limit int) ([]interface{}, error)
}

type VariantWarmer interface {
	FindByProduct(ctx context.Context, productID uint) ([]interface{}, error)
}

// CacheWarmer handles async cache warming on startup
type CacheWarmer struct {
	cacheSvc     *CacheService
	categoryRepo CategoryWarmer
	productRepo  ProductWarmer
	variantRepo  VariantWarmer
	productLimit int
}

// NewCacheWarmer creates a new cache warmer
func NewCacheWarmer(cacheSvc *CacheService, categoryRepo CategoryWarmer, productRepo ProductWarmer, variantRepo VariantWarmer, productLimit int) *CacheWarmer {
	return &CacheWarmer{
		cacheSvc:     cacheSvc,
		categoryRepo: categoryRepo,
		productRepo:  productRepo,
		variantRepo:  variantRepo,
		productLimit: productLimit,
	}
}

// Warm runs cache warming asynchronously (non-blocking)
func (w *CacheWarmer) Warm(ctx context.Context) {
	go func() {
		start := time.Now()
		log.Println("Cache warming started")

		w.warmCategories(ctx)
		w.warmProducts(ctx)
		w.warmVariants(ctx)

		duration := time.Since(start)
		log.Printf("Cache warming completed in %v", duration)
	}()
}

func (w *CacheWarmer) warmCategories(ctx context.Context) {
	if w.categoryRepo == nil {
		log.Println("Cache warming: no category repository configured, skipping")
		return
	}

	categories, err := w.categoryRepo.FindAll(ctx)
	if err != nil {
		log.Printf("Cache warming: failed to fetch categories: %v", err)
		return
	}

	// Cache the full list
	listKey := w.cacheSvc.Key("cache", "category", "list")
	if err := w.cacheSvc.Set(ctx, listKey, categories); err != nil {
		log.Printf("Cache warming: failed to cache category list: %v", err)
	}

	// Cache each individual category
	cached := 0
	for _, cat := range categories {
		// Try to extract ID from the category (assumes map or struct with "id" field)
		var id string
		switch v := cat.(type) {
		case map[string]interface{}:
			if idVal, ok := v["id"]; ok {
				id = fmt.Sprintf("%v", idVal)
			}
		default:
			// Fallback: use index-based key if we can't extract ID
			id = fmt.Sprintf("idx_%d", cached)
		}

		if id != "" {
			key := w.cacheSvc.Key("cache", "category", id)
			if err := w.cacheSvc.Set(ctx, key, cat); err != nil {
				log.Printf("Cache warming: failed to cache category %s: %v", id, err)
			} else {
				cached++
			}
		}
	}

	log.Printf("Cache warming: %d categories cached", cached)
}

func (w *CacheWarmer) warmProducts(ctx context.Context) {
	if w.productRepo == nil {
		log.Println("Cache warming: no product repository configured, skipping")
		return
	}

	limit := w.productLimit
	if limit <= 0 {
		limit = 100
	}

	products, err := w.productRepo.FindActive(ctx, 0, limit)
	if err != nil {
		log.Printf("Cache warming: failed to fetch products: %v", err)
		return
	}

	cached := 0
	for _, prod := range products {
		var id string
		switch v := prod.(type) {
		case map[string]interface{}:
			if idVal, ok := v["id"]; ok {
				id = fmt.Sprintf("%v", idVal)
			}
		default:
			id = fmt.Sprintf("idx_%d", cached)
		}

		if id != "" {
			key := w.cacheSvc.Key("cache", "product", id)
			if err := w.cacheSvc.Set(ctx, key, prod); err != nil {
				log.Printf("Cache warming: failed to cache product %s: %v", id, err)
			} else {
				cached++
			}
		}
	}

	log.Printf("Cache warming: %d products cached", cached)
}

func (w *CacheWarmer) warmVariants(ctx context.Context) {
	if w.variantRepo == nil {
		log.Println("Cache warming: no variant repository configured, skipping")
		return
	}

	// Get all active products to warm their variants
	limit := w.productLimit
	if limit <= 0 {
		limit = 100
	}

	products, err := w.productRepo.FindActive(ctx, 0, limit)
	if err != nil {
		log.Printf("Cache warming: failed to fetch products for variant warming: %v", err)
		return
	}

	totalCached := 0
	for _, prod := range products {
		var id uint
		switch v := prod.(type) {
		case map[string]interface{}:
			if idVal, ok := v["id"]; ok {
				id = uint(idVal.(float64))
			}
		default:
			continue
		}

		if id == 0 {
			continue
		}

		variants, err := w.variantRepo.FindByProduct(ctx, id)
		if err != nil {
			log.Printf("Cache warming: failed to fetch variants for product %d: %v", id, err)
			continue
		}

		key := w.cacheSvc.Key("cache", "variant", "product", fmt.Sprintf("%d", id))
		if err := w.cacheSvc.Set(ctx, key, variants); err != nil {
			log.Printf("Cache warming: failed to cache variants for product %d: %v", id, err)
		} else {
			totalCached += len(variants)
		}
	}

	log.Printf("Cache warming: %d variants cached for %d products", totalCached, len(products))
}
