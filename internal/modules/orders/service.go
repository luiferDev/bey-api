package orders

import (
	"errors"
	"time"

	"bey/internal/concurrency"
)

type OrderService struct {
	repo      *OrderRepository
	taskQueue concurrency.TaskQueue
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
		unitPrice := 0.0
		items[i] = OrderItem{
			ProductID: item.ProductID,
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
