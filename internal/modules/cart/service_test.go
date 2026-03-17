package cart

import (
	"errors"
	"testing"
	"time"

	"bey/internal/modules/products"
)

type MockCartRepository struct {
	getCartFunc    func(userID uint) (*Cart, error)
	saveCartFunc   func(cart *Cart) error
	deleteCartFunc func(userID uint) error
}

func (m *MockCartRepository) GetCart(userID uint) (*Cart, error) {
	if m.getCartFunc != nil {
		return m.getCartFunc(userID)
	}
	return &Cart{UserID: userID, Items: []CartItem{}}, nil
}

func (m *MockCartRepository) SaveCart(cart *Cart) error {
	if m.saveCartFunc != nil {
		return m.saveCartFunc(cart)
	}
	return nil
}

func (m *MockCartRepository) DeleteCart(userID uint) error {
	if m.deleteCartFunc != nil {
		return m.deleteCartFunc(userID)
	}
	return nil
}

type MockVariantFinder struct {
	findByIDFunc         func(id uint) (*products.ProductVariant, error)
	getPriceAndStockFunc func(id uint) (float64, int, int, error)
}

func (m *MockVariantFinder) FindByID(id uint) (*products.ProductVariant, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(id)
	}
	return nil, nil
}

func (m *MockVariantFinder) GetPriceAndStock(id uint) (float64, int, int, error) {
	if m.getPriceAndStockFunc != nil {
		return m.getPriceAndStockFunc(id)
	}
	return 0.0, 0, 0, nil
}

func TestCartService_AddItem(t *testing.T) {
	tests := []struct {
		name              string
		userID            uint
		variantID         uint
		quantity          int
		mockCartRepo      *MockCartRepository
		mockVariantFinder *MockVariantFinder
		wantErr           bool
		errType           error
	}{
		{
			name:      "add item with sufficient stock - success",
			userID:    1,
			variantID: 1,
			quantity:  2,
			mockCartRepo: &MockCartRepository{
				getCartFunc: func(userID uint) (*Cart, error) {
					return &Cart{UserID: userID, Items: []CartItem{}}, nil
				},
				saveCartFunc: func(cart *Cart) error {
					return nil
				},
			},
			mockVariantFinder: &MockVariantFinder{
				findByIDFunc: func(id uint) (*products.ProductVariant, error) {
					return &products.ProductVariant{ID: 1, ProductID: 1}, nil
				},
				getPriceAndStockFunc: func(id uint) (float64, int, int, error) {
					return 10.0, 10, 0, nil
				},
			},
			wantErr: false,
		},
		{
			name:      "add item with insufficient stock - failure",
			userID:    1,
			variantID: 1,
			quantity:  15,
			mockCartRepo: &MockCartRepository{
				getCartFunc: func(userID uint) (*Cart, error) {
					return &Cart{UserID: userID, Items: []CartItem{}}, nil
				},
			},
			mockVariantFinder: &MockVariantFinder{
				findByIDFunc: func(id uint) (*products.ProductVariant, error) {
					return &products.ProductVariant{ID: 1, ProductID: 1}, nil
				},
				getPriceAndStockFunc: func(id uint) (float64, int, int, error) {
					return 10.0, 10, 0, nil
				},
			},
			wantErr: true,
			errType: ErrInsufficientStock,
		},
		{
			name:      "add item with non-existent variant - failure",
			userID:    1,
			variantID: 999,
			quantity:  2,
			mockCartRepo: &MockCartRepository{
				getCartFunc: func(userID uint) (*Cart, error) {
					return &Cart{UserID: userID, Items: []CartItem{}}, nil
				},
			},
			mockVariantFinder: &MockVariantFinder{
				findByIDFunc: func(id uint) (*products.ProductVariant, error) {
					return nil, nil
				},
			},
			wantErr: true,
			errType: ErrVariantNotFound,
		},
		{
			name:      "add item that already exists in cart - updates quantity",
			userID:    1,
			variantID: 1,
			quantity:  3,
			mockCartRepo: &MockCartRepository{
				getCartFunc: func(userID uint) (*Cart, error) {
					return &Cart{
						UserID: userID,
						Items:  []CartItem{{VariantID: 1, Quantity: 2}},
					}, nil
				},
				saveCartFunc: func(cart *Cart) error {
					return nil
				},
			},
			mockVariantFinder: &MockVariantFinder{
				findByIDFunc: func(id uint) (*products.ProductVariant, error) {
					return &products.ProductVariant{ID: 1, ProductID: 1}, nil
				},
				getPriceAndStockFunc: func(id uint) (float64, int, int, error) {
					return 10.0, 10, 0, nil
				},
			},
			wantErr: false,
		},
		{
			name:      "add item exceeds stock when updating existing",
			userID:    1,
			variantID: 1,
			quantity:  9,
			mockCartRepo: &MockCartRepository{
				getCartFunc: func(userID uint) (*Cart, error) {
					return &Cart{
						UserID: userID,
						Items:  []CartItem{{VariantID: 1, Quantity: 2}},
					}, nil
				},
			},
			mockVariantFinder: &MockVariantFinder{
				findByIDFunc: func(id uint) (*products.ProductVariant, error) {
					return &products.ProductVariant{ID: 1, ProductID: 1}, nil
				},
				getPriceAndStockFunc: func(id uint) (float64, int, int, error) {
					return 10.0, 10, 0, nil
				},
			},
			wantErr: true,
			errType: ErrInsufficientStock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewCartService(tt.mockCartRepo, tt.mockVariantFinder)
			_, err := service.AddItem(tt.userID, tt.variantID, tt.quantity)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if !errors.Is(err, tt.errType) {
					t.Errorf("error = %v; want %v", err, tt.errType)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCartService_UpdateQuantity(t *testing.T) {
	tests := []struct {
		name              string
		userID            uint
		variantID         uint
		quantity          int
		mockCartRepo      *MockCartRepository
		mockVariantFinder *MockVariantFinder
		wantErr           bool
		errType           error
	}{
		{
			name:      "update quantity with sufficient stock - success",
			userID:    1,
			variantID: 1,
			quantity:  5,
			mockCartRepo: &MockCartRepository{
				getCartFunc: func(userID uint) (*Cart, error) {
					return &Cart{
						UserID: userID,
						Items:  []CartItem{{VariantID: 1, Quantity: 2}},
					}, nil
				},
				saveCartFunc: func(cart *Cart) error {
					return nil
				},
			},
			mockVariantFinder: &MockVariantFinder{
				getPriceAndStockFunc: func(id uint) (float64, int, int, error) {
					return 10.0, 10, 0, nil
				},
			},
			wantErr: false,
		},
		{
			name:      "update quantity with insufficient stock - failure",
			userID:    1,
			variantID: 1,
			quantity:  20,
			mockCartRepo: &MockCartRepository{
				getCartFunc: func(userID uint) (*Cart, error) {
					return &Cart{
						UserID: userID,
						Items:  []CartItem{{VariantID: 1, Quantity: 2}},
					}, nil
				},
			},
			mockVariantFinder: &MockVariantFinder{
				getPriceAndStockFunc: func(id uint) (float64, int, int, error) {
					return 10.0, 10, 0, nil
				},
			},
			wantErr: true,
			errType: ErrInsufficientStock,
		},
		{
			name:      "update quantity for non-existent item - failure",
			userID:    1,
			variantID: 999,
			quantity:  5,
			mockCartRepo: &MockCartRepository{
				getCartFunc: func(userID uint) (*Cart, error) {
					return &Cart{
						UserID: userID,
						Items:  []CartItem{{VariantID: 1, Quantity: 2}},
					}, nil
				},
			},
			mockVariantFinder: &MockVariantFinder{
				getPriceAndStockFunc: func(id uint) (float64, int, int, error) {
					return 10.0, 10, 0, nil
				},
			},
			wantErr: true,
			errType: ErrVariantNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewCartService(tt.mockCartRepo, tt.mockVariantFinder)
			_, err := service.UpdateQuantity(tt.userID, tt.variantID, tt.quantity)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if !errors.Is(err, tt.errType) {
					t.Errorf("error = %v; want %v", err, tt.errType)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCartService_RemoveItem(t *testing.T) {
	tests := []struct {
		name         string
		userID       uint
		variantID    uint
		mockCartRepo *MockCartRepository
		wantErr      bool
		checkRemoved func(*testing.T, *Cart)
	}{
		{
			name:      "remove item - success",
			userID:    1,
			variantID: 1,
			mockCartRepo: &MockCartRepository{
				getCartFunc: func(userID uint) (*Cart, error) {
					return &Cart{
						UserID: userID,
						Items:  []CartItem{{VariantID: 1, Quantity: 2}, {VariantID: 2, Quantity: 1}},
					}, nil
				},
				saveCartFunc: func(cart *Cart) error {
					return nil
				},
			},
			wantErr: false,
			checkRemoved: func(t *testing.T, cart *Cart) {
				if len(cart.Items) != 1 {
					t.Errorf("expected 1 item, got %d", len(cart.Items))
				}
				if cart.Items[0].VariantID != 2 {
					t.Errorf("expected remaining item with variant_id 2, got %d", cart.Items[0].VariantID)
				}
			},
		},
		{
			name:      "remove item not in cart - returns original cart",
			userID:    1,
			variantID: 999,
			mockCartRepo: &MockCartRepository{
				getCartFunc: func(userID uint) (*Cart, error) {
					return &Cart{
						UserID: userID,
						Items:  []CartItem{{VariantID: 1, Quantity: 2}},
					}, nil
				},
			},
			wantErr: false,
			checkRemoved: func(t *testing.T, cart *Cart) {
				if len(cart.Items) != 1 {
					t.Errorf("expected 1 item, got %d", len(cart.Items))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variantFinder := &MockVariantFinder{}
			service := NewCartService(tt.mockCartRepo, variantFinder)
			cart, err := service.RemoveItem(tt.userID, tt.variantID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.checkRemoved != nil {
					tt.checkRemoved(t, cart)
				}
			}
		})
	}
}

func TestCartService_ClearCart(t *testing.T) {
	tests := []struct {
		name         string
		userID       uint
		mockCartRepo *MockCartRepository
		wantErr      bool
	}{
		{
			name:   "clear cart - success",
			userID: 1,
			mockCartRepo: &MockCartRepository{
				deleteCartFunc: func(userID uint) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name:   "clear cart - repository error",
			userID: 1,
			mockCartRepo: &MockCartRepository{
				deleteCartFunc: func(userID uint) error {
					return errors.New("database error")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variantFinder := &MockVariantFinder{}
			service := NewCartService(tt.mockCartRepo, variantFinder)
			err := service.ClearCart(tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCartService_GetCart(t *testing.T) {
	tests := []struct {
		name         string
		userID       uint
		mockCartRepo *MockCartRepository
		wantErr      bool
		checkCart    func(*testing.T, *Cart)
	}{
		{
			name:   "get cart - success",
			userID: 1,
			mockCartRepo: &MockCartRepository{
				getCartFunc: func(userID uint) (*Cart, error) {
					return &Cart{
						UserID:    userID,
						Items:     []CartItem{{VariantID: 1, Quantity: 2}},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil
				},
			},
			wantErr: false,
			checkCart: func(t *testing.T, cart *Cart) {
				if cart.UserID != 1 {
					t.Errorf("userID = %d; want 1", cart.UserID)
				}
				if len(cart.Items) != 1 {
					t.Errorf("items length = %d; want 1", len(cart.Items))
				}
			},
		},
		{
			name:   "get cart - empty cart",
			userID: 1,
			mockCartRepo: &MockCartRepository{
				getCartFunc: func(userID uint) (*Cart, error) {
					return &Cart{
						UserID:    userID,
						Items:     []CartItem{},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil
				},
			},
			wantErr: false,
			checkCart: func(t *testing.T, cart *Cart) {
				if len(cart.Items) != 0 {
					t.Errorf("items length = %d; want 0", len(cart.Items))
				}
			},
		},
		{
			name:   "get cart - repository error",
			userID: 1,
			mockCartRepo: &MockCartRepository{
				getCartFunc: func(userID uint) (*Cart, error) {
					return nil, errors.New("database error")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variantFinder := &MockVariantFinder{}
			service := NewCartService(tt.mockCartRepo, variantFinder)
			cart, err := service.GetCart(tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.checkCart != nil {
					tt.checkCart(t, cart)
				}
			}
		})
	}
}
