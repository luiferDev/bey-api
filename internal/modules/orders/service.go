package orders

import (
	"errors"
	"time"

	"bey/internal/concurrency"
	inventory "bey/internal/modules/inventory"
)

type InventoryReserver interface {
	Reserve(productID uint, quantity int) error
	FindByProductID(productID uint) (*inventory.Inventory, error)
}

type VariantStockManager interface {
	GetPriceAndStock(id uint) (float64, int, int, error) // price, stock, reserved, error
	ReserveStock(id uint, quantity int) error
	ReleaseStock(id uint, quantity int) error
	ConfirmSale(id uint, quantity int) error
}

type OrderService struct {
	repo        *OrderRepository
	taskQueue   concurrency.TaskQueue
	productRepo interface {
		GetPriceByID(id uint) (float64, error)
	}
	inventoryRepo InventoryReserver
	variantRepo   VariantStockManager
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
	GetPriceByID(id uint) (float64, error)
}) *OrderService {
	return &OrderService{
		repo:        repo,
		productRepo: productRepo,
	}
}

func NewOrderServiceWithAll(repo *OrderRepository, taskQueue concurrency.TaskQueue, productRepo interface {
	GetPriceByID(id uint) (float64, error)
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
		GetPriceByID(id uint) (float64, error)
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
		GetPriceByID(id uint) (float64, error)
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
		GetPriceByID(id uint) (float64, error)
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

func (s *OrderService) SubmitAsyncOrder(req CreateOrderRequest) (string, error) {
	if s.taskQueue == nil {
		return "", errors.New("task queue not configured")
	}

	task := &concurrency.Task{
		Type:    concurrency.TaskTypeOrderProcessing,
		Status:  concurrency.TaskStatusPending,
		Payload: req,
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

	req, ok := task.Payload.(CreateOrderRequest)
	if !ok {
		task.Status = concurrency.TaskStatusFailed
		task.Error = "invalid payload type"
		task.UpdatedAt = time.Now()
		return
	}

	var totalPrice float64
	items := make([]OrderItem, len(req.Items))

	for i, item := range req.Items {
		var unitPrice float64

		// If variant_id is specified, use variant price and stock
		if item.VariantID != nil && s.variantRepo != nil {
			price, stock, reserved, err := s.variantRepo.GetPriceAndStock(*item.VariantID)
			if err != nil {
				task.Status = concurrency.TaskStatusFailed
				task.Error = "failed to get variant info"
				task.UpdatedAt = time.Now()
				return
			}

			// Check available stock (stock - reserved)
			available := stock - reserved
			if available < item.Quantity {
				task.Status = concurrency.TaskStatusFailed
				task.Error = "insufficient stock for variant"
				task.UpdatedAt = time.Now()
				return
			}

			// Reserve stock in variant only (inventory is sum of variants)
			if err := s.variantRepo.ReserveStock(*item.VariantID, item.Quantity); err != nil {
				task.Status = concurrency.TaskStatusFailed
				task.Error = "failed to reserve variant stock"
				task.UpdatedAt = time.Now()
				return
			}

			unitPrice = price
		} else {
			// Use product price if no variant
			if s.productRepo != nil {
				price, err := s.productRepo.GetPriceByID(item.ProductID)
				if err != nil {
					task.Status = concurrency.TaskStatusFailed
					task.Error = "failed to get product price"
					task.UpdatedAt = time.Now()
					return
				}
				unitPrice = price
			}

			// Reserve inventory if available (for products without variants)
			if s.inventoryRepo != nil {
				inv, err := s.inventoryRepo.FindByProductID(item.ProductID)
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

				if err := s.inventoryRepo.Reserve(item.ProductID, item.Quantity); err != nil {
					task.Status = concurrency.TaskStatusFailed
					task.Error = "failed to reserve inventory"
					task.UpdatedAt = time.Now()
					return
				}
			}
		}

		items[i] = OrderItem{
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			UnitPrice: unitPrice,
		}
		totalPrice += float64(item.Quantity) * unitPrice
	}

	order := &Order{
		UserID:          req.UserID,
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
