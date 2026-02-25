package concurrency

import (
	"sync"
	"time"
)

type TaskHandler func(task *Task) error

type WorkerPool interface {
	Submit(task *Task) error
	Start() error
	Shutdown() error
}

type workerPool struct {
	workerCount int
	queueDepth  int
	taskQueue   chan *Task
	wg          sync.WaitGroup
	stopChan    chan struct{}
	handler     TaskHandler
	started     bool
}

func NewWorkerPool(workerCount int, queueDepth int, handler TaskHandler) WorkerPool {
	return &workerPool{
		workerCount: workerCount,
		queueDepth:  queueDepth,
		taskQueue:   make(chan *Task, queueDepth),
		stopChan:    make(chan struct{}),
		handler:     handler,
		started:     false,
	}
}

func (p *workerPool) Start() error {
	if p.started {
		return nil
	}

	p.started = true

	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	return nil
}

func (p *workerPool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case task, ok := <-p.taskQueue:
			if !ok {
				return
			}
			if task.Status == TaskStatusCancelled {
				continue
			}
			task.Status = TaskStatusRunning
			task.UpdatedAt = time.Now()

			if err := p.handler(task); err != nil {
				task.Status = TaskStatusFailed
				task.Error = err.Error()
			} else {
				task.Status = TaskStatusCompleted
			}
			task.UpdatedAt = time.Now()
		case <-p.stopChan:
			return
		}
	}
}

func (p *workerPool) Submit(task *Task) error {
	if !p.started {
		return ErrWorkerPoolNotStarted
	}

	select {
	case p.taskQueue <- task:
		return nil
	default:
		return ErrQueueFull
	}
}

func (p *workerPool) Shutdown() error {
	if !p.started {
		return nil
	}

	close(p.stopChan)
	p.wg.Wait()
	close(p.taskQueue)

	p.started = false
	return nil
}

var (
	ErrWorkerPoolNotStarted = &WorkerPoolError{"worker pool not started"}
	ErrQueueFull            = &WorkerPoolError{"task queue is full"}
)

type WorkerPoolError struct {
	message string
}

func (e *WorkerPoolError) Error() string {
	return e.message
}
