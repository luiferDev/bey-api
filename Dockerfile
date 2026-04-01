# syntax=docker/dockerfile:1
# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install git for go modules and build dependencies
RUN apk add --no-cache git gcc musl-dev

# Copy go mod files for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main ./cmd/api

# Final stage
FROM alpine:3.21

# Install ca-certificates for HTTPS requests and timezone data
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .
COPY --from=builder /app/config.yaml .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
