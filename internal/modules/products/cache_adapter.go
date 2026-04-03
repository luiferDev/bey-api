package products

import (
	"context"

	"github.com/gofrs/uuid/v5"
)

// categoryWarmerAdapter adapts CategoryRepository to cache.CategoryWarmer interface
type categoryWarmerAdapter struct {
	repo *CategoryRepository
}

// NewCategoryWarmerAdapter creates an adapter for cache warming
func NewCategoryWarmerAdapter(repo *CategoryRepository) *categoryWarmerAdapter {
	return &categoryWarmerAdapter{repo: repo}
}

func (a *categoryWarmerAdapter) FindAll(ctx context.Context) ([]interface{}, error) {
	categories, err := a.repo.FindAll()
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(categories))
	for i, cat := range categories {
		result[i] = cat
	}
	return result, nil
}

// productWarmerAdapter adapts ProductRepository to cache.ProductWarmer interface
type productWarmerAdapter struct {
	repo *ProductRepository
}

// NewProductWarmerAdapter creates an adapter for cache warming
func NewProductWarmerAdapter(repo *ProductRepository) *productWarmerAdapter {
	return &productWarmerAdapter{repo: repo}
}

func (a *productWarmerAdapter) FindActive(ctx context.Context, offset, limit int) ([]interface{}, error) {
	products, err := a.repo.FindByActive(true, offset, limit)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(products))
	for i, prod := range products {
		result[i] = prod
	}
	return result, nil
}

// variantWarmerAdapter adapts ProductVariantRepository to cache.VariantWarmer interface
type variantWarmerAdapter struct {
	repo *ProductVariantRepository
}

// NewVariantWarmerAdapter creates an adapter for cache warming
func NewVariantWarmerAdapter(repo *ProductVariantRepository) *variantWarmerAdapter {
	return &variantWarmerAdapter{repo: repo}
}

func (a *variantWarmerAdapter) FindByProduct(ctx context.Context, productID uuid.UUID) ([]interface{}, error) {
	variants, err := a.repo.FindByProductID(productID)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(variants))
	for i, v := range variants {
		result[i] = v
	}
	return result, nil
}
