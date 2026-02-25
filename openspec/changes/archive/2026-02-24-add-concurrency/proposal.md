# Proposal: Add Concurrency Support to Bey API

## Intent

The Bey API currently handles requests synchronously without background processing capabilities. As the e-commerce platform grows, certain operations like bulk inventory updates, order processing, and report generation will require asynchronous execution to improve response times and system reliability. This change introduces concurrency primitives including a worker pool, async task processing, and parallel data fetching capabilities.

## Scope

### In Scope
- Background worker pool implementation for async task processing
- Task queue system with job persistence (in-memory with optional DB persistence)
- Parallel product data fetching (product + variants + images in concurrent goroutines)
- Rate limiting middleware to control concurrent request throughput
- Configuration for worker pool size and queue depth

### Out of Scope
- Distributed task queue (Redis-based)
- WebSocket support for real-time task status
- Scheduled/cron jobs
- Message broker integration (RabbitMQ, Kafka)

## Approach

### Architecture
1. **Worker Pool**: Implement a bounded worker pool pattern using Go's `sync.WaitGroup` and channels. Workers process tasks from a shared queue.
2. **Task Queue**: Create a task queue interface with `Submit(task)`, `GetStatus(id)`, and `Cancel(id)` methods. Initial implementation uses in-memory queue; designed for future Redis migration.
3. **Parallel Fetching**: Add concurrent repository methods that use Go's `errgroup` for parallel data loading with proper error handling.
4. **Rate Limiter**: Implement token bucket rate limiter as Gin middleware, configurable per-endpoint.

### Implementation Pattern
- New `internal/concurrency/` package with:
  - `worker.go`: Worker pool implementation
  - `task.go`: Task definitions and status
  - `task_queue.go`: Queue management
- New `internal/middleware/ratelimit.go`: Rate limiting
- Extend repositories with parallel fetch methods

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/concurrency/` | New | Worker pool, task queue, task types |
| `internal/middleware/ratelimit.go` | New | Rate limiting middleware |
| `internal/modules/products/repository.go` | Modified | Add parallel fetch methods |
| `internal/modules/products/service.go` | Modified | Use async task for bulk operations |
| `internal/modules/orders/service.go` | Modified | Use async task for order processing |
| `config.yaml` | Modified | Add concurrency config (worker_pool_size, rate_limit) |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Memory pressure from unbounded queues | Medium | Add configurable queue depth limits with backpressure |
| Goroutine leaks on shutdown | Low | Graceful shutdown with WaitGroup in worker pool |
| Race conditions in shared state | Low | Use mutexes for task queue; avoid shared state in workers |
| Database connection exhaustion | Medium | Worker pool size should be ≤ database max_open_conns |

## Rollback Plan

1. Remove `internal/concurrency/` directory
2. Remove `internal/middleware/ratelimit.go`
3. Revert repository changes to synchronous methods
4. Remove concurrency config from `config.yaml`
5. Rollback is low-risk as no database schema changes are required

## Dependencies

- Go 1.25+ standard library (`sync`, `context`, `errgroup`)
- No new external dependencies required for initial implementation

## Success Criteria

- [ ] Worker pool processes background tasks without blocking API responses
- [ ] Product detail endpoint with parallel fetching responds faster than sequential
- [ ] Rate limiter correctly limits requests per second
- [ ] Graceful shutdown completes all pending tasks
- [ ] No goroutine leaks under load (verified via pprof)
