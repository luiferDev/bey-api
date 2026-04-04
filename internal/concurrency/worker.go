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
	mu          sync.Mutex
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
	}
}

func (p *workerPool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

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
			if task.GetStatus() == TaskStatusCancelled {
				continue
			}
			task.SetStatus(TaskStatusRunning)
			task.SetUpdatedAt(time.Now())

			if err := p.handler(task); err != nil {
				task.SetStatus(TaskStatusFailed)
				task.SetError(err.Error())
			} else {
				task.SetStatus(TaskStatusCompleted)
			}
			task.SetUpdatedAt(time.Now())
		case <-p.stopChan:
			return
		}
	}
}

func (p *workerPool) Submit(task *Task) error {
	p.mu.Lock()
	started := p.started
	p.mu.Unlock()

	if !started {
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
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return nil
	}
	p.started = false
	p.mu.Unlock()

	close(p.taskQueue)
	p.wg.Wait()
	close(p.stopChan)

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
