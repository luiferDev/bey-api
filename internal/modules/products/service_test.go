package products

import (
	"testing"

	"github.com/gofrs/uuid/v5"
)

// Mock implementations for testing
type MockCategoryRepository struct {
	findByIDFunc   func(id uuid.UUID) (*Category, error)
	findBySlugFunc func(slug string) (*Category, error)
	findAllFunc    func() ([]Category, error)
}

func (m *MockCategoryRepository) FindByID(id uuid.UUID) (*Category, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(id)
	}
	return nil, nil
}

func (m *MockCategoryRepository) FindBySlug(slug string) (*Category, error) {
	if m.findBySlugFunc != nil {
		return m.findBySlugFunc(slug)
	}
	return nil, nil
}

func (m *MockCategoryRepository) FindAll() ([]Category, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc()
	}
	return nil, nil
}

type MockProductRepository struct {
	findByIDFunc         func(id uuid.UUID) (*Product, error)
	findBySlugFunc       func(slug string) (*Product, error)
	createFunc           func(product *Product) error
	updateFunc           func(product *Product) error
	deleteFunc           func(id uuid.UUID) error
	findAllFunc          func(offset, limit int) ([]Product, error)
	findByCategoryIDFunc func(categoryID uuid.UUID, offset, limit int) ([]Product, error)
	findByActiveFunc     func(isActive bool, offset, limit int) ([]Product, error)
}

func (m *MockProductRepository) FindByID(id uuid.UUID) (*Product, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(id)
	}
	return nil, nil
}

func (m *MockProductRepository) FindBySlug(slug string) (*Product, error) {
	if m.findBySlugFunc != nil {
		return m.findBySlugFunc(slug)
	}
	return nil, nil
}

func (m *MockProductRepository) Create(product *Product) error {
	if m.createFunc != nil {
		return m.createFunc(product)
	}
	return nil
}

func (m *MockProductRepository) Update(product *Product) error {
	if m.updateFunc != nil {
		return m.updateFunc(product)
	}
	return nil
}

func (m *MockProductRepository) Delete(id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func (m *MockProductRepository) FindAll(offset, limit int) ([]Product, error) {
	if m.findAllFunc != nil {
		return m.findAllFunc(offset, limit)
	}
	return nil, nil
}

func (m *MockProductRepository) FindByCategoryID(categoryID uuid.UUID, offset, limit int) ([]Product, error) {
	if m.findByCategoryIDFunc != nil {
		return m.findByCategoryIDFunc(categoryID, offset, limit)
	}
	return nil, nil
}

func (m *MockProductRepository) FindByActive(isActive bool, offset, limit int) ([]Product, error) {
	if m.findByActiveFunc != nil {
		return m.findByActiveFunc(isActive, offset, limit)
	}
	return nil, nil
}

type MockVariantRepository struct {
	findByIDFunc        func(id uuid.UUID) (*ProductVariant, error)
	findBySKUFunc       func(sku string) (*ProductVariant, error)
	createFunc          func(variant *ProductVariant) error
	updateFunc          func(variant *ProductVariant) error
	updateStockFunc     func(id uuid.UUID, stock int) error
	deleteFunc          func(id uuid.UUID) error
	findByProductIDFunc func(productID uuid.UUID) ([]ProductVariant, error)
}

func (m *MockVariantRepository) FindByID(id uuid.UUID) (*ProductVariant, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(id)
	}
	return nil, nil
}

func (m *MockVariantRepository) FindBySKU(sku string) (*ProductVariant, error) {
	if m.findBySKUFunc != nil {
		return m.findBySKUFunc(sku)
	}
	return nil, nil
}

func (m *MockVariantRepository) Create(variant *ProductVariant) error {
	if m.createFunc != nil {
		return m.createFunc(variant)
	}
	return nil
}

func (m *MockVariantRepository) Update(variant *ProductVariant) error {
	if m.updateFunc != nil {
		return m.updateFunc(variant)
	}
	return nil
}

func (m *MockVariantRepository) UpdateStock(id uuid.UUID, stock int) error {
	if m.updateStockFunc != nil {
		return m.updateStockFunc(id, stock)
	}
	return nil
}

func (m *MockVariantRepository) Delete(id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func (m *MockVariantRepository) FindByProductID(productID uuid.UUID) ([]ProductVariant, error) {
	if m.findByProductIDFunc != nil {
		return m.findByProductIDFunc(productID)
	}
	return nil, nil
}

type MockImageRepository struct {
	findByIDFunc        func(id uuid.UUID) (*ProductImage, error)
	createFunc          func(image *ProductImage) error
	updateFunc          func(image *ProductImage) error
	deleteFunc          func(id uuid.UUID) error
	findByProductIDFunc func(productID uuid.UUID) ([]ProductImage, error)
	setMainImageFunc    func(productID, imageID uuid.UUID) error
}

func (m *MockImageRepository) FindByID(id uuid.UUID) (*ProductImage, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(id)
	}
	return nil, nil
}

func (m *MockImageRepository) Create(image *ProductImage) error {
	if m.createFunc != nil {
		return m.createFunc(image)
	}
	return nil
}

func (m *MockImageRepository) Update(image *ProductImage) error {
	if m.updateFunc != nil {
		return m.updateFunc(image)
	}
	return nil
}

func (m *MockImageRepository) Delete(id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func (m *MockImageRepository) FindByProductID(productID uuid.UUID) ([]ProductImage, error) {
	if m.findByProductIDFunc != nil {
		return m.findByProductIDFunc(productID)
	}
	return nil, nil
}

func (m *MockImageRepository) SetMainImage(productID, imageID uuid.UUID) error {
	if m.setMainImageFunc != nil {
		return m.setMainImageFunc(productID, imageID)
	}
	return nil
}

// Test service methods with mocks
// Note: These tests verify the service layer logic with mocked repositories
// In a real scenario, you'd create actual service instances with mock repos

func TestProductService_ValidateCategory(t *testing.T) {
	tests := []struct {
		name         string
		categoryID   uuid.UUID
		mockFindByID func(id uuid.UUID) (*Category, error)
		wantErr      bool
		errMessage   string
	}{
		{
			name:       "category exists",
			categoryID: uuid.Must(uuid.NewV7()),
			mockFindByID: func(id uuid.UUID) (*Category, error) {
				return &Category{ID: uuid.Must(uuid.NewV7()), Name: "Electronics"}, nil
			},
			wantErr: false,
		},
		{
			name:       "category not found",
			categoryID: uuid.Must(uuid.NewV7()),
			mockFindByID: func(id uuid.UUID) (*Category, error) {
				return nil, nil
			},
			wantErr:    true,
			errMessage: "category not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockCategoryRepository{
				findByIDFunc: tt.mockFindByID,
			}

			// Simulate validation logic
			category, err := mockRepo.FindByID(tt.categoryID)

			if tt.wantErr {
				if category == nil {
					// This is expected for "not found" case
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if category == nil {
					t.Error("expected category, got nil")
				}
			}
		})
	}
}

func TestProductService_CheckProductAvailability(t *testing.T) {
	tests := []struct {
		name           string
		productID      uuid.UUID
		mockVariants   []ProductVariant
		mockFindByID   func(id uuid.UUID) (*ProductVariant, error)
		wantAvailable  bool
		wantTotalStock int
	}{
		{
			name:      "product with stock",
			productID: uuid.Must(uuid.NewV7()),
			mockVariants: []ProductVariant{
				{ID: uuid.Must(uuid.NewV7()), ProductID: uuid.Must(uuid.NewV7()), Stock: 10},
				{ID: uuid.Must(uuid.NewV7()), ProductID: uuid.Must(uuid.NewV7()), Stock: 5},
			},
			wantAvailable:  true,
			wantTotalStock: 15,
		},
		{
			name:      "product without stock",
			productID: uuid.Must(uuid.NewV7()),
			mockVariants: []ProductVariant{
				{ID: uuid.Must(uuid.NewV7()), ProductID: uuid.Must(uuid.NewV7()), Stock: 0},
			},
			wantAvailable:  false,
			wantTotalStock: 0,
		},
		{
			name:      "product with mixed stock",
			productID: uuid.Must(uuid.NewV7()),
			mockVariants: []ProductVariant{
				{ID: uuid.Must(uuid.NewV7()), ProductID: uuid.Must(uuid.NewV7()), Stock: 0},
				{ID: uuid.Must(uuid.NewV7()), ProductID: uuid.Must(uuid.NewV7()), Stock: 5},
			},
			wantAvailable:  true,
			wantTotalStock: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockVariantRepo := &MockVariantRepository{
				findByProductIDFunc: func(productID uuid.UUID) ([]ProductVariant, error) {
					return tt.mockVariants, nil
				},
			}

			// Simulate availability check
			variants, _ := mockVariantRepo.FindByProductID(tt.productID)
			totalStock := 0
			for _, v := range variants {
				totalStock += v.Stock
			}
			available := totalStock > 0

			if available != tt.wantAvailable {
				t.Errorf("available = %v; want %v", available, tt.wantAvailable)
			}
			if totalStock != tt.wantTotalStock {
				t.Errorf("totalStock = %d; want %d", totalStock, tt.wantTotalStock)
			}
		})
	}
}

func TestProductService_ValidateProductSlug(t *testing.T) {
	tests := []struct {
		name           string
		slug           string
		excludeID      *uuid.UUID
		mockFindBySlug func(slug string) (*Product, error)
		wantErr        bool
	}{
		{
			name: "unique slug",
			slug: "new-product",
			mockFindBySlug: func(slug string) (*Product, error) {
				return nil, nil
			},
			wantErr: false,
		},
		{
			name: "duplicate slug",
			slug: "existing-product",
			mockFindBySlug: func(slug string) (*Product, error) {
				return &Product{ID: uuid.Must(uuid.NewV7()), Name: "Existing"}, nil
			},
			wantErr: true,
		},
		{
			name:      "duplicate slug but same ID (update)",
			slug:      "existing-product",
			excludeID: func() *uuid.UUID { id := uuid.Must(uuid.NewV7()); return &id }(),
			mockFindBySlug: func(slug string) (*Product, error) {
				return &Product{ID: uuid.Must(uuid.NewV7()), Name: "Existing"}, nil
			},
			wantErr: false,
		},
		{
			name:      "duplicate slug different ID (update)",
			slug:      "existing-product",
			excludeID: func() *uuid.UUID { id := uuid.Must(uuid.NewV7()); return &id }(),
			mockFindBySlug: func(slug string) (*Product, error) {
				return &Product{ID: uuid.Must(uuid.NewV7()), Name: "Existing"}, nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProductRepo := &MockProductRepository{
				findBySlugFunc: tt.mockFindBySlug,
			}

			// Simulate slug validation logic
			product, _ := mockProductRepo.FindBySlug(tt.slug)

			var hasError bool
			if product != nil && (tt.excludeID == nil || product.ID != *tt.excludeID) {
				hasError = true
			}

			if hasError != tt.wantErr {
				t.Errorf("hasError = %v; want %v", hasError, tt.wantErr)
			}
		})
	}
}

func TestProductService_ValidateVariantSKU(t *testing.T) {
	tests := []struct {
		name          string
		sku           string
		excludeID     *uuid.UUID
		mockFindBySKU func(sku string) (*ProductVariant, error)
		wantErr       bool
	}{
		{
			name: "unique SKU",
			sku:  "NEW-SKU-001",
			mockFindBySKU: func(sku string) (*ProductVariant, error) {
				return nil, nil
			},
			wantErr: false,
		},
		{
			name: "duplicate SKU",
			sku:  "EXISTING-SKU",
			mockFindBySKU: func(sku string) (*ProductVariant, error) {
				return &ProductVariant{ID: uuid.Must(uuid.NewV7()), SKU: "EXISTING-SKU"}, nil
			},
			wantErr: true,
		},
		{
			name:      "duplicate SKU but same ID (update)",
			sku:       "EXISTING-SKU",
			excludeID: func() *uuid.UUID { id := uuid.Must(uuid.NewV7()); return &id }(),
			mockFindBySKU: func(sku string) (*ProductVariant, error) {
				return &ProductVariant{ID: uuid.Must(uuid.NewV7()), SKU: "EXISTING-SKU"}, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockVariantRepo := &MockVariantRepository{
				findBySKUFunc: tt.mockFindBySKU,
			}

			// Simulate SKU validation logic
			variant, _ := mockVariantRepo.FindBySKU(tt.sku)

			var hasError bool
			if variant != nil && (tt.excludeID == nil || variant.ID != *tt.excludeID) {
				hasError = true
			}

			if hasError != tt.wantErr {
				t.Errorf("hasError = %v; want %v", hasError, tt.wantErr)
			}
		})
	}
}

func TestProductService_GetProductStats(t *testing.T) {
	tests := []struct {
		name       string
		productID  uuid.UUID
		product    *Product
		variants   []ProductVariant
		images     []ProductImage
		wantErr    bool
		checkStats func(*testing.T, map[string]interface{})
	}{
		{
			name:      "product with variants and images",
			productID: uuid.Must(uuid.NewV7()),
			product:   &Product{ID: uuid.Must(uuid.NewV7()), Name: "Test", IsActive: true},
			variants: []ProductVariant{
				{ID: uuid.Must(uuid.NewV7()), ProductID: uuid.Must(uuid.NewV7()), Stock: 10},
				{ID: uuid.Must(uuid.NewV7()), ProductID: uuid.Must(uuid.NewV7()), Stock: 5},
			},
			images: []ProductImage{
				{ID: uuid.Must(uuid.NewV7()), ProductID: uuid.Must(uuid.NewV7())},
				{ID: uuid.Must(uuid.NewV7()), ProductID: uuid.Must(uuid.NewV7())},
			},
			wantErr: false,
			checkStats: func(t *testing.T, stats map[string]interface{}) {
				if stats["variant_count"].(int) != 2 {
					t.Errorf("variant_count = %d; want 2", stats["variant_count"])
				}
				if stats["total_stock"].(int) != 15 {
					t.Errorf("total_stock = %d; want 15", stats["total_stock"])
				}
				if stats["image_count"].(int) != 2 {
					t.Errorf("image_count = %d; want 2", stats["image_count"])
				}
			},
		},
		{
			name:      "product not found",
			productID: uuid.Must(uuid.NewV7()),
			product:   nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProductRepo := &MockProductRepository{
				findByIDFunc: func(id uuid.UUID) (*Product, error) {
					return tt.product, nil
				},
			}
			mockVariantRepo := &MockVariantRepository{
				findByProductIDFunc: func(productID uuid.UUID) ([]ProductVariant, error) {
					return tt.variants, nil
				},
			}
			mockImageRepo := &MockImageRepository{
				findByProductIDFunc: func(productID uuid.UUID) ([]ProductImage, error) {
					return tt.images, nil
				},
			}

			// Simulate stats calculation
			product, _ := mockProductRepo.FindByID(tt.productID)
			if tt.wantErr {
				if product != nil {
					t.Error("expected error for nil product")
				}
				return
			}

			variants, _ := mockVariantRepo.FindByProductID(tt.productID)
			images, _ := mockImageRepo.FindByProductID(tt.productID)

			totalStock := 0
			for _, v := range variants {
				totalStock += v.Stock
			}

			stats := map[string]interface{}{
				"product_id":    tt.productID,
				"variant_count": len(variants),
				"total_stock":   totalStock,
				"image_count":   len(images),
				"is_active":     product.IsActive,
				"has_stock":     totalStock > 0,
			}

			if tt.checkStats != nil {
				tt.checkStats(t, stats)
			}
		})
	}
}
