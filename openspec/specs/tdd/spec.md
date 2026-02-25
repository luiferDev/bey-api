# TDD Workflow Specification

## Purpose

This specification defines the requirements for implementing Test-Driven Development (TDD) in the Bey API project. TDD ensures reliable, testable code by writing tests before implementation.

## Requirements

### Requirement: Test File Structure

All new Go code MUST have corresponding test files following the `*_test.go` naming convention.

Test files MUST be placed in the same package as the code being tested.

#### Scenario: Creating a new handler

- GIVEN a new handler file `products/handler.go` is created
- WHEN implementing new functionality
- THEN a corresponding `products/handler_test.go` file MUST exist
- AND tests MUST be written BEFORE the implementation is complete

#### Scenario: Test file follows Go conventions

- GIVEN a test file exists
- WHEN the file is inspected
- THEN it MUST have the `_test.go` suffix
- AND MUST include the `package` statement matching the package being tested

### Requirement: Test-Driven Development Cycle

Developers MUST follow the red-green-refactor TDD cycle:

1. Write a failing test (red)
2. Write minimal code to make the test pass (green)
3. Refactor while keeping tests passing (refactor)

#### Scenario: TDD cycle - failing test first

- GIVEN implementing a new feature
- WHEN starting work on the feature
- THEN a test that describes the expected behavior MUST be written first
- AND the test MUST fail initially (red phase)

#### Scenario: TDD cycle - making test pass

- GIVEN a failing test exists
- WHEN implementing the feature
- THEN the code MUST be written to make the test pass
- AND no other tests MUST break in the process

### Requirement: Handler Tests

All HTTP handler functions MUST have corresponding test cases.

Handler tests MUST verify the handler's response for both success and failure scenarios.

#### Scenario: Handler success response test

- GIVEN a handler function exists
- WHEN testing a successful request
- THEN the test MUST verify the HTTP status code is correct
- AND the response body MUST match expected structure

#### Scenario: Handler error response test

- GIVEN a handler function exists
- WHEN testing an invalid request
- THEN the test MUST verify the appropriate error status code (4xx)
- AND the error message MUST be present in the response

#### Scenario: Handler input validation test

- GIVEN a handler with required fields exists
- WHEN a request is made with missing required fields
- THEN the handler MUST return HTTP 400 Bad Request
- AND the response MUST indicate which fields are missing

### Requirement: Repository Tests

Repository functions that interact with the database SHOULD have test coverage.

Tests SHOULD use mocks or an in-memory database for isolation.

#### Scenario: Repository FindByID test

- GIVEN a repository with FindByID method
- WHEN testing with a valid ID
- THEN the test MUST verify the returned object matches expected data

#### Scenario: Repository not found case

- GIVEN a repository with FindByID method
- WHEN testing with a non-existent ID
- THEN the test MUST verify nil is returned
- AND no error is returned

### Requirement: Table-Driven Tests

Tests with multiple test cases SHOULD use table-driven test patterns.

Table-driven tests MUST define test cases in a slice of structs.

#### Scenario: Table-driven handler test

- GIVEN a handler with multiple input scenarios
- WHEN writing tests
- THEN a table of test cases SHOULD be defined
- AND each case MUST have: name, input, expected status, expected response

#### Scenario: Running table-driven tests

- GIVEN table-driven tests exist
- WHEN `go test -v` is executed
- THEN each test case MUST run independently
- AND each case MUST report pass/fail status

### Requirement: Test Coverage

New code SHOULD achieve greater than 80% test coverage on handler logic.

Coverage reports SHOULD be generated using `go test -cover`.

#### Scenario: Coverage report generation

- GIVEN tests exist in the project
- WHEN `go test -cover ./...` is executed
- THEN a coverage report MUST be generated
- AND the output MUST show per-package coverage percentages

### Requirement: Test Execution

All tests MUST pass before code is considered complete.

Running `go test ./...` MUST result in zero failures.

#### Scenario: All tests pass

- GIVEN all test files are implemented
- WHEN `go test ./...` is executed
- THEN the exit code MUST be 0
- AND all tests MUST show PASS status

#### Scenario: Test failure indicates regression

- GIVEN existing passing tests
- WHEN new code breaks an existing test
- THEN the test failure MUST be fixed before merging
- AND the failing test MUST not be bypassed

### Requirement: Mock Dependencies

Tests for handlers and repositories SHOULD use mocks for external dependencies.

Mocks SHOULD be created using interfaces or a mocking library.

#### Scenario: Mocking repository in handler test

- GIVEN a handler depends on a repository interface
- WHEN writing handler tests
- THEN a mock implementation of the repository SHOULD be used
- AND the mock MUST return controlled responses for testing

## Out of Scope (Not Required)

- Legacy code test coverage (existing code without tests)
- Integration tests with real database
- Performance/load testing
- Contract testing
