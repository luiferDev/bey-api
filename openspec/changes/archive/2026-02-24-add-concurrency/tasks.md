# Tasks: Add Concurrency Support to Bey API

## Phase 1: Configuration & Types

- [x] 1.1 Create `internal/concurrency/config.go` with `ConcurrencyConfig`, `RateLimitConfig`, `WorkerPoolConfig` structs matching spec in design.md:193-198
- [x] 1.2 Modify `internal/config/config.go` to add `Concurrency` field of type `ConcurrencyConfig`
- [x] 1.3 Update `config.yaml` to add `concurrency` section with `worker_pool_size`, `queue_depth_limit`, `rate_limit` settings per config/spec.md

## Phase 2: Core Concurrency Infrastructure

- [x] 2.1 Create `internal/concurrency/task.go` with `TaskStatus`, `TaskType` enums and `Task` struct per design.md:140-169
- [x] 2.2 Create `internal/concurrency/worker.go` with `WorkerPool` interface and implementation using WaitGroup and channels per concurrency/spec.md scenarios
- [x] 2.3 Create `internal/concurrency/task_queue.go` with `TaskQueue` interface and in-memory implementation per design.md:131-136, supporting Submit, GetStatus, Cancel methods

## Phase 3: Middleware

- [x] 3.1 Create `internal/shared/middleware/ratelimit.go` with token bucket rate limiter per middleware/spec.md
- [x] 3.2 Implement per-client tracking using IP or auth token
- [x] 3.3 Implement per-endpoint rate limit overrides per spec.md scenario

## Phase 4: Products Module Integration

- [x] 4.1 Modify `internal/modules/products/repository.go` to add `FindByIDWithRelationsParallel` method using `golang.org/x/sync/errgroup` per products/spec.md
- [x] 4.2 Modify `internal/modules/products/service.go` to add `SubmitBulkUpdateTask`, `SubmitBulkCreateTask`, `SubmitBulkDeleteTask` methods per products/spec.md
- [x] 4.3 Add `GetTaskStatus` method to products service per products/spec.md:79-84

## Phase 5: Orders Module Integration

- [x] 5.1 Create `internal/modules/orders/service.go` with async order processing via task queue per orders/spec.md
- [x] 5.2 Add `SubmitAsyncOrder` method that validates and submits order task per orders/spec.md:13-20
- [x] 5.3 Modify `internal/modules/orders/handler.go` to return HTTP 202 with task ID instead of blocking per orders/spec.md
- [x] 5.4 Modify `internal/modules/orders/routes.go` to add task status endpoint

## Phase 6: Application Wiring

- [x] 6.1 Modify `cmd/api/main.go` to initialize worker pool with config values
- [x] 6.2 Initialize task queue in main.go
- [x] 6.3 Register rate limiter middleware in router per design.md
- [x] 6.4 Add graceful shutdown handling with worker pool Shutdown() per concurrency/spec.md:20-26
- [x] 6.5 Validate config values (worker_pool_size > 0) per config/spec.md:61-66

## Phase 7: Testing

- [x] 7.1 Write unit tests for worker pool task processing and queue depth limits per concurrency/spec.md:13-33
- [x] 7.2 Write unit tests for rate limiter token bucket algorithm per middleware/spec.md
- [x] 7.3 Write unit tests for parallel fetch errgroup behavior per concurrency/spec.md:86-104
- [x] 7.4 Write integration tests for graceful shutdown per concurrency/spec.md
- [x] 7.5 Write integration tests for rate limiter end-to-end per middleware/spec.md
- [x] 7.6 Write E2E test for async order creation flow per orders/spec.md

## Phase 8: Documentation & Cleanup

- [x] 8.1 Update `AGENTS.md` with concurrency commands if needed
- [x] 8.2 Verify no goroutine leaks with pprof per proposal success criteria
