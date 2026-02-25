# Design: Add Concurrency Support to Bey API

## Technical Approach

Implement concurrency primitives for the Bey API using Go's standard library (`sync`, `context`, `errgroup`). Create a new `internal/concurrency/` package with worker pool, task queue, and task types. Add rate limiting middleware using token bucket algorithm. Extend products and orders modules with async task processing and parallel data fetching. All configuration managed via `config.yaml`.

## Architecture Decisions

### Decision: Worker Pool Implementation

**Choice**: Bounded worker pool using `sync.WaitGroup` and channels
**Alternatives considered**: 
- Go's `golang.org/x/sync/errgroup` for worker pool - rejected as it doesn't provide built-in queue management
- Third-party worker pool libraries (e.g., `github.com/ivpusic/grpool`) - rejected to minimize dependencies
**Rationale**: Using standard library provides full control over worker lifecycle, queue depth limits, and graceful shutdown. Simple and maintainable.

### Decision: Task Queue Interface Design

**Choice**: Interface-based design with `Submit`, `GetStatus`, `Cancel` methods; in-memory implementation
**Alternatives considered**:
- Directly implementing Redis-based queue - rejected for initial phase (out of scope)
- Using channels directly without abstraction - rejected for future Redis migration compatibility
**Rationale**: Interface allows swapping implementations without changing callers. In-memory first for simplicity, designed for Redis migration.

### Decision: Rate Limiter Algorithm

**Choice**: Token bucket algorithm implemented as Gin middleware
**Alternatives considered**:
- Leaky bucket - rejected as less intuitive for burst handling
- Fixed window - rejected for allowing burst traffic at window boundaries
- Sliding window - rejected as more complex to implement correctly
**Rationale**: Token bucket provides smooth rate limiting with burst capability, which matches the spec requirements for burst handling.

### Decision: Parallel Data Fetching

**Choice**: Use `golang.org/x/sync/errgroup` for parallel goroutines with context cancellation
**Alternatives considered**:
- Manual goroutine + WaitGroup - rejected as errgroup provides better error handling
- Sequential fetching - rejected as spec requires parallel fetching for performance
**Rationale**: errgroup handles error propagation (first error wins), supports context cancellation, and is idiomatic Go.

### Decision: Middleware Organization

**Choice**: New file `internal/shared/middleware/ratelimit.go` alongside existing middleware
**Alternatives considered**:
- Separate package `internal/middleware/` - rejected to maintain existing shared pattern
- Inline in routes - rejected as not reusable
**Rationale**: Follows existing project convention of `internal/shared/middleware/`.

## Data Flow

```
                                    ┌─────────────────┐
                                    │   Handler API    │
                                    └────────┬────────┘
                                             │
                     ┌───────────────────────┼───────────────────────┐
                     │                       │                       │
                     ▼                       ▼                       ▼
           ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
           │  Task Queue     │    │ Parallel Fetch  │    │  Rate Limiter  │
           │  (Submit)       │    │ (errgroup)      │    │  (middleware)  │
           └────────┬────────┘    └────────┬────────┘    └────────┬────────┘
                    │                       │                       │
                    ▼                       ▼                       ▼
           ┌─────────────────┐    ┌─────────────────┐
           │  Worker Pool    │    │  Repositories  │
           │  (workers)      │    │  (concurrent)   │
           └─────────────────┘    └─────────────────┘
```

**Order Creation Flow**:
```
POST /api/v1/orders
    │
    ▼
RateLimiter (middleware) ──→ OK/429
    │
    ▼
OrderHandler.CreateOrder()
    │
    ▼
OrderService.SubmitAsyncOrder() ──→ TaskQueue.Submit(task)
    │
    ▼
Return task_id immediately (HTTP 202)
    │
    ▼
Worker Pool processes task
    │
    ├──→ Validate inventory
    ├──→ Process payment
    └──→ Create order in DB
```

**Parallel Product Fetch**:
```
GET /api/v1/products/:id
    │
    ▼
errgroup.Go(func) ──→ productRepo.FindByID()
    │
    errgroup.Go(func) ──→ variantRepo.FindByProductID()
    │
    errgroup.Go(func) ──→ imageRepo.FindByProductID()
    │
    ▼
Combine results ──→ Return response
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/concurrency/worker.go` | Create | Worker pool implementation with configurable worker count, queue depth, graceful shutdown |
| `internal/concurrency/task.go` | Create | Task struct, TaskStatus enum, TaskType definitions |
| `internal/concurrency/task_queue.go` | Create | Task queue interface and in-memory implementation |
| `internal/concurrency/config.go` | Create | Concurrency config struct merged with app config |
| `internal/shared/middleware/ratelimit.go` | Create | Token bucket rate limiter middleware with per-endpoint support |
| `internal/modules/products/repository.go` | Modify | Add `FindByIDWithRelationsParallel` method using errgroup |
| `internal/modules/products/service.go` | Modify | Add `SubmitBulkUpdateTask`, `GetTaskStatus` methods |
| `internal/modules/orders/service.go` | Create | New file with async order processing via task queue |
| `internal/modules/orders/handler.go` | Modify | Update to return 202 for async order creation |
| `internal/config/config.go` | Modify | Add ConcurrencyConfig struct |
| `config.yaml` | Modify | Add worker_pool, rate_limit configuration sections |
| `cmd/api/main.go` | Modify | Initialize worker pool, task queue, register middleware |

## Interfaces / Contracts

### Task Queue Interface
```go
type TaskQueue interface {
    Submit(task *Task) (string, error)
    GetStatus(taskID string) (*Task, error)
    Cancel(taskID string) error
}
```

### Task Structure
```go
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
```

### Worker Pool Interface
```go
type WorkerPool interface {
    Submit(task *Task) error
    Start() error
    Shutdown() error
}
```

### Rate Limiter Config
```go
type RateLimitConfig struct {
    Enabled           bool              `yaml:"enabled"`
    RequestsPerSecond int               `yaml:"requests_per_second"`
    BurstCapacity    int               `yaml:"burst_capacity"`
    EndpointLimits   map[string]int    `yaml:"endpoint_limits,omitempty"`
}
```

### Concurrency Config
```go
type ConcurrencyConfig struct {
    WorkerPoolSize    int `yaml:"worker_pool_size"`
    QueueDepthLimit   int `yaml:"queue_depth_limit"`
    RateLimit         RateLimitConfig `yaml:"rate_limit"`
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Worker pool task processing, queue depth limits, task status transitions | Table-driven tests with mock DB |
| Unit | Rate limiter token bucket algorithm, per-client tracking | Unit tests with mocked time |
| Unit | Parallel fetch errgroup behavior | Tests with controlled goroutine execution |
| Integration | Worker pool graceful shutdown | Integration test with tasks in progress |
| Integration | Rate limiter end-to-end | Test with actual HTTP requests |
| E2E | Async order creation flow | Full API test with task status polling |

## Migration / Rollout

No database migration required. Configuration-driven rollout:

1. Add concurrency config to `config.yaml` (defaults will work if omitted)
2. Initialize worker pool in `main.go` before HTTP server starts
3. Register rate limiter middleware (disabled by default)
4. Add async endpoints alongside sync endpoints
5. Monitor goroutine count via pprof before enabling full traffic

Rollback:
1. Remove `internal/concurrency/` directory
2. Remove `internal/shared/middleware/ratelimit.go`
3. Revert products repository and service changes
4. Remove orders service (restore to handler-only)
5. Remove concurrency config from `config.yaml` and `config.go`

## Open Questions

- [ ] Should task queue persist tasks to database for durability? (not in initial scope, but worth noting)
- [ ] Should there be a health check endpoint for worker pool status?
- [ ] What's the max retry count for failed tasks? (currently undefined)
