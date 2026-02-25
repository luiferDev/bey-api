.PHONY: all build run test lint clean swagger

all: lint test build

build:
	go build -o main ./cmd/api/

run:
	go run ./cmd/api/

test:
	go test -v ./...

test-coverage:
	go test -cover ./...

lint:
	go vet ./...
	golangci-lint run

clean:
	rm -f main api
	rm -f coverage.out

swagger:
	swag init -g cmd/api/main.go -o cmd/api/docs

swagger-docs:
	@echo "Swagger documentation generated at /swagger/index.html"
	@echo "Run 'make server' and visit http://localhost:8080/swagger/index.html"

server: build
	./main

help:
	@echo "Available targets:"
	@echo "  build          - Build the application binary"
	@echo "  run            - Run the application"
	@echo "  test           - Run all tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  lint           - Run linters (go vet, golangci-lint)"
	@echo "  clean          - Remove build artifacts"
	@echo "  swagger        - Generate Swagger documentation"
	@echo "  swagger-docs   - Display Swagger endpoint info"
	@echo "  server         - Build and run the server"
