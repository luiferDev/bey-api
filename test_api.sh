#!/bin/bash

# API Test Script for Bey API
# Usage: ./test_api.sh

BASE_URL="http://localhost:8080"

echo "========================================"
echo "  Bey API Test Script"
echo "========================================"
echo ""

# Test 1: Health Check
echo "1. Testing Health Check..."
curl -s -w "\nHTTP Status: %{http_code}\n" "${BASE_URL}/health"
echo ""

# Test 2: Get All Users
echo "2. Testing Get All Users..."
curl -s -w "\nHTTP Status: %{http_code}\n" "${BASE_URL}/api/v1/users"
echo ""

# Test 3: Create User
echo "3. Testing Create User..."
curl -s -w "\nHTTP Status: %{http_code}\n" -X POST \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","name":"Test User","password":"password123"}' \
  "${BASE_URL}/api/v1/users"
echo ""

# Test 4: Get All Products
echo "4. Testing Get All Products..."
curl -s -w "\nHTTP Status: %{http_code}\n" "${BASE_URL}/api/v1/products"
echo ""

# Test 5: Create Product
echo "5. Testing Create Product..."
curl -s -w "\nHTTP Status: %{http_code}\n" -X POST \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Product","description":"A test product","price":29.99}' \
  "${BASE_URL}/api/v1/products"
echo ""

# Test 6: Get All Orders
echo "6. Testing Get All Orders..."
curl -s -w "\nHTTP Status: %{http_code}\n" "${BASE_URL}/api/v1/orders"
echo ""

# Test 7: Get All Inventory
echo "7. Testing Get All Inventory..."
curl -s -w "\nHTTP Status: %{http_code}\n" "${BASE_URL}/api/v1/inventory"
echo ""

echo "========================================"
echo "  Tests Complete"
echo "========================================"
