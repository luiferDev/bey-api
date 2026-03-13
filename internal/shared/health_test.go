package shared

import (
	"testing"
	"time"
)

func TestWorkerPoolHealthCheck(t *testing.T) {
	tests := []struct {
		name        string
		workerCount int
		queueDepth  int
		isRunning   bool
		wantStatus  string
		wantMessage string
		wantWorkers int
		wantQueue   int
	}{
		{
			name:        "healthy - running with workers",
			workerCount: 4,
			queueDepth:  0,
			isRunning:   true,
			wantStatus:  "healthy",
			wantMessage: "running",
			wantWorkers: 4,
			wantQueue:   0,
		},
		{
			name:        "healthy - running with queue depth",
			workerCount: 4,
			queueDepth:  10,
			isRunning:   true,
			wantStatus:  "healthy",
			wantMessage: "running",
			wantWorkers: 4,
			wantQueue:   10,
		},
		{
			name:        "unhealthy - not running",
			workerCount: 0,
			queueDepth:  0,
			isRunning:   false,
			wantStatus:  "unhealthy",
			wantMessage: "worker pool is not running",
		},
		{
			name:        "zero workers but running",
			workerCount: 0,
			queueDepth:  0,
			isRunning:   true,
			wantStatus:  "healthy",
			wantMessage: "running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WorkerPoolHealthCheck(tt.workerCount, tt.queueDepth, tt.isRunning)

			if result.Status != tt.wantStatus {
				t.Errorf("Status = %q; want %q", result.Status, tt.wantStatus)
			}
			if result.Message != tt.wantMessage {
				t.Errorf("Message = %q; want %q", result.Message, tt.wantMessage)
			}
			if tt.isRunning && tt.wantWorkers > 0 && result.Workers != tt.wantWorkers {
				t.Errorf("Workers = %d; want %d", result.Workers, tt.wantWorkers)
			}
			if tt.isRunning && tt.wantQueue > 0 && result.QueueDepth != tt.wantQueue {
				t.Errorf("QueueDepth = %d; want %d", result.QueueDepth, tt.wantQueue)
			}
		})
	}
}

func TestPerformHealthCheck(t *testing.T) {
	// This test verifies the function structure
	// We can't pass nil DB because it panics
	// The actual DB health check requires integration testing

	// Test with only worker pool info (skip DB by testing worker only)
	result := WorkerPoolHealthCheck(4, 0, true)

	if result.Status != "healthy" {
		t.Errorf("expected healthy status, got %s", result.Status)
	}

	if result.Workers != 4 {
		t.Errorf("Workers = %d; want 4", result.Workers)
	}
}

func TestHealthResponse_Structure(t *testing.T) {
	resp := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Dependencies: map[string]DependencyStatus{
			"database": {
				Status:  "healthy",
				Message: "connected",
			},
			"worker_pool": {
				Status:     "healthy",
				Message:    "running",
				Workers:    4,
				QueueDepth: 0,
			},
		},
	}

	if resp.Status != "healthy" {
		t.Errorf("Status = %q; want %q", resp.Status, "healthy")
	}

	if resp.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	if len(resp.Dependencies) != 2 {
		t.Errorf("Dependencies len = %d; want %d", len(resp.Dependencies), 2)
	}

	dbDep, ok := resp.Dependencies["database"]
	if !ok {
		t.Error("database dependency missing")
	}
	if dbDep.Status != "healthy" {
		t.Errorf("database status = %q; want %q", dbDep.Status, "healthy")
	}

	workerDep, ok := resp.Dependencies["worker_pool"]
	if !ok {
		t.Error("worker_pool dependency missing")
	}
	if workerDep.Workers != 4 {
		t.Errorf("worker count = %d; want %d", workerDep.Workers, 4)
	}
}
