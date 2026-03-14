# Delta for Health

## Purpose

This spec covers improved health check endpoint with dependency status.

---

## ADDED Requirements

### Requirement: Comprehensive Health Check

The system SHALL provide a health check endpoint that reports status of all dependencies.

The endpoint MUST check and report status for:
- Database connectivity
- Worker pool status

#### Scenario: All dependencies healthy

- GIVEN database is connected and worker pool is running
- WHEN client requests `GET /health`
- THEN response status is 200 OK
- AND response body includes `"status": "healthy"`
- AND `"dependencies"` includes:
  - `"database": "healthy"`
  - `"worker_pool": "healthy"`

#### Scenario: Database unhealthy

- GIVEN database connection is down
- WHEN client requests `GET /health`
- THEN response status is 503 Service Unavailable
- AND response body includes `"status": "unhealthy"`
- AND `"dependencies"` includes `"database": "unhealthy"` with error details

#### Scenario: Worker pool unhealthy

- GIVEN worker pool has crashed or is not processing
- WHEN client requests `GET /health`
- THEN response includes `"worker_pool": "unhealthy"` with details
- AND overall status reflects the unhealthy dependency

---

### Requirement: Health Check Response Format

The health check endpoint SHALL return JSON with consistent structure.

```json
{
  "status": "healthy",
  "timestamp": "2026-03-13T12:00:00Z",
  "dependencies": {
    "database": {
      "status": "healthy",
      "message": "connected"
    },
    "worker_pool": {
      "status": "healthy",
      "message": "running",
      "workers": 4,
      "queue_depth": 0
    }
  }
}
```

#### Scenario: Health check response structure

- GIVEN health endpoint is called
- WHEN response is returned
- THEN response includes status, timestamp, and dependency details

---

### Requirement: Database Health Check

The system SHALL verify database connectivity for health check.

#### Scenario: Database ping succeeds

- GIVEN database is reachable
- WHEN health check runs
- THEN database check returns healthy

#### Scenario: Database ping fails

- GIVEN database is not reachable
- WHEN health check runs
- THEN database check returns unhealthy
- AND error message explains the failure

---

### Requirement: Worker Pool Health Check

The system SHALL verify worker pool is operational for health check.

#### Scenario: Worker pool running

- GIVEN worker pool has active workers
- WHEN health check runs
- THEN worker pool check returns healthy
- AND includes current queue depth

#### Scenario: Worker pool not responding

- GIVEN worker pool workers have crashed
- WHEN health check runs
- THEN worker pool check returns unhealthy
- AND includes error details

---

## MODIFIED Requirements

### Requirement: Health Check Endpoint (Previously: Basic ping)

The health check endpoint SHALL return comprehensive dependency status.

(Previously: Only returned basic OK response without dependency checks)

---

## Testing Requirements

### Requirement: Health Check Tests

The health check endpoint MUST have tests for all scenarios.

#### Scenario: Test database healthy

- GIVEN mocked database that returns healthy
- WHEN health endpoint is called
- THEN returns 200 with healthy status

#### Scenario: Test database unhealthy

- GIVEN mocked database that fails
- WHEN health endpoint is called
- THEN returns 503 with unhealthy status

#### Scenario: Test worker pool healthy

- GIVEN mocked worker pool that is running
- WHEN health endpoint is called
- THEN worker pool status is healthy

#### Scenario: Test worker pool unhealthy

- GIVEN mocked worker pool that is crashed
- WHEN health endpoint is called
- THEN returns appropriate unhealthy status
