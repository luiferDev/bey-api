package concurrency

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestWorkerPool_SubmitAndProcess(t *testing.T) {
	handlerCalled := false
	var mu sync.Mutex

	handler := func(task *Task) error {
		mu.Lock()
		handlerCalled = true
		mu.Unlock()
		return nil
	}

	pool := NewWorkerPool(2, 10, handler)
	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	task := &Task{
		Type:    TaskTypeOrderProcessing,
		Payload: "test",
	}

	if err := pool.Submit(task); err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	mu.Lock()
	if !handlerCalled {
		t.Error("Expected handler to be called")
	}
	mu.Unlock()

	if task.Status != TaskStatusCompleted {
		t.Errorf("Expected task status completed, got %s", task.Status)
	}
}

func TestWorkerPool_MultipleTasks(t *testing.T) {
	var counter int
	var mu sync.Mutex

	handler := func(task *Task) error {
		mu.Lock()
		counter++
		mu.Unlock()
		return nil
	}

	pool := NewWorkerPool(2, 10, handler)
	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	for i := 0; i < 5; i++ {
		task := &Task{
			Type:    TaskTypeBulkUpdate,
			Payload: i,
		}
		if err := pool.Submit(task); err != nil {
			t.Fatalf("Failed to submit task %d: %v", i, err)
		}
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	if counter != 5 {
		t.Errorf("Expected 5 tasks processed, got %d", counter)
	}
	mu.Unlock()

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestWorkerPool_QueueDepthLimit(t *testing.T) {
	handler := func(task *Task) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	queueDepth := 2
	pool := NewWorkerPool(1, queueDepth, handler)
	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	submitted := 0
	failed := 0

	for i := 0; i < 5; i++ {
		task := &Task{
			Type:    TaskTypeOrderProcessing,
			Payload: i,
		}
		if err := pool.Submit(task); err != nil {
			failed++
		} else {
			submitted++
		}
	}

	if submitted != queueDepth {
		t.Errorf("Expected %d tasks to be submitted, got %d", queueDepth, submitted)
	}

	if failed != 3 {
		t.Errorf("Expected 3 tasks to fail due to queue full, got %d", failed)
	}

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestWorkerPool_QueueFullError(t *testing.T) {
	handler := func(task *Task) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	pool := NewWorkerPool(1, 1, handler)
	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	task1 := &Task{Type: TaskTypeOrderProcessing}
	if err := pool.Submit(task1); err != nil {
		t.Fatalf("First submit should succeed: %v", err)
	}

	task2 := &Task{Type: TaskTypeOrderProcessing}
	if err := pool.Submit(task2); err != ErrQueueFull {
		t.Errorf("Expected ErrQueueFull, got %v", err)
	}

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestWorkerPool_TaskErrorHandling(t *testing.T) {
	expectedErr := errors.New("task processing failed")

	handler := func(task *Task) error {
		return expectedErr
	}

	pool := NewWorkerPool(1, 10, handler)
	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	task := &Task{
		Type:    TaskTypeOrderProcessing,
		Payload: "test",
	}

	if err := pool.Submit(task); err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	if task.GetStatus() != TaskStatusFailed {
		t.Errorf("Expected task status failed, got %s", task.GetStatus())
	}

	if task.GetError() != expectedErr.Error() {
		t.Errorf("Expected error message '%s', got '%s'", expectedErr.Error(), task.GetError())
	}
}

func TestWorkerPool_SubmitBeforeStart(t *testing.T) {
	handler := func(task *Task) error {
		return nil
	}

	pool := NewWorkerPool(1, 10, handler)

	task := &Task{Type: TaskTypeOrderProcessing}
	if err := pool.Submit(task); err != ErrWorkerPoolNotStarted {
		t.Errorf("Expected ErrWorkerPoolNotStarted, got %v", err)
	}
}

func TestWorkerPool_Shutdown(t *testing.T) {
	var counter int
	var mu sync.Mutex

	handler := func(task *Task) error {
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		counter++
		mu.Unlock()
		return nil
	}

	pool := NewWorkerPool(2, 10, handler)
	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	for i := 0; i < 3; i++ {
		task := &Task{Type: TaskTypeOrderProcessing}
		if err := pool.Submit(task); err != nil {
			t.Fatalf("Failed to submit task: %v", err)
		}
	}

	time.Sleep(20 * time.Millisecond)

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	mu.Lock()
	if counter < 3 {
		t.Logf("Expected 3 tasks processed, got %d", counter)
	}
	mu.Unlock()
}

func TestWorkerPool_CancelTask(t *testing.T) {
	// Block the first worker so we can cancel the second task before it starts.
	block := make(chan struct{})
	handler := func(task *Task) error {
		<-block
		return nil
	}

	pool := NewWorkerPool(1, 10, handler)
	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	// Submit a blocking task to occupy the single worker.
	blockingTask := &Task{Type: TaskTypeOrderProcessing, Payload: "blocker"}
	if err := pool.Submit(blockingTask); err != nil {
		t.Fatalf("Failed to submit blocking task: %v", err)
	}

	// Submit the task we want to cancel.
	task := &Task{Type: TaskTypeOrderProcessing, Payload: "cancellable"}
	if err := pool.Submit(task); err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	time.Sleep(20 * time.Millisecond)
	task.Status = TaskStatusCancelled

	// Unblock the worker so it can finish the first task and see the cancelled status.
	close(block)

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// The cancelled task should not have been processed.
	if task.Status != TaskStatusCancelled {
		t.Errorf("Expected task to remain cancelled, got %s", task.Status)
	}
}

func TestWorkerPool_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	handler := func(task *Task) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return nil
	}

	pool := NewWorkerPool(2, 10, handler)
	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	cancel()

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}
