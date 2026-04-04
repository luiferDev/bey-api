package concurrency

import (
	"sync"
	"time"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

type TaskType string

const (
	TaskTypeOrderProcessing TaskType = "order_processing"
	TaskTypeBulkUpdate      TaskType = "bulk_update"
	TaskTypeBulkCreate      TaskType = "bulk_create"
	TaskTypeBulkDelete      TaskType = "bulk_delete"
)

type Task struct {
	mu        sync.RWMutex
	ID        string      `json:"id"`
	Type      TaskType    `json:"type"`
	Status    TaskStatus  `json:"status"`
	Payload   interface{} `json:"payload"`
	Result    interface{} `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// SetStatus safely updates the task status.
func (t *Task) SetStatus(s TaskStatus) {
	t.mu.Lock()
	t.Status = s
	t.mu.Unlock()
}

// GetStatus safely reads the task status.
func (t *Task) GetStatus() TaskStatus {
	t.mu.RLock()
	s := t.Status
	t.mu.RUnlock()
	return s
}

// SetError safely updates the task error.
func (t *Task) SetError(err string) {
	t.mu.Lock()
	t.Error = err
	t.mu.Unlock()
}

// GetError safely reads the task error.
func (t *Task) GetError() string {
	t.mu.RLock()
	e := t.Error
	t.mu.RUnlock()
	return e
}

// SetResult safely updates the task result.
func (t *Task) SetResult(result interface{}) {
	t.mu.Lock()
	t.Result = result
	t.mu.Unlock()
}

// GetResult safely reads the task result.
func (t *Task) GetResult() interface{} {
	t.mu.RLock()
	r := t.Result
	t.mu.RUnlock()
	return r
}

// SetUpdatedAt safely updates the task timestamp.
func (t *Task) SetUpdatedAt(ts time.Time) {
	t.mu.Lock()
	t.UpdatedAt = ts
	t.mu.Unlock()
}

// Copy returns a thread-safe copy of the task.
func (t *Task) Copy() Task {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return Task{
		ID:        t.ID,
		Type:      t.Type,
		Status:    t.Status,
		Payload:   t.Payload,
		Result:    t.Result,
		Error:     t.Error,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}
