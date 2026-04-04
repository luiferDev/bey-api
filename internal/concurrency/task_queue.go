package concurrency

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TaskQueue interface {
	Submit(task *Task) (string, error)
	GetStatus(taskID string) (*Task, error)
	Cancel(taskID string) error
}

type inMemoryTaskQueue struct {
	tasks map[string]*Task
	mu    sync.RWMutex
}

func NewInMemoryTaskQueue() TaskQueue {
	return &inMemoryTaskQueue{
		tasks: make(map[string]*Task),
	}
}

func (q *inMemoryTaskQueue) Submit(task *Task) (string, error) {
	if task == nil {
		return "", errors.New("task cannot be nil")
	}

	task.ID = uuid.New().String()
	task.SetStatus(TaskStatusPending)
	task.SetUpdatedAt(time.Now())

	q.mu.Lock()
	defer q.mu.Unlock()

	q.tasks[task.ID] = task

	return task.ID, nil
}

func (q *inMemoryTaskQueue) GetStatus(taskID string) (*Task, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	task, exists := q.tasks[taskID]
	if !exists {
		return nil, nil
	}

	// Return a thread-safe copy to prevent data races with worker goroutines.
	taskCopy := task.Copy()
	return &taskCopy, nil
}

func (q *inMemoryTaskQueue) Cancel(taskID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	task, exists := q.tasks[taskID]
	if !exists {
		return errors.New("task not found")
	}

	if task.GetStatus() == TaskStatusCompleted || task.GetStatus() == TaskStatusFailed {
		return errors.New("cannot cancel completed or failed task")
	}

	task.SetStatus(TaskStatusCancelled)
	task.SetUpdatedAt(time.Now())

	return nil
}
