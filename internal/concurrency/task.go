package concurrency

import (
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
	ID        string      `json:"id"`
	Type      TaskType    `json:"type"`
	Status    TaskStatus  `json:"status"`
	Payload   interface{} `json:"payload"`
	Result    interface{} `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}
