package cart

import (
	"errors"
	"sync"
	"time"

	"bey/internal/modules/products"

	"github.com/gofrs/uuid/v5"
)

var (
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrUnauthorized      = errors.New("unauthorized access to cart")
	ErrVariantNotFound   = errors.New("variant not found")
	ErrCartEmpty         = errors.New("cart is empty")
)

type VariantFinder interface {
	FindByID(id uuid.UUID) (*products.ProductVariant, error)
	GetPriceAndStock(id uuid.UUID) (float64, int, int, error)
}

type CartService struct {
	cartRepo    CartRepository
	variantRepo VariantFinder
	mu          sync.RWMutex
	userLocks   map[string]*sync.Mutex
}

func NewCartService(cartRepo CartRepository, variantRepo VariantFinder) *CartService {
	return &CartService{
		cartRepo:    cartRepo,
		variantRepo: variantRepo,
		userLocks:   make(map[string]*sync.Mutex),
	}
}

func (s *CartService) getUserLock(userID uuid.UUID) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := userID.String()
	lock, exists := s.userLocks[key]
	if !exists {
		lock = &sync.Mutex{}
		s.userLocks[key] = lock
	}
	return lock
}

func (s *CartService) GetCart(userID uuid.UUID) (*Cart, error) {
	return s.cartRepo.GetCart(userID)
}

func (s *CartService) AddItem(userID uuid.UUID, variantID uuid.UUID, quantity int) (*Cart, error) {
	lock := s.getUserLock(userID)
	lock.Lock()
	defer lock.Unlock()

	variant, err := s.variantRepo.FindByID(variantID)
	if err != nil {
		return nil, err
	}
	if variant == nil {
		return nil, ErrVariantNotFound
	}

	_, stock, _, err := s.variantRepo.GetPriceAndStock(variantID)
	if err != nil {
		return nil, err
	}
	if stock < quantity {
		return nil, ErrInsufficientStock
	}

	cart, err := s.cartRepo.GetCart(userID)
	if err != nil {
		return nil, err
	}

	variantIDStr := variantID.String()
	existingIdx := -1
	for i, item := range cart.Items {
		if item.VariantID == variantIDStr {
			existingIdx = i
			break
		}
	}

	if existingIdx >= 0 {
		newQuantity := cart.Items[existingIdx].Quantity + quantity
		if stock < newQuantity {
			return nil, ErrInsufficientStock
		}
		cart.Items[existingIdx].Quantity = newQuantity
	} else {
		cart.Items = append(cart.Items, CartItem{
			VariantID: variantIDStr,
			Quantity:  quantity,
		})
	}

	cart.UpdatedAt = time.Now()

	if err := s.cartRepo.SaveCart(cart); err != nil {
		return nil, err
	}

	return cart, nil
}

func (s *CartService) RemoveItem(userID uuid.UUID, variantID uuid.UUID) (*Cart, error) {
	lock := s.getUserLock(userID)
	lock.Lock()
	defer lock.Unlock()

	cart, err := s.cartRepo.GetCart(userID)
	if err != nil {
		return nil, err
	}

	variantIDStr := variantID.String()
	found := false
	newItems := make([]CartItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		if item.VariantID == variantIDStr {
			found = true
			continue
		}
		newItems = append(newItems, item)
	}

	if !found {
		return cart, nil
	}

	cart.Items = newItems
	cart.UpdatedAt = time.Now()

	if err := s.cartRepo.SaveCart(cart); err != nil {
		return nil, err
	}

	return cart, nil
}

func (s *CartService) UpdateQuantity(userID uuid.UUID, variantID uuid.UUID, quantity int) (*Cart, error) {
	lock := s.getUserLock(userID)
	lock.Lock()
	defer lock.Unlock()

	if quantity == 0 {
		return s.RemoveItem(userID, variantID)
	}

	_, stock, _, err := s.variantRepo.GetPriceAndStock(variantID)
	if err != nil {
		return nil, err
	}
	if stock < quantity {
		return nil, ErrInsufficientStock
	}

	cart, err := s.cartRepo.GetCart(userID)
	if err != nil {
		return nil, err
	}

	variantIDStr := variantID.String()
	found := false
	for i, item := range cart.Items {
		if item.VariantID == variantIDStr {
			cart.Items[i].Quantity = quantity
			found = true
			break
		}
	}

	if !found {
		return nil, ErrVariantNotFound
	}

	cart.UpdatedAt = time.Now()

	if err := s.cartRepo.SaveCart(cart); err != nil {
		return nil, err
	}

	return cart, nil
}

func (s *CartService) ClearCart(userID uuid.UUID) error {
	lock := s.getUserLock(userID)
	lock.Lock()
	defer lock.Unlock()

	return s.cartRepo.DeleteCart(userID)
}

// CheckoutResult contains the order data ready to be persisted and the cart to clear
type CheckoutResult struct {
	UserID          uuid.UUID
	ShippingAddress string
	Notes           string
	Items           []CheckoutItem
	TotalPrice      float64
}

type CheckoutItem struct {
	ProductID uuid.UUID
	VariantID *uuid.UUID
	Quantity  int
	UnitPrice float64
}

func (s *CartService) PrepareCheckout(userID uuid.UUID, shippingAddress string, notes string) (*CheckoutResult, error) {
	lock := s.getUserLock(userID)
	lock.Lock()
	defer lock.Unlock()

	cart, err := s.cartRepo.GetCart(userID)
	if err != nil {
		return nil, err
	}

	if len(cart.Items) == 0 {
		return nil, ErrCartEmpty
	}

	var items []CheckoutItem
	var totalPrice float64

	for _, item := range cart.Items {
		variantID, err := uuid.FromString(item.VariantID)
		if err != nil {
			return nil, err
		}

		price, stock, _, err := s.variantRepo.GetPriceAndStock(variantID)
		if err != nil {
			return nil, err
		}
		if stock < item.Quantity {
			return nil, ErrInsufficientStock
		}

		variant, err := s.variantRepo.FindByID(variantID)
		if err != nil {
			return nil, err
		}
		if variant == nil {
			return nil, ErrVariantNotFound
		}

		items = append(items, CheckoutItem{
			ProductID: variant.ProductID,
			VariantID: &variantID,
			Quantity:  item.Quantity,
			UnitPrice: price,
		})
		totalPrice += price * float64(item.Quantity)
	}

	return &CheckoutResult{
		UserID:          userID,
		ShippingAddress: shippingAddress,
		Notes:           notes,
		Items:           items,
		TotalPrice:      totalPrice,
	}, nil
}

func (s *CartService) ClearCartAfterCheckout(userID uuid.UUID) error {
	return s.cartRepo.DeleteCart(userID)
}

func (s *CartService) ValidateCartOwnership(cartUserID, tokenUserID uuid.UUID) error {
	if cartUserID != tokenUserID {
		return ErrUnauthorized
	}
	return nil
}

// GetVariantPrice returns the price for a variant (helper for checkout response)
func (s *CartService) GetVariantPrice(variantID uuid.UUID) (float64, error) {
	price, _, _, err := s.variantRepo.GetPriceAndStock(variantID)
	return price, err
}
