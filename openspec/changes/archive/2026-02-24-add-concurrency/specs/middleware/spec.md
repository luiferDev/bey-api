# Middleware Specification

## Purpose

This specification defines the middleware components for the Bey API, specifically the rate limiting functionality to control concurrent request throughput and protect the system from overload.

## Requirements

### Requirement: Rate Limiting Middleware

The system SHALL implement a token bucket rate limiter as Gin middleware. The rate limiter MUST be configurable per-endpoint and limit requests based on tokens replenishing at a defined rate.

#### Scenario: Requests within rate limit succeed

- GIVEN a rate limiter configured with 100 requests per second
- WHEN 50 requests are made within one second
- THEN all requests SHALL pass through the middleware
- AND the response SHALL be handled normally

#### Scenario: Requests exceeding rate limit are rejected

- GIVEN a rate limiter configured with 10 requests per second
- WHEN 15 requests are made within one second
- THEN the first 10 requests SHALL pass through
- AND the remaining 5 requests SHALL receive HTTP 429 (Too Many Requests) response
- AND the response SHALL include Retry-After header

#### Scenario: Rate limiter tracks per-client

- GIVEN a rate limiter configured with 10 requests per second per client
- WHEN client A makes 8 requests and client B makes 8 requests within one second
- THEN both clients SHALL have 8 requests pass through
- AND neither SHALL be rate limited based on the other's requests

#### Scenario: Rate limiter allows burst traffic

- GIVEN a rate limiter configured with 10 requests per second and burst capacity of 20
- WHEN 20 requests are made simultaneously
- THEN all 20 requests SHALL pass through
- AND subsequent requests SHALL be rate limited until tokens replenish

### Requirement: Rate Limiter Configuration

The system SHALL support configurable rate limits per endpoint or globally. Configuration MUST include requests per second and burst capacity.

#### Scenario: Global rate limit applies to all endpoints

- GIVEN a global rate limit of 100 req/s configured
- WHEN a request is made to any endpoint
- THEN the rate limiter SHALL apply the global limit

#### Scenario: Endpoint-specific rate limit overrides global

- GIVEN global rate limit of 100 req/s and endpoint-specific limit of 10 req/s for /api/v1/orders
- WHEN requests are made to /api/v1/orders
- THEN the endpoint-specific limit of 10 req/s SHALL apply
- AND other endpoints SHALL use the global limit of 100 req/s

### Requirement: Rate Limiter Graceful Degradation

The system SHOULD continue operating even if the rate limiter encounters errors. Failed rate limiter checks SHOULD NOT block requests.

#### Scenario: Rate limiter error does not block requests

- GIVEN a rate limiter that encounters an internal error
- WHEN a request is processed
- THEN the request SHALL pass through (fail-open behavior)
- AND an error SHALL be logged for monitoring
