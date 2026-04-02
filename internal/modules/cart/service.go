package cart

import (
	"errors"
	"sync"
	"time"

	"bey/internal/modules/orders"
	"bey/internal/modules/products"
)

var (
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrUnauthorized      = errors.New("unauthorized access to cart")
	ErrVariantNotFound   = errors.New("variant not found")
	ErrCartEmpty         = errors.New("cart is empty")
)

type VariantFinder interface {
	FindByID(id uint) (*products.ProductVariant, error)
	GetPriceAndStock(id uint) (float64, int, int, error)
}

type CartService struct {
	cartRepo    CartRepository
	variantRepo VariantFinder
	mu          sync.RWMutex
	userLocks   map[uint]*sync.Mutex
}

func NewCartService(cartRepo CartRepository, variantRepo VariantFinder) *CartService {
	return &CartService{
		cartRepo:    cartRepo,
		variantRepo: variantRepo,
		userLocks:   make(map[uint]*sync.Mutex),
	}
}

func (s *CartService) getUserLock(userID uint) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()

	lock, exists := s.userLocks[userID]
	if !exists {
		lock = &sync.Mutex{}
		s.userLocks[userID] = lock
	}
	return lock
}

func (s *CartService) GetCart(userID uint) (*Cart, error) {
	return s.cartRepo.GetCart(userID)
}

func (s *CartService) AddItem(userID uint, variantID uint, quantity int) (*Cart, error) {
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

	existingIdx := -1
	for i, item := range cart.Items {
		if item.VariantID == variantID {
			existingIdx = i
			break
		}
	}

	newQuantity := quantity
	if existingIdx >= 0 {
		newQuantity = cart.Items[existingIdx].Quantity + quantity
		if stock < newQuantity {
			return nil, ErrInsufficientStock
		}
		cart.Items[existingIdx].Quantity = newQuantity
	} else {
		cart.Items = append(cart.Items, CartItem{
			VariantID: variantID,
			Quantity:  quantity,
		})
	}

	cart.UpdatedAt = time.Now()

	if err := s.cartRepo.SaveCart(cart); err != nil {
		return nil, err
	}

	return cart, nil
}

func (s *CartService) RemoveItem(userID uint, variantID uint) (*Cart, error) {
	lock := s.getUserLock(userID)
	lock.Lock()
	defer lock.Unlock()

	cart, err := s.cartRepo.GetCart(userID)
	if err != nil {
		return nil, err
	}

	found := false
	newItems := make([]CartItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		if item.VariantID == variantID {
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

func (s *CartService) UpdateQuantity(userID uint, variantID uint, quantity int) (*Cart, error) {
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

	found := false
	for i, item := range cart.Items {
		if item.VariantID == variantID {
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

func (s *CartService) ClearCart(userID uint) error {
	lock := s.getUserLock(userID)
	lock.Lock()
	defer lock.Unlock()

	return s.cartRepo.DeleteCart(userID)
}

func (s *CartService) CartToOrder(userID uint, shippingAddress string, notes string) (*orders.CreateOrderRequest, error) {
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

	orderItems := make([]orders.CreateOrderItemRequest, 0, len(cart.Items))
	var totalPrice float64

	for _, item := range cart.Items {
		price, stock, _, err := s.variantRepo.GetPriceAndStock(item.VariantID)
		if err != nil {
			return nil, err
		}
		if stock < item.Quantity {
			return nil, ErrInsufficientStock
		}

		variant, err := s.variantRepo.FindByID(item.VariantID)
		if err != nil {
			return nil, err
		}
		if variant == nil {
			return nil, ErrVariantNotFound
		}

		orderItems = append(orderItems, orders.CreateOrderItemRequest{
			ProductID: variant.ProductID,
			VariantID: &item.VariantID,
			Quantity:  item.Quantity,
		})
		totalPrice += price * float64(item.Quantity)
	}

	req := &orders.CreateOrderRequest{
		ShippingAddress: shippingAddress,
		Notes:           notes,
		Items:           orderItems,
	}

	if err := s.cartRepo.DeleteCart(userID); err != nil {
		return nil, err
	}

	_ = totalPrice

	return req, nil
}

func (s *CartService) ValidateCartOwnership(cartUserID, tokenUserID uint) error {
	if cartUserID != tokenUserID {
		return ErrUnauthorized
	}
	return nil
}

// GetVariantPrice returns the price for a variant (helper for checkout response)
func (s *CartService) GetVariantPrice(variantID *uint) (float64, error) {
	if variantID == nil {
		return 0, nil
	}
	price, _, _, err := s.variantRepo.GetPriceAndStock(*variantID)
	return price, err
}
