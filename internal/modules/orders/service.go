package orders

import (
	"errors"
	"fmt"
	"log"
	"time"

	"bey/internal/concurrency"
	inventory "bey/internal/modules/inventory"

	"github.com/gofrs/uuid/v5"
)

type InventoryReserver interface {
	Reserve(productID uuid.UUID, quantity int) error
	FindByProductID(productID uuid.UUID) (*inventory.Inventory, error)
}

type VariantStockManager interface {
	GetPriceAndStock(id uuid.UUID) (float64, int, int, error)
	ReserveStock(id uuid.UUID, quantity int) error
	ReleaseStock(id uuid.UUID, quantity int) error
	ConfirmSale(id uuid.UUID, quantity int) error
}

type OrderService struct {
	repo        *OrderRepository
	taskQueue   concurrency.TaskQueue
	productRepo interface {
		GetPriceByID(id uuid.UUID) (float64, error)
	}
	variantRepo   VariantStockManager
	inventoryRepo InventoryReserver
}

// CreateOrderPayload wraps the request with authenticated user ID
type CreateOrderPayload struct {
	Request CreateOrderRequest
	UserID  uuid.UUID
}

func NewOrderService(repo *OrderRepository) *OrderService {
	return &OrderService{
		repo: repo,
	}
}

func NewOrderServiceWithTaskQueue(repo *OrderRepository, taskQueue concurrency.TaskQueue) *OrderService {
	return &OrderService{
		repo:      repo,
		taskQueue: taskQueue,
	}
}

func NewOrderServiceWithProductRepo(repo *OrderRepository, productRepo interface {
	GetPriceByID(id uuid.UUID) (float64, error)
}) *OrderService {
	return &OrderService{
		repo:        repo,
		productRepo: productRepo,
	}
}

func NewOrderServiceWithAll(repo *OrderRepository, taskQueue concurrency.TaskQueue, productRepo interface {
	GetPriceByID(id uuid.UUID) (float64, error)
}) *OrderService {
	return &OrderService{
		repo:        repo,
		taskQueue:   taskQueue,
		productRepo: productRepo,
	}
}

func NewOrderServiceWithInventory(repo *OrderRepository, inventoryRepo InventoryReserver) *OrderService {
	return &OrderService{
		repo:          repo,
		inventoryRepo: inventoryRepo,
	}
}

func NewOrderServiceWithProductAndInventory(
	repo *OrderRepository,
	productRepo interface {
		GetPriceByID(id uuid.UUID) (float64, error)
	},
	inventoryRepo InventoryReserver,
) *OrderService {
	return &OrderService{
		repo:          repo,
		productRepo:   productRepo,
		inventoryRepo: inventoryRepo,
	}
}

func NewOrderServiceWithAllDeps(
	repo *OrderRepository,
	taskQueue concurrency.TaskQueue,
	productRepo interface {
		GetPriceByID(id uuid.UUID) (float64, error)
	},
	inventoryRepo InventoryReserver,
) *OrderService {
	return &OrderService{
		repo:          repo,
		taskQueue:     taskQueue,
		productRepo:   productRepo,
		inventoryRepo: inventoryRepo,
	}
}

// NewOrderServiceWithAllDepsAndVariant creates OrderService with all dependencies including variant
func NewOrderServiceWithAllDepsAndVariant(
	repo *OrderRepository,
	taskQueue concurrency.TaskQueue,
	productRepo interface {
		GetPriceByID(id uuid.UUID) (float64, error)
	},
	inventoryRepo InventoryReserver,
	variantRepo VariantStockManager,
) *OrderService {
	return &OrderService{
		repo:          repo,
		taskQueue:     taskQueue,
		productRepo:   productRepo,
		inventoryRepo: inventoryRepo,
		variantRepo:   variantRepo,
	}
}

func (s *OrderService) SubmitAsyncOrder(req CreateOrderRequest, userID uuid.UUID) (string, error) {
	if s.taskQueue == nil {
		return "", errors.New("task queue not configured")
	}

	task := &concurrency.Task{
		Type:    concurrency.TaskTypeOrderProcessing,
		Status:  concurrency.TaskStatusPending,
		Payload: CreateOrderPayload{Request: req, UserID: userID},
	}

	taskID, err := s.taskQueue.Submit(task)
	if err != nil {
		return "", err
	}

	go s.processOrderTask(task)

	return taskID, nil
}

func (s *OrderService) processOrderTask(task *concurrency.Task) {
	task.Status = concurrency.TaskStatusRunning
	task.UpdatedAt = time.Now()

	payload, ok := task.Payload.(CreateOrderPayload)
	if !ok {
		task.Status = concurrency.TaskStatusFailed
		task.Error = "invalid payload type"
		task.UpdatedAt = time.Now()
		return
	}
	req := payload.Request
	userID := payload.UserID

	var totalPrice float64
	items := make([]OrderItem, len(req.Items))

	for i, item := range req.Items {
		var unitPrice float64

		if item.VariantID != nil && s.variantRepo != nil {
			variantID, err := uuid.FromString(*item.VariantID)
			if err != nil {
				task.Status = concurrency.TaskStatusFailed
				task.Error = "invalid variant_id format"
				task.UpdatedAt = time.Now()
				return
			}

			price, stock, reserved, err := s.variantRepo.GetPriceAndStock(variantID)
			if err != nil {
				task.Status = concurrency.TaskStatusFailed
				task.Error = "failed to get variant info"
				task.UpdatedAt = time.Now()
				return
			}

			available := stock - reserved
			if available < item.Quantity {
				task.Status = concurrency.TaskStatusFailed
				task.Error = "insufficient stock for variant"
				task.UpdatedAt = time.Now()
				return
			}

			if err := s.variantRepo.ReserveStock(variantID, item.Quantity); err != nil {
				task.Status = concurrency.TaskStatusFailed
				task.Error = "failed to reserve variant stock"
				task.UpdatedAt = time.Now()
				return
			}

			unitPrice = price
		} else {
			productID, err := uuid.FromString(item.ProductID)
			if err != nil {
				task.Status = concurrency.TaskStatusFailed
				task.Error = "invalid product_id format"
				task.UpdatedAt = time.Now()
				return
			}

			if s.productRepo != nil {
				price, err := s.productRepo.GetPriceByID(productID)
				if err != nil {
					task.Status = concurrency.TaskStatusFailed
					task.Error = "failed to get product price"
					task.UpdatedAt = time.Now()
					return
				}
				unitPrice = price
			}

			if s.inventoryRepo != nil {
				inv, err := s.inventoryRepo.FindByProductID(productID)
				if err != nil {
					task.Status = concurrency.TaskStatusFailed
					task.Error = "failed to check inventory"
					task.UpdatedAt = time.Now()
					return
				}
				if inv == nil || inv.Quantity < item.Quantity {
					task.Status = concurrency.TaskStatusFailed
					task.Error = "insufficient inventory"
					task.UpdatedAt = time.Now()
					return
				}

				if err := s.inventoryRepo.Reserve(productID, item.Quantity); err != nil {
					task.Status = concurrency.TaskStatusFailed
					task.Error = "failed to reserve inventory"
					task.UpdatedAt = time.Now()
					return
				}
			}
		}

		orderItem := OrderItem{
			Quantity:  item.Quantity,
			UnitPrice: unitPrice,
		}
		if productID, err := uuid.FromString(item.ProductID); err == nil {
			orderItem.ProductID = productID
		}
		if item.VariantID != nil && *item.VariantID != "" {
			if variantID, err := uuid.FromString(*item.VariantID); err == nil {
				orderItem.VariantID = &variantID
			}
		}
		items[i] = orderItem
		totalPrice += float64(item.Quantity) * unitPrice
	}

	order := &Order{
		UserID:          userID,
		Status:          "pending",
		TotalPrice:      totalPrice,
		ShippingAddress: req.ShippingAddress,
		Notes:           req.Notes,
		Items:           items,
	}

	if err := s.repo.Create(order); err != nil {
		task.Status = concurrency.TaskStatusFailed
		task.Error = err.Error()
		task.UpdatedAt = time.Now()
		return
	}

	task.Result = map[string]interface{}{
		"order_id":    order.ID,
		"total_price": order.TotalPrice,
		"status":      order.Status,
	}
	task.Status = concurrency.TaskStatusCompleted
	task.UpdatedAt = time.Now()
}

func (s *OrderService) GetTaskStatus(taskID string) (*concurrency.Task, error) {
	if s.taskQueue == nil {
		return nil, errors.New("task queue not configured")
	}

	return s.taskQueue.GetStatus(taskID)
}

func (s *OrderService) GetOrderByID(id uuid.UUID) (*Order, error) {
	return s.repo.FindByID(id)
}

func (s *OrderService) UpdatePaymentStatus(orderID uuid.UUID, paymentStatus, transactionID string) error {
	order, err := s.repo.FindByID(orderID)
	if err != nil {
		return fmt.Errorf("find order: %w", err)
	}
	if order == nil {
		return errors.New("order not found")
	}

	order.PaymentStatus = paymentStatus
	if transactionID != "" {
		order.PaymentTransactionID = transactionID
	}

	if paymentStatus == "paid" {
		order.Status = "confirmed"
		if s.variantRepo != nil {
			for _, item := range order.Items {
				if item.VariantID != nil {
					if err := s.variantRepo.ConfirmSale(*item.VariantID, item.Quantity); err != nil {
						log.Printf("Failed to confirm sale for variant %d: %v", *item.VariantID, err)
					}
				}
			}
		}
	}

	if err := s.repo.Update(order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	log.Printf("Updated order %d payment status to %s", orderID, paymentStatus)
	return nil
}
