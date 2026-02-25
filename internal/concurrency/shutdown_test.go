package concurrency

import (
	"sync"
	"testing"
	"time"
)

func TestWorkerPool_GracefulShutdown_EmptyQueue(t *testing.T) {
	handler := func(task *Task) error {
		return nil
	}

	pool := NewWorkerPool(2, 10, handler)
	pool.Start()

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	if err := pool.Submit(&Task{Type: TaskTypeOrderProcessing}); err != ErrWorkerPoolNotStarted {
		t.Errorf("Expected ErrWorkerPoolNotStarted after shutdown, got %v", err)
	}
}

func TestWorkerPool_GracefulShutdown_WithPendingTasks(t *testing.T) {
	var counter int
	var mu sync.Mutex

	handler := func(task *Task) error {
		time.Sleep(20 * time.Millisecond)
		mu.Lock()
		counter++
		mu.Unlock()
		return nil
	}

	pool := NewWorkerPool(1, 10, handler)
	pool.Start()

	for i := 0; i < 3; i++ {
		task := &Task{Type: TaskTypeOrderProcessing}
		pool.Submit(task)
	}

	time.Sleep(10 * time.Millisecond)

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	mu.Lock()
	if counter < 1 {
		t.Errorf("Expected at least 1 task to be processed, got %d", counter)
	}
	mu.Unlock()
}

func TestWorkerPool_GracefulShutdown_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler := func(task *Task) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			}

			pool := NewWorkerPool(2, 5, handler)
			pool.Start()

			for j := 0; j < 3; j++ {
				pool.Submit(&Task{Type: TaskTypeOrderProcessing})
			}

			if err := pool.Shutdown(); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent shutdown error: %v", err)
		}
	}
}

func TestWorkerPool_GracefulShutdown_DoubleShutdown(t *testing.T) {
	handler := func(task *Task) error {
		return nil
	}

	pool := NewWorkerPool(2, 10, handler)
	pool.Start()

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("First shutdown failed: %v", err)
	}

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("Second shutdown failed: %v", err)
	}
}

func TestWorkerPool_ShutdownDuringTaskExecution(t *testing.T) {
	var executed int
	var mu sync.Mutex
	shutdownCalled := make(chan bool)

	handler := func(task *Task) error {
		mu.Lock()
		executed++
		mu.Unlock()
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	pool := NewWorkerPool(1, 10, handler)
	pool.Start()

	pool.Submit(&Task{Type: TaskTypeOrderProcessing})

	time.Sleep(10 * time.Millisecond)

	go func() {
		pool.Shutdown()
		shutdownCalled <- true
	}()

	select {
	case <-shutdownCalled:
	case <-time.After(500 * time.Millisecond):
		t.Error("Shutdown did not complete in time")
	}

	mu.Lock()
	if executed != 1 {
		t.Errorf("Expected 1 task to be executed, got %d", executed)
	}
	mu.Unlock()
}

func TestWorkerPool_Shutdown_ResubmitAfterShutdown(t *testing.T) {
	handler := func(task *Task) error {
		return nil
	}

	pool := NewWorkerPool(2, 10, handler)
	pool.Start()

	pool.Shutdown()

	task := &Task{Type: TaskTypeOrderProcessing}
	if err := pool.Submit(task); err != ErrWorkerPoolNotStarted {
		t.Errorf("Expected ErrWorkerPoolNotStarted after shutdown, got %v", err)
	}
}

func TestWorkerPool_GracefulShutdown_ZeroWorkers(t *testing.T) {
	handler := func(task *Task) error {
		return nil
	}

	pool := NewWorkerPool(0, 10, handler)

	if err := pool.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if err := pool.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestWorkerPool_StartMultipleTimes(t *testing.T) {
	handler := func(task *Task) error {
		return nil
	}

	pool := NewWorkerPool(2, 10, handler)

	if err := pool.Start(); err != nil {
		t.Fatalf("First start failed: %v", err)
	}

	if err := pool.Start(); err != nil {
		t.Fatalf("Second start failed: %v", err)
	}

	pool.Shutdown()
}
