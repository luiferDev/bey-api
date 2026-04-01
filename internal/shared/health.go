package shared

import (
	"time"

	"gorm.io/gorm"

	"bey/internal/shared/cache"
)

// HealthResponse represents the health check response structure
type HealthResponse struct {
	Status       string                      `json:"status"`
	Timestamp    time.Time                   `json:"timestamp"`
	Dependencies map[string]DependencyStatus `json:"dependencies"`
}

// DependencyStatus represents the status of a single dependency
type DependencyStatus struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	Workers    int    `json:"workers,omitempty"`
	QueueDepth int    `json:"queue_depth,omitempty"`
}

// DatabaseHealthCheck checks database connectivity
func DatabaseHealthCheck(db *gorm.DB) DependencyStatus {
	sqlDB, err := db.DB()
	if err != nil {
		return DependencyStatus{
			Status:  "unhealthy",
			Message: "failed to get database connection",
		}
	}

	if err := sqlDB.Ping(); err != nil {
		return DependencyStatus{
			Status:  "unhealthy",
			Message: "database ping failed: " + err.Error(),
		}
	}

	return DependencyStatus{
		Status:  "healthy",
		Message: "connected",
	}
}

// RedisHealthCheck checks Redis connectivity
func RedisHealthCheck(pool *cache.RedisPool) DependencyStatus {
	if pool == nil {
		return DependencyStatus{
			Status:  "unhealthy",
			Message: "redis pool not initialized",
		}
	}

	if err := pool.Ping(); err != nil {
		return DependencyStatus{
			Status:  "unhealthy",
			Message: "redis ping failed: " + err.Error(),
		}
	}

	return DependencyStatus{
		Status:  "healthy",
		Message: "connected",
	}
}

// WorkerPoolHealthCheck checks worker pool status
// workerCount: number of workers configured
// queueDepth: current queue depth (length of task channel)
// isRunning: whether the pool has been started
func WorkerPoolHealthCheck(workerCount int, queueDepth int, isRunning bool) DependencyStatus {
	if !isRunning {
		return DependencyStatus{
			Status:  "unhealthy",
			Message: "worker pool is not running",
		}
	}

	return DependencyStatus{
		Status:     "healthy",
		Message:    "running",
		Workers:    workerCount,
		QueueDepth: queueDepth,
	}
}

// PerformHealthCheck runs all health checks and returns the overall status
func PerformHealthCheck(db *gorm.DB, workerCount int, queueDepth int, isRunning bool, redisPool *cache.RedisPool) HealthResponse {
	deps := make(map[string]DependencyStatus)

	// Check database
	deps["database"] = DatabaseHealthCheck(db)

	// Check worker pool
	deps["worker_pool"] = WorkerPoolHealthCheck(workerCount, queueDepth, isRunning)

	// Check Redis (if pool is provided)
	if redisPool != nil {
		deps["redis"] = RedisHealthCheck(redisPool)
	}

	// Determine overall status
	overallStatus := "healthy"
	for _, dep := range deps {
		if dep.Status == "unhealthy" {
			overallStatus = "unhealthy"
			break
		}
	}

	return HealthResponse{
		Status:       overallStatus,
		Timestamp:    time.Now().UTC(),
		Dependencies: deps,
	}
}
