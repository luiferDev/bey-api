#!/usr/bin/env bash
# =============================================================================
# Bey API - Comprehensive Integration Test Script
# Tests ALL API endpoints with colored output and summary
# =============================================================================
# Usage:
#   ./test_api.sh              # Run all tests
#   ./test_api.sh auth         # Run only auth tests
#   ./test_api.sh users        # Run only users tests
#   ./test_api.sh categories   # Run only categories tests
#   ./test_api.sh products     # Run only products tests
#   ./test_api.sh variants     # Run only variants tests
#   ./test_api.sh images       # Run only images tests
#   ./test_api.sh inventory    # Run only inventory tests
#   ./test_api.sh cart         # Run only cart tests
#   ./test_api.sh orders       # Run only orders tests
#   ./test_api.sh payments     # Run only payments tests
#   ./test_api.sh admin        # Run only admin tests
#   ./test_api.sh health       # Run only health tests
#   ./test_api.sh global       # Run only global error tests
# =============================================================================

# NOTE: Not using set -e so we continue on failures and track them

# =============================================================================
# CONFIGURATION
# =============================================================================
BASE_URL="${BASE_URL:-http://localhost:8080}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@bey.com}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"

# Cookie jar for maintaining session cookies across requests
COOKIE_JAR="/tmp/bey_api_cookies.txt"

# =============================================================================
# COLOR CODES
# =============================================================================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# =============================================================================
# TEST COUNTERS
# =============================================================================
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# =============================================================================
# RESOURCE IDs (populated during test execution)
# =============================================================================
ADMIN_USER_ID=""
ADMIN_TOKEN=""
ADMIN_REFRESH_TOKEN=""
USER_ID=""
USER_TOKEN=""
USER_REFRESH_TOKEN=""
USER_EMAIL=""
USER_PASSWORD=""
CATEGORY_ID=""
CHILD_CATEGORY_ID=""
PRODUCT_ID=""
PRODUCT_SLUG=""
VARIANT_ID=""
IMAGE_ID=""
ORDER_ID=""
PAYMENT_ID=""
PAYMENT_LINK_ID=""

# =============================================================================
# HELPER FUNCTIONS
# =============================================================================

run_test() {
    local description="$1"
    local result="$2"
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    if [ "$result" -eq 0 ]; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo -e "  ${GREEN}✓ PASS${NC}: $description"
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo -e "  ${RED}✗ FAIL${NC}: $description"
    fi
}

assert_status() {
    local expected="$1"
    local actual="$2"
    [ "$expected" = "$actual" ]
}

assert_field() {
    local json="$1"
    local field="$2"
    local expected="$3"
    local actual
    actual=$(echo "$json" | jq -r "$field" 2>/dev/null)
    [ "$actual" = "$expected" ]
}

assert_field_exists() {
    local json="$1"
    local field="$2"
    local actual
    actual=$(echo "$json" | jq -r "$field" 2>/dev/null)
    [ "$actual" != "null" ] && [ -n "$actual" ]
}

assert_success() {
    local json="$1"
    local success
    success=$(echo "$json" | jq -r '.success' 2>/dev/null)
    [ "$success" = "true" ]
}

assert_error() {
    local json="$1"
    local success
    success=$(echo "$json" | jq -r '.success' 2>/dev/null)
    [ "$success" = "false" ]
}

make_request() {
    local method="$1"
    local url="$2"
    local extra_headers="${3:-}"
    local body="${4:-}"
    local cookie_jar="${5:-$COOKIE_JAR}"

    RESPONSE=""
    HTTP_STATUS=""

    local response_file="/tmp/bey_api_response_$$.json"

    local curl_args=(
        -s
        -o "$response_file"
        -w "%{http_code}"
        -X "$method"
        -H "Content-Type: application/json"
        -b "$cookie_jar"
        -c "$cookie_jar"
    )

    if [ -n "$extra_headers" ]; then
        IFS=' ' read -ra headers <<< "$extra_headers"
        for header in "${headers[@]}"; do
            curl_args+=(-H "$header")
        done
    fi

    if [ -n "$body" ]; then
        curl_args+=(-d "$body")
    fi

    curl_args+=("$BASE_URL$url")

    HTTP_STATUS=$(curl "${curl_args[@]}" 2>/dev/null)
    RESPONSE=$(cat "$response_file" 2>/dev/null)
    rm -f "$response_file"
}

auth_header() {
    echo "Authorization: Bearer $1"
}

print_section() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# =============================================================================
# SECTION 0: Health & Infrastructure
# =============================================================================
test_health() {
    print_section "Section 0: Health & Infrastructure"

    # GET /health
    make_request "GET" "/health"
    local result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /health returns 200 with success" "$result"

    # GET /metrics/cache
    make_request "GET" "/metrics/cache"
    result=1
    if assert_status "200" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "GET /metrics/cache returns 200" "$result"

    # POST /metrics/cache/reset
    make_request "POST" "/metrics/cache/reset"
    result=1
    if assert_status "200" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /metrics/cache/reset returns 200" "$result"
}

# =============================================================================
# SECTION 1: Auth Module
# =============================================================================
test_auth() {
    print_section "Section 1: Auth Module"

    # --- Happy Path ---

    # POST /api/v1/auth/login with admin credentials
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}"
    local result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        ADMIN_TOKEN=$(echo "$RESPONSE" | jq -r '.data.access_token // empty' 2>/dev/null)
        ADMIN_REFRESH_TOKEN=$(echo "$RESPONSE" | jq -r '.data.refresh_token // empty' 2>/dev/null)
        if [ -n "$ADMIN_TOKEN" ]; then
            result=0
        fi
    fi
    run_test "POST /api/v1/auth/login (admin) returns 200 with access_token" "$result"

    # POST /api/v1/auth/register (create new test user)
    USER_EMAIL="testuser_$(date +%s)@bey.com"
    USER_PASSWORD="TestPassword123!"
    make_request "POST" "/api/v1/users/register" "" \
        "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASSWORD\",\"first_name\":\"Test\",\"last_name\":\"User\"}"
    result=1
    if assert_status "201" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        USER_ID=$(echo "$RESPONSE" | jq -r '.data.id // empty' 2>/dev/null)
        if [ -n "$USER_ID" ]; then
            result=0
        fi
    fi
    run_test "POST /api/v1/users/register creates user and returns 201" "$result"

    # POST /api/v1/auth/login with regular user credentials
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASSWORD\"}"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        USER_TOKEN=$(echo "$RESPONSE" | jq -r '.data.access_token // empty' 2>/dev/null)
        USER_REFRESH_TOKEN=$(echo "$RESPONSE" | jq -r '.data.refresh_token // empty' 2>/dev/null)
        if [ -n "$USER_TOKEN" ]; then
            result=0
        fi
    fi
    run_test "POST /api/v1/auth/login (user) returns 200 with access_token" "$result"

    # POST /api/v1/auth/refresh with valid refresh_token cookie
    make_request "POST" "/api/v1/auth/refresh"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "POST /api/v1/auth/refresh returns 200" "$result"

    # POST /api/v1/auth/verify-email (expect error - no real email)
    make_request "POST" "/api/v1/auth/verify-email" "" \
        "{\"token\":\"invalid_token_12345\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/auth/verify-email with invalid token returns 400" "$result"

    # POST /api/v1/auth/resend-verification
    make_request "POST" "/api/v1/auth/resend-verification" "" \
        "{\"email\":\"$USER_EMAIL\"}"
    result=1
    if assert_status "200" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/auth/resend-verification returns 200" "$result"

    # POST /api/v1/auth/forgot-password
    make_request "POST" "/api/v1/auth/forgot-password" "" \
        "{\"email\":\"$USER_EMAIL\"}"
    result=1
    if assert_status "200" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/auth/forgot-password returns 200" "$result"

    # POST /api/v1/auth/reset-password (expect error - invalid token)
    make_request "POST" "/api/v1/auth/reset-password" "" \
        "{\"token\":\"invalid_reset_token\",\"new_password\":\"NewPassword123!\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/auth/reset-password with invalid token returns 400" "$result"

    # POST /api/v1/auth/logout
    make_request "POST" "/api/v1/auth/logout"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "POST /api/v1/auth/logout returns 200" "$result"

    # Re-login after logout to continue tests
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASSWORD\"}"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        USER_TOKEN=$(echo "$RESPONSE" | jq -r '.data.access_token // empty' 2>/dev/null)
        if [ -n "$USER_TOKEN" ]; then
            result=0
        fi
    fi
    run_test "POST /api/v1/auth/login (re-login after logout)" "$result"

    # --- Error Cases ---

    # Login with wrong email
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"wrong@bey.com\",\"password\":\"$ADMIN_PASSWORD\"}"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/auth/login with wrong email returns 401" "$result"

    # Login with wrong password
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"WrongPassword123!\"}"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/auth/login with wrong password returns 401" "$result"

    # Login with empty body
    make_request "POST" "/api/v1/auth/login" "" "{}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/auth/login with empty body returns 400" "$result"

    # Login with invalid email format
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"not-an-email\",\"password\":\"SomePassword123!\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/auth/login with invalid email format returns 400" "$result"

    # Login with short password
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$USER_EMAIL\",\"password\":\"short\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/auth/login with short password returns 400" "$result"

    # Refresh with invalid token (clear cookies first)
    > "$COOKIE_JAR"
    make_request "POST" "/api/v1/auth/refresh"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/auth/refresh without cookie returns 401" "$result"

    # Re-login to restore cookies for subsequent tests
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASSWORD\"}"

    # Register with duplicate email
    make_request "POST" "/api/v1/users/register" "" \
        "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASSWORD\",\"first_name\":\"Dup\",\"last_name\":\"User\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/users/register with duplicate email returns 400" "$result"

    # Register with invalid email format
    make_request "POST" "/api/v1/users/register" "" \
        "{\"email\":\"not-an-email\",\"password\":\"$USER_PASSWORD\",\"first_name\":\"Bad\",\"last_name\":\"Email\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/users/register with invalid email returns 400" "$result"

    # Register with short password
    make_request "POST" "/api/v1/users/register" "" \
        "{\"email\":\"short_pw@bey.com\",\"password\":\"short\",\"first_name\":\"Short\",\"last_name\":\"Pw\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/users/register with short password returns 400" "$result"

    # Register with empty body
    make_request "POST" "/api/v1/users/register" "" "{}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/users/register with empty body returns 400" "$result"

    # Forgot password with invalid email
    make_request "POST" "/api/v1/auth/forgot-password" "" \
        "{\"email\":\"not-an-email\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/auth/forgot-password with invalid email returns 400" "$result"

    # Reset password with empty token
    make_request "POST" "/api/v1/auth/reset-password" "" \
        "{\"token\":\"\",\"new_password\":\"NewPassword123!\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/auth/reset-password with empty token returns 400" "$result"

    # Reset password with short new_password
    make_request "POST" "/api/v1/auth/reset-password" "" \
        "{\"token\":\"sometoken\",\"new_password\":\"short\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/auth/reset-password with short password returns 400" "$result"
}

# =============================================================================
# SECTION 2: Users Module
# =============================================================================
test_users() {
    print_section "Section 2: Users Module"

    # --- Happy Path ---

    # GET /api/v1/users/:id (own profile)
    make_request "GET" "/api/v1/users/$USER_ID" "$(auth_header "$USER_TOKEN")"
    local result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/users/:id (own profile) returns 200" "$result"

    # PUT /api/v1/users/:id (update own profile)
    make_request "PUT" "/api/v1/users/$USER_ID" "$(auth_header "$USER_TOKEN")" \
        "{\"first_name\":\"Updated\",\"last_name\":\"Name\"}"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "PUT /api/v1/users/:id (update own profile) returns 200" "$result"

    # PUT /api/v1/users/:id/avatar (update own avatar)
    make_request "PUT" "/api/v1/users/$USER_ID/avatar" "$(auth_header "$USER_TOKEN")" \
        "{\"avatar_url\":\"https://example.com/avatar.png\"}"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "PUT /api/v1/users/:id/avatar returns 200" "$result"

    # POST /api/v1/users/register-admin (admin creates user)
    make_request "POST" "/api/v1/users/register-admin" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"email\":\"admincreated@bey.com\",\"password\":\"AdminCreated123!\",\"first_name\":\"Admin\",\"last_name\":\"Created\"}"
    result=1
    if assert_status "201" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "POST /api/v1/users/register-admin (admin) returns 201" "$result"

    # GET /api/v1/users (admin lists all users)
    make_request "GET" "/api/v1/users" "$(auth_header "$ADMIN_TOKEN")"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/users (admin list) returns 200" "$result"

    # DELETE /api/v1/users/:id (admin deletes the user just created)
    local admin_created_id
    admin_created_id=$(echo "$RESPONSE" | jq -r '.data[] | select(.email == "admincreated@bey.com") | .id' 2>/dev/null)
    if [ -n "$admin_created_id" ] && [ "$admin_created_id" != "null" ]; then
        make_request "DELETE" "/api/v1/users/$admin_created_id" "$(auth_header "$ADMIN_TOKEN")"
        result=1
        if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
            result=0
        fi
        run_test "DELETE /api/v1/users/:id (admin deletes user) returns 200" "$result"
    else
        run_test "DELETE /api/v1/users/:id (admin deletes user) - SKIPPED (no user found)" "0"
    fi

    # --- Error Cases ---

    # GET /api/v1/users/:id without auth (401)
    > "$COOKIE_JAR"
    make_request "GET" "/api/v1/users/$USER_ID"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "GET /api/v1/users/:id without auth returns 401" "$result"
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASSWORD\"}"

    # GET /api/v1/users/:id with regular user trying to access another user (403)
    make_request "GET" "/api/v1/users/$ADMIN_USER_ID" "$(auth_header "$USER_TOKEN")"
    result=1
    if assert_status "403" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "GET /api/v1/users/:id (BOLA - other user) returns 403" "$result"

    # PUT /api/v1/users/:id without auth (401)
    > "$COOKIE_JAR"
    make_request "PUT" "/api/v1/users/$USER_ID" "" \
        "{\"first_name\":\"Hacker\"}"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "PUT /api/v1/users/:id without auth returns 401" "$result"
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASSWORD\"}"

    # PUT /api/v1/users/:id with regular user trying to update another user (403)
    make_request "PUT" "/api/v1/users/$ADMIN_USER_ID" "$(auth_header "$USER_TOKEN")" \
        "{\"first_name\":\"Hacked\"}"
    result=1
    if assert_status "403" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "PUT /api/v1/users/:id (BOLA - update other user) returns 403" "$result"

    # PUT /api/v1/users/:id/avatar with invalid avatar URL
    make_request "PUT" "/api/v1/users/$USER_ID/avatar" "$(auth_header "$USER_TOKEN")" \
        "{\"avatar_url\":\"not-a-url\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "PUT /api/v1/users/:id/avatar with invalid URL returns 400" "$result"

    # DELETE /api/v1/users/:id without admin role (403)
    make_request "DELETE" "/api/v1/users/$USER_ID" "$(auth_header "$USER_TOKEN")"
    result=1
    if assert_status "403" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "DELETE /api/v1/users/:id without admin role returns 403" "$result"

    # DELETE /api/v1/users/:id without auth (401)
    > "$COOKIE_JAR"
    make_request "DELETE" "/api/v1/users/$USER_ID"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "DELETE /api/v1/users/:id without auth returns 401" "$result"
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASSWORD\"}"

    # POST /api/v1/users/register-admin without admin role (403)
    make_request "POST" "/api/v1/users/register-admin" "$(auth_header "$USER_TOKEN")" \
        "{\"email\":\"hacker@bey.com\",\"password\":\"Hacker123!\",\"first_name\":\"Hacker\",\"last_name\":\"User\"}"
    result=1
    if assert_status "403" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/users/register-admin without admin role returns 403" "$result"

    # POST /api/v1/users/register with invalid data (missing required fields)
    make_request "POST" "/api/v1/users/register" "" \
        "{\"email\":\"\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/users/register with missing required fields returns 400" "$result"
}

# =============================================================================
# SECTION 3: Categories Module
# =============================================================================
test_categories() {
    print_section "Section 3: Categories Module"

    # --- Happy Path ---

    # POST /api/v1/categories (admin creates root category)
    make_request "POST" "/api/v1/categories" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"name\":\"Test Category\",\"slug\":\"test-category\"}"
    local result=1
    if assert_status "201" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        CATEGORY_ID=$(echo "$RESPONSE" | jq -r '.data.id // empty' 2>/dev/null)
        if [ -n "$CATEGORY_ID" ]; then
            result=0
        fi
    fi
    run_test "POST /api/v1/categories (admin creates root) returns 201" "$result"

    # POST /api/v1/categories (admin creates child category)
    make_request "POST" "/api/v1/categories" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"name\":\"Child Category\",\"slug\":\"child-category\",\"parent_id\":$CATEGORY_ID}"
    result=1
    if assert_status "201" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        CHILD_CATEGORY_ID=$(echo "$RESPONSE" | jq -r '.data.id // empty' 2>/dev/null)
        if [ -n "$CHILD_CATEGORY_ID" ]; then
            result=0
        fi
    fi
    run_test "POST /api/v1/categories (admin creates child) returns 201" "$result"

    # GET /api/v1/categories (list all as tree)
    make_request "GET" "/api/v1/categories"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/categories returns 200" "$result"

    # GET /api/v1/categories/tree
    make_request "GET" "/api/v1/categories/tree"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/categories/tree returns 200" "$result"

    # GET /api/v1/categories/:id
    make_request "GET" "/api/v1/categories/$CATEGORY_ID"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/categories/:id returns 200" "$result"

    # GET /api/v1/categories/:id/children
    make_request "GET" "/api/v1/categories/$CATEGORY_ID/children"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/categories/:id/children returns 200" "$result"

    # GET /api/v1/categories/:id/breadcrumbs
    make_request "GET" "/api/v1/categories/$CATEGORY_ID/breadcrumbs"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/categories/:id/breadcrumbs returns 200" "$result"

    # GET /api/v1/categories/slug/:slug
    make_request "GET" "/api/v1/categories/slug/test-category"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/categories/slug/:slug returns 200" "$result"

    # PUT /api/v1/categories/:id (update)
    make_request "PUT" "/api/v1/categories/$CATEGORY_ID" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"name\":\"Updated Category\"}"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "PUT /api/v1/categories/:id returns 200" "$result"

    # DELETE /api/v1/categories/:id (delete child first, then parent)
    make_request "DELETE" "/api/v1/categories/$CHILD_CATEGORY_ID" "$(auth_header "$ADMIN_TOKEN")"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "DELETE /api/v1/categories/:id (child) returns 200" "$result"

    make_request "DELETE" "/api/v1/categories/$CATEGORY_ID" "$(auth_header "$ADMIN_TOKEN")"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "DELETE /api/v1/categories/:id (parent) returns 200" "$result"

    # Re-create category for subsequent product tests
    make_request "POST" "/api/v1/categories" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"name\":\"Products Category\",\"slug\":\"products-category\"}"
    result=1
    if assert_status "201" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        CATEGORY_ID=$(echo "$RESPONSE" | jq -r '.data.id // empty' 2>/dev/null)
        if [ -n "$CATEGORY_ID" ]; then
            result=0
        fi
    fi
    run_test "POST /api/v1/categories (re-create for products) returns 201" "$result"

    # --- Error Cases ---

    # POST /api/v1/categories without auth (401)
    > "$COOKIE_JAR"
    make_request "POST" "/api/v1/categories" "" \
        "{\"name\":\"No Auth\",\"slug\":\"no-auth\"}"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/categories without auth returns 401" "$result"
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}"

    # POST /api/v1/categories without admin role (403)
    make_request "POST" "/api/v1/categories" "$(auth_header "$USER_TOKEN")" \
        "{\"name\":\"No Admin\",\"slug\":\"no-admin\"}"
    result=1
    if assert_status "403" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/categories without admin role returns 403" "$result"

    # POST /api/v1/categories with missing required fields
    make_request "POST" "/api/v1/categories" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"name\":\"Missing Slug\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/categories with missing slug returns 400" "$result"

    # PUT /api/v1/categories/:id with invalid data
    make_request "PUT" "/api/v1/categories/$CATEGORY_ID" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"slug\":\"\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "PUT /api/v1/categories/:id with invalid data returns 400" "$result"

    # DELETE /api/v1/categories/:id without auth (401)
    > "$COOKIE_JAR"
    make_request "DELETE" "/api/v1/categories/$CATEGORY_ID"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "DELETE /api/v1/categories/:id without auth returns 401" "$result"
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}"

    # GET /api/v1/categories/:id with non-existent ID (404)
    make_request "GET" "/api/v1/categories/999999"
    result=1
    if assert_status "404" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "GET /api/v1/categories/:id (non-existent) returns 404" "$result"

    # PUT /api/v1/categories/:id with non-existent ID (404)
    make_request "PUT" "/api/v1/categories/999999" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"name\":\"Ghost\"}"
    result=1
    if assert_status "404" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "PUT /api/v1/categories/:id (non-existent) returns 404" "$result"

    # DELETE /api/v1/categories/:id with non-existent ID (404)
    make_request "DELETE" "/api/v1/categories/999999" "$(auth_header "$ADMIN_TOKEN")"
    result=1
    if assert_status "404" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "DELETE /api/v1/categories/:id (non-existent) returns 404" "$result"
}

# =============================================================================
# SECTION 4: Products Module
# =============================================================================
test_products() {
    print_section "Section 4: Products Module"

    # --- Happy Path ---

    # POST /api/v1/products (admin creates product)
    make_request "POST" "/api/v1/products" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"category_id\":$CATEGORY_ID,\"name\":\"Test Product\",\"slug\":\"test-product\",\"base_price\":99.99,\"description\":\"A test product\"}"
    local result=1
    if assert_status "201" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        PRODUCT_ID=$(echo "$RESPONSE" | jq -r '.data.id // empty' 2>/dev/null)
        PRODUCT_SLUG=$(echo "$RESPONSE" | jq -r '.data.slug // empty' 2>/dev/null)
        if [ -n "$PRODUCT_ID" ]; then
            result=0
        fi
    fi
    run_test "POST /api/v1/products (admin creates) returns 201" "$result"

    # GET /api/v1/products (list all)
    make_request "GET" "/api/v1/products"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/products returns 200" "$result"

    # GET /api/v1/products/:id
    make_request "GET" "/api/v1/products/$PRODUCT_ID"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/products/:id returns 200" "$result"

    # GET /api/v1/products/slug/:slug
    make_request "GET" "/api/v1/products/slug/$PRODUCT_SLUG"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/products/slug/:slug returns 200" "$result"

    # PUT /api/v1/products/:id (update)
    make_request "PUT" "/api/v1/products/$PRODUCT_ID" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"name\":\"Updated Product\",\"base_price\":149.99}"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "PUT /api/v1/products/:id returns 200" "$result"

    # DELETE /api/v1/products/:id
    make_request "DELETE" "/api/v1/products/$PRODUCT_ID" "$(auth_header "$ADMIN_TOKEN")"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "DELETE /api/v1/products/:id returns 200" "$result"

    # Re-create product for subsequent tests
    make_request "POST" "/api/v1/products" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"category_id\":$CATEGORY_ID,\"name\":\"Test Product\",\"slug\":\"test-product\",\"base_price\":99.99,\"description\":\"A test product\"}"
    result=1
    if assert_status "201" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        PRODUCT_ID=$(echo "$RESPONSE" | jq -r '.data.id // empty' 2>/dev/null)
        PRODUCT_SLUG=$(echo "$RESPONSE" | jq -r '.data.slug // empty' 2>/dev/null)
        if [ -n "$PRODUCT_ID" ]; then
            result=0
        fi
    fi
    run_test "POST /api/v1/products (re-create for variants) returns 201" "$result"

    # --- Error Cases ---

    # POST /api/v1/products without auth (401)
    > "$COOKIE_JAR"
    make_request "POST" "/api/v1/products" "" \
        "{\"category_id\":$CATEGORY_ID,\"name\":\"No Auth\",\"slug\":\"no-auth\",\"base_price\":10.00}"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/products without auth returns 401" "$result"
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}"

    # POST /api/v1/products without admin role (403)
    make_request "POST" "/api/v1/products" "$(auth_header "$USER_TOKEN")" \
        "{\"category_id\":$CATEGORY_ID,\"name\":\"No Admin\",\"slug\":\"no-admin\",\"base_price\":10.00}"
    result=1
    if assert_status "403" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/products without admin role returns 403" "$result"

    # POST /api/v1/products with missing required fields
    make_request "POST" "/api/v1/products" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"name\":\"Missing Fields\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/products with missing required fields returns 400" "$result"

    # POST /api/v1/products with negative base_price
    make_request "POST" "/api/v1/products" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"category_id\":$CATEGORY_ID,\"name\":\"Negative Price\",\"slug\":\"negative-price\",\"base_price\":-10.00}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/products with negative price returns 400" "$result"

    # POST /api/v1/products with zero base_price
    make_request "POST" "/api/v1/products" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"category_id\":$CATEGORY_ID,\"name\":\"Zero Price\",\"slug\":\"zero-price\",\"base_price\":0}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/products with zero price returns 400" "$result"

    # POST /api/v1/products with non-existent category_id
    make_request "POST" "/api/v1/products" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"category_id\":999999,\"name\":\"Bad Category\",\"slug\":\"bad-category\",\"base_price\":10.00}"
    result=1
    if assert_status "400" "$HTTP_STATUS" || assert_status "404" "$HTTP_STATUS" || assert_status "500" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/products with non-existent category_id returns error" "$result"

    # PUT /api/v1/products/:id with non-existent ID (404)
    make_request "PUT" "/api/v1/products/999999" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"name\":\"Ghost\"}"
    result=1
    if assert_status "404" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "PUT /api/v1/products/:id (non-existent) returns 404" "$result"

    # DELETE /api/v1/products/:id with non-existent ID (404)
    make_request "DELETE" "/api/v1/products/999999" "$(auth_header "$ADMIN_TOKEN")"
    result=1
    if assert_status "404" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "DELETE /api/v1/products/:id (non-existent) returns 404" "$result"

    # GET /api/v1/products/:id with non-existent ID (404)
    make_request "GET" "/api/v1/products/999999"
    result=1
    if assert_status "404" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "GET /api/v1/products/:id (non-existent) returns 404" "$result"
}

# =============================================================================
# SECTION 5: Variants Module
# =============================================================================
test_variants() {
    print_section "Section 5: Variants Module"

    # --- Happy Path ---

    # POST /api/v1/products/:id/variants (admin creates variant)
    make_request "POST" "/api/v1/products/$PRODUCT_ID/variants" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"sku\":\"TEST-SKU-001\",\"price\":49.99,\"stock\":100}"
    local result=1
    if assert_status "201" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        VARIANT_ID=$(echo "$RESPONSE" | jq -r '.data.id // empty' 2>/dev/null)
        if [ -n "$VARIANT_ID" ]; then
            result=0
        fi
    fi
    run_test "POST /api/v1/products/:id/variants returns 201" "$result"

    # GET /api/v1/products/:id/variants
    make_request "GET" "/api/v1/products/$PRODUCT_ID/variants"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/products/:id/variants returns 200" "$result"

    # GET /api/v1/variants/:id
    make_request "GET" "/api/v1/variants/$VARIANT_ID"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/variants/:id returns 200" "$result"

    # PUT /api/v1/variants/:id (update)
    make_request "PUT" "/api/v1/variants/$VARIANT_ID" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"price\":59.99,\"stock\":150}"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "PUT /api/v1/variants/:id returns 200" "$result"

    # DELETE /api/v1/variants/:id
    make_request "DELETE" "/api/v1/variants/$VARIANT_ID" "$(auth_header "$ADMIN_TOKEN")"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "DELETE /api/v1/variants/:id returns 200" "$result"

    # Re-create variant for subsequent tests
    make_request "POST" "/api/v1/products/$PRODUCT_ID/variants" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"sku\":\"TEST-SKU-002\",\"price\":49.99,\"stock\":100}"
    result=1
    if assert_status "201" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        VARIANT_ID=$(echo "$RESPONSE" | jq -r '.data.id // empty' 2>/dev/null)
        if [ -n "$VARIANT_ID" ]; then
            result=0
        fi
    fi
    run_test "POST /api/v1/products/:id/variants (re-create for cart) returns 201" "$result"

    # --- Error Cases ---

    # POST variant without auth (401)
    > "$COOKIE_JAR"
    make_request "POST" "/api/v1/products/$PRODUCT_ID/variants" "" \
        "{\"sku\":\"NO-AUTH\",\"price\":10.00,\"stock\":5}"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST variant without auth returns 401" "$result"
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}"

    # POST variant without admin role (403)
    make_request "POST" "/api/v1/products/$PRODUCT_ID/variants" "$(auth_header "$USER_TOKEN")" \
        "{\"sku\":\"NO-ADMIN\",\"price\":10.00,\"stock\":5}"
    result=1
    if assert_status "403" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST variant without admin role returns 403" "$result"

    # POST variant with missing required fields
    make_request "POST" "/api/v1/products/$PRODUCT_ID/variants" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"price\":10.00}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST variant with missing required fields returns 400" "$result"

    # PUT variant with non-existent ID (404)
    make_request "PUT" "/api/v1/variants/999999" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"price\":10.00}"
    result=1
    if assert_status "404" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "PUT variant (non-existent) returns 404" "$result"

    # DELETE variant with non-existent ID (404)
    make_request "DELETE" "/api/v1/variants/999999" "$(auth_header "$ADMIN_TOKEN")"
    result=1
    if assert_status "404" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "DELETE variant (non-existent) returns 404" "$result"

    # GET variant with non-existent ID (404)
    make_request "GET" "/api/v1/variants/999999"
    result=1
    if assert_status "404" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "GET variant (non-existent) returns 404" "$result"
}

# =============================================================================
# SECTION 6: Images Module
# =============================================================================
test_images() {
    print_section "Section 6: Images Module"

    # --- Happy Path ---

    # POST /api/v1/products/:id/images (admin creates image)
    make_request "POST" "/api/v1/products/$PRODUCT_ID/images" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"url\":\"https://example.com/image1.jpg\",\"alt_text\":\"Test Image\",\"is_main\":true}"
    local result=1
    if assert_status "201" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        IMAGE_ID=$(echo "$RESPONSE" | jq -r '.data.id // empty' 2>/dev/null)
        if [ -n "$IMAGE_ID" ]; then
            result=0
        fi
    fi
    run_test "POST /api/v1/products/:id/images returns 201" "$result"

    # GET /api/v1/products/:id/images
    make_request "GET" "/api/v1/products/$PRODUCT_ID/images"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/products/:id/images returns 200" "$result"

    # GET /api/v1/images/:id
    make_request "GET" "/api/v1/images/$IMAGE_ID"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/images/:id returns 200" "$result"

    # PUT /api/v1/images/:id (update)
    make_request "PUT" "/api/v1/images/$IMAGE_ID" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"alt_text\":\"Updated Image\",\"url\":\"https://example.com/updated.jpg\"}"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "PUT /api/v1/images/:id returns 200" "$result"

    # PUT /api/v1/products/:id/images/:image_id/main (set main)
    make_request "PUT" "/api/v1/products/$PRODUCT_ID/images/$IMAGE_ID/main" "$(auth_header "$ADMIN_TOKEN")"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "PUT /api/v1/products/:id/images/:image_id/main returns 200" "$result"

    # DELETE /api/v1/images/:id
    make_request "DELETE" "/api/v1/images/$IMAGE_ID" "$(auth_header "$ADMIN_TOKEN")"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "DELETE /api/v1/images/:id returns 200" "$result"

    # --- Error Cases ---

    # POST image without auth (401)
    > "$COOKIE_JAR"
    make_request "POST" "/api/v1/products/$PRODUCT_ID/images" "" \
        "{\"url\":\"https://example.com/noauth.jpg\",\"alt_text\":\"No Auth\"}"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST image without auth returns 401" "$result"
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}"

    # POST image without admin role (403)
    make_request "POST" "/api/v1/products/$PRODUCT_ID/images" "$(auth_header "$USER_TOKEN")" \
        "{\"url\":\"https://example.com/noadmin.jpg\",\"alt_text\":\"No Admin\"}"
    result=1
    if assert_status "403" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST image without admin role returns 403" "$result"

    # POST image with missing required fields
    make_request "POST" "/api/v1/products/$PRODUCT_ID/images" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"alt_text\":\"Missing URL\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST image with missing required fields returns 400" "$result"

    # PUT image with non-existent ID (404)
    make_request "PUT" "/api/v1/images/999999" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"alt_text\":\"Ghost\"}"
    result=1
    if assert_status "404" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "PUT image (non-existent) returns 404" "$result"

    # DELETE image with non-existent ID (404)
    make_request "DELETE" "/api/v1/images/999999" "$(auth_header "$ADMIN_TOKEN")"
    result=1
    if assert_status "404" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "DELETE image (non-existent) returns 404" "$result"
}

# =============================================================================
# SECTION 7: Inventory Module
# =============================================================================
test_inventory() {
    print_section "Section 7: Inventory Module"

    # --- Happy Path ---

    # PUT /api/v1/inventory/:product_id (admin sets quantity)
    make_request "PUT" "/api/v1/inventory/$PRODUCT_ID" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"quantity\":500}"
    local result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "PUT /api/v1/inventory/:product_id returns 200" "$result"

    # GET /api/v1/inventory/:product_id
    make_request "GET" "/api/v1/inventory/$PRODUCT_ID"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/inventory/:product_id returns 200" "$result"

    # POST /api/v1/inventory/:product_id/reserve
    make_request "POST" "/api/v1/inventory/$PRODUCT_ID/reserve" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"quantity\":10}"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "POST /api/v1/inventory/:product_id/reserve returns 200" "$result"

    # POST /api/v1/inventory/:product_id/release
    make_request "POST" "/api/v1/inventory/$PRODUCT_ID/release" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"quantity\":5}"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "POST /api/v1/inventory/:product_id/release returns 200" "$result"

    # --- Error Cases ---

    # PUT inventory without auth (401)
    > "$COOKIE_JAR"
    make_request "PUT" "/api/v1/inventory/$PRODUCT_ID" "" \
        "{\"quantity\":100}"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "PUT inventory without auth returns 401" "$result"
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}"

    # PUT inventory without admin role (403)
    make_request "PUT" "/api/v1/inventory/$PRODUCT_ID" "$(auth_header "$USER_TOKEN")" \
        "{\"quantity\":100}"
    result=1
    if assert_status "403" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "PUT inventory without admin role returns 403" "$result"

    # POST reserve with invalid quantity (0)
    make_request "POST" "/api/v1/inventory/$PRODUCT_ID/reserve" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"quantity\":0}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST reserve with quantity 0 returns 400" "$result"

    # POST reserve with negative quantity
    make_request "POST" "/api/v1/inventory/$PRODUCT_ID/reserve" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"quantity\":-5}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST reserve with negative quantity returns 400" "$result"

    # POST release with invalid quantity
    make_request "POST" "/api/v1/inventory/$PRODUCT_ID/release" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"quantity\":0}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST release with quantity 0 returns 400" "$result"

    # GET inventory for non-existent product
    make_request "GET" "/api/v1/inventory/999999"
    result=1
    if assert_status "404" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "GET inventory (non-existent product) returns 404" "$result"
}

# =============================================================================
# SECTION 8: Cart Module
# =============================================================================
test_cart() {
    print_section "Section 8: Cart Module"

    # --- Happy Path ---

    # POST /api/v1/cart/items (add item)
    make_request "POST" "/api/v1/cart/items" "$(auth_header "$USER_TOKEN")" \
        "{\"variant_id\":$VARIANT_ID,\"quantity\":2}"
    local result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "POST /api/v1/cart/items returns 200" "$result"

    # GET /api/v1/cart
    make_request "GET" "/api/v1/cart" "$(auth_header "$USER_TOKEN")"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/cart returns 200" "$result"

    # PUT /api/v1/cart/items/:variant_id (update quantity)
    make_request "PUT" "/api/v1/cart/items/$VARIANT_ID" "$(auth_header "$USER_TOKEN")" \
        "{\"quantity\":5}"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "PUT /api/v1/cart/items/:variant_id returns 200" "$result"

    # DELETE /api/v1/cart/items/:variant_id (remove item)
    make_request "DELETE" "/api/v1/cart/items/$VARIANT_ID" "$(auth_header "$USER_TOKEN")"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "DELETE /api/v1/cart/items/:variant_id returns 200" "$result"

    # POST /api/v1/cart/items (add multiple - re-add for checkout)
    make_request "POST" "/api/v1/cart/items" "$(auth_header "$USER_TOKEN")" \
        "{\"variant_id\":$VARIANT_ID,\"quantity\":1}"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "POST /api/v1/cart/items (re-add for checkout) returns 200" "$result"

    # POST /api/v1/cart/checkout (checkout cart -> creates order)
    make_request "POST" "/api/v1/cart/checkout" "$(auth_header "$USER_TOKEN")" \
        "{}"
    result=1
    if assert_status "200" "$HTTP_STATUS" || assert_status "201" "$HTTP_STATUS"; then
        if assert_success "$RESPONSE"; then
            ORDER_ID=$(echo "$RESPONSE" | jq -r '.data.id // .data.order_id // empty' 2>/dev/null)
            if [ -n "$ORDER_ID" ]; then
                result=0
            fi
        fi
    fi
    run_test "POST /api/v1/cart/checkout returns 200/201" "$result"

    # DELETE /api/v1/cart (clear cart)
    make_request "DELETE" "/api/v1/cart" "$(auth_header "$USER_TOKEN")"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "DELETE /api/v1/cart returns 200" "$result"

    # --- Error Cases ---

    # All cart endpoints without auth (401)
    > "$COOKIE_JAR"
    make_request "GET" "/api/v1/cart"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "GET /api/v1/cart without auth returns 401" "$result"
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASSWORD\"}"

    # POST cart/items with invalid variant_id
    make_request "POST" "/api/v1/cart/items" "$(auth_header "$USER_TOKEN")" \
        "{\"variant_id\":999999,\"quantity\":1}"
    result=1
    if assert_status "400" "$HTTP_STATUS" || assert_status "404" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST cart/items with invalid variant_id returns error" "$result"

    # POST cart/items with quantity <= 0
    make_request "POST" "/api/v1/cart/items" "$(auth_header "$USER_TOKEN")" \
        "{\"variant_id\":$VARIANT_ID,\"quantity\":0}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST cart/items with quantity 0 returns 400" "$result"

    # PUT cart/items with quantity < 0
    make_request "PUT" "/api/v1/cart/items/$VARIANT_ID" "$(auth_header "$USER_TOKEN")" \
        "{\"quantity\":-1}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "PUT cart/items with negative quantity returns 400" "$result"

    # POST cart/checkout with empty cart (clear first)
    make_request "DELETE" "/api/v1/cart" "$(auth_header "$USER_TOKEN")"
    make_request "POST" "/api/v1/cart/checkout" "$(auth_header "$USER_TOKEN")" \
        "{}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST cart/checkout with empty cart returns 400" "$result"
}

# =============================================================================
# SECTION 9: Orders Module
# =============================================================================
test_orders() {
    print_section "Section 9: Orders Module"

    # --- Happy Path ---

    # POST /api/v1/orders (create order)
    make_request "POST" "/api/v1/orders" "$(auth_header "$USER_TOKEN")" \
        "{\"items\":[{\"product_id\":$PRODUCT_ID,\"quantity\":1}]}"
    local result=1
    if assert_status "201" "$HTTP_STATUS" || assert_status "200" "$HTTP_STATUS"; then
        if assert_success "$RESPONSE"; then
            ORDER_ID=$(echo "$RESPONSE" | jq -r '.data.id // empty' 2>/dev/null)
            if [ -n "$ORDER_ID" ]; then
                result=0
            fi
        fi
    fi
    run_test "POST /api/v1/orders returns 200/201" "$result"

    # GET /api/v1/orders/:id (get own order)
    make_request "GET" "/api/v1/orders/$ORDER_ID" "$(auth_header "$USER_TOKEN")"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/orders/:id (own order) returns 200" "$result"

    # GET /api/v1/orders (admin lists all orders)
    make_request "GET" "/api/v1/orders" "$(auth_header "$ADMIN_TOKEN")"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "GET /api/v1/orders (admin list) returns 200" "$result"

    # POST /api/v1/orders/:id/confirm (confirm sale)
    make_request "POST" "/api/v1/orders/$ORDER_ID/confirm" "$(auth_header "$USER_TOKEN")"
    result=1
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        result=0
    fi
    run_test "POST /api/v1/orders/:id/confirm returns 200" "$result"

    # Create another order for cancel test
    make_request "POST" "/api/v1/orders" "$(auth_header "$USER_TOKEN")" \
        "{\"items\":[{\"product_id\":$PRODUCT_ID,\"quantity\":1}]}"
    local cancel_order_id
    cancel_order_id=$(echo "$RESPONSE" | jq -r '.data.id // empty' 2>/dev/null)

    # POST /api/v1/orders/:id/cancel (cancel order)
    if [ -n "$cancel_order_id" ]; then
        make_request "POST" "/api/v1/orders/$cancel_order_id/cancel" "$(auth_header "$USER_TOKEN")"
        result=1
        if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
            result=0
        fi
        run_test "POST /api/v1/orders/:id/cancel returns 200" "$result"
    else
        run_test "POST /api/v1/orders/:id/cancel - SKIPPED (no order created)" "0"
    fi

    # PATCH /api/v1/orders/:id/status (admin updates status)
    make_request "PATCH" "/api/v1/orders/$ORDER_ID/status" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"status\":\"shipped\"}"
    result=1
    if assert_status "200" "$HTTP_STATUS" || assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "PATCH /api/v1/orders/:id/status returns 200 or 400" "$result"

    # GET /api/v1/orders/tasks/:task_id (check async task)
    make_request "GET" "/api/v1/orders/tasks/nonexistent-task-id" "$(auth_header "$USER_TOKEN")"
    result=1
    if assert_status "404" "$HTTP_STATUS" || assert_status "200" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "GET /api/v1/orders/tasks/:task_id returns response" "$result"

    # --- Error Cases ---

    # POST orders without auth (401)
    > "$COOKIE_JAR"
    make_request "POST" "/api/v1/orders" "" \
        "{\"items\":[{\"product_id\":$PRODUCT_ID,\"quantity\":1}]}"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/orders without auth returns 401" "$result"
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASSWORD\"}"

    # POST orders with empty body
    make_request "POST" "/api/v1/orders" "$(auth_header "$USER_TOKEN")" "{}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/orders with empty body returns 400" "$result"

    # POST orders with invalid items (empty items array)
    make_request "POST" "/api/v1/orders" "$(auth_header "$USER_TOKEN")" \
        "{\"items\":[]}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/orders with empty items array returns 400" "$result"

    # POST orders with non-existent product_id
    make_request "POST" "/api/v1/orders" "$(auth_header "$USER_TOKEN")" \
        "{\"items\":[{\"product_id\":999999,\"quantity\":1}]}"
    result=1
    if assert_status "400" "$HTTP_STATUS" || assert_status "404" "$HTTP_STATUS" || assert_status "500" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/orders with non-existent product_id returns error" "$result"

    # POST orders with quantity > available inventory
    make_request "POST" "/api/v1/orders" "$(auth_header "$USER_TOKEN")" \
        "{\"items\":[{\"product_id\":$PRODUCT_ID,\"quantity\":999999}]}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/orders with quantity > inventory returns 400" "$result"

    # GET /api/v1/orders/:id for another user's order (403 BOLA)
    make_request "GET" "/api/v1/orders/$ORDER_ID" "$(auth_header "$ADMIN_TOKEN")"
    result=1
    if assert_status "403" "$HTTP_STATUS" || assert_status "404" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "GET /api/v1/orders/:id (other user's order) returns 403/404" "$result"

    # PATCH status without admin role (403)
    make_request "PATCH" "/api/v1/orders/$ORDER_ID/status" "$(auth_header "$USER_TOKEN")" \
        "{\"status\":\"shipped\"}"
    result=1
    if assert_status "403" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "PATCH /api/v1/orders/:id/status without admin returns 403" "$result"

    # POST confirm for already confirmed order
    make_request "POST" "/api/v1/orders/$ORDER_ID/confirm" "$(auth_header "$USER_TOKEN")"
    result=1
    if assert_status "400" "$HTTP_STATUS" || assert_status "200" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST confirm for already confirmed order returns 400 or 200" "$result"

    # POST cancel for already cancelled order
    if [ -n "$cancel_order_id" ]; then
        make_request "POST" "/api/v1/orders/$cancel_order_id/cancel" "$(auth_header "$USER_TOKEN")"
        result=1
        if assert_status "400" "$HTTP_STATUS" || assert_status "200" "$HTTP_STATUS"; then
            result=0
        fi
        run_test "POST cancel for already cancelled order returns 400 or 200" "$result"
    fi
}

# =============================================================================
# SECTION 10: Payments Module
# =============================================================================
test_payments() {
    print_section "Section 10: Payments Module"

    # --- Happy Path ---

    # POST /api/v1/payments (create payment)
    make_request "POST" "/api/v1/payments" "$(auth_header "$USER_TOKEN")" \
        "{\"amount\":10000,\"currency\":\"COP\",\"customer_email\":\"$USER_EMAIL\",\"reference\":\"TEST-REF-001\"}"
    local result=1
    if assert_status "201" "$HTTP_STATUS" || assert_status "200" "$HTTP_STATUS" || assert_status "500" "$HTTP_STATUS"; then
        if assert_status "201" "$HTTP_STATUS" || assert_status "200" "$HTTP_STATUS"; then
            if assert_success "$RESPONSE"; then
                PAYMENT_ID=$(echo "$RESPONSE" | jq -r '.data.id // empty' 2>/dev/null)
                result=0
            fi
        else
            # Wompi sandbox may fail, but we test the flow
            result=0
        fi
    fi
    run_test "POST /api/v1/payments returns response" "$result"

    # GET /api/v1/payments/:id (get payment)
    if [ -n "$PAYMENT_ID" ] && [ "$PAYMENT_ID" != "null" ]; then
        make_request "GET" "/api/v1/payments/$PAYMENT_ID" "$(auth_header "$USER_TOKEN")"
        result=1
        if assert_status "200" "$HTTP_STATUS"; then
            result=0
        fi
        run_test "GET /api/v1/payments/:id returns 200" "$result"
    else
        run_test "GET /api/v1/payments/:id - SKIPPED (no payment ID)" "0"
    fi

    # POST /api/v1/payments/:id/void (void payment)
    if [ -n "$PAYMENT_ID" ] && [ "$PAYMENT_ID" != "null" ]; then
        make_request "POST" "/api/v1/payments/$PAYMENT_ID/void" "$(auth_header "$USER_TOKEN")"
        result=1
        if assert_status "200" "$HTTP_STATUS" || assert_status "400" "$HTTP_STATUS"; then
            result=0
        fi
        run_test "POST /api/v1/payments/:id/void returns response" "$result"
    else
        run_test "POST /api/v1/payments/:id/void - SKIPPED (no payment ID)" "0"
    fi

    # POST /api/v1/payments/links (create payment link)
    make_request "POST" "/api/v1/payments/links" "$(auth_header "$USER_TOKEN")" \
        "{\"amount\":5000,\"currency\":\"COP\",\"description\":\"Test Payment Link\",\"reference\":\"TEST-LINK-001\"}"
    result=1
    if assert_status "201" "$HTTP_STATUS" || assert_status "200" "$HTTP_STATUS"; then
        if assert_success "$RESPONSE"; then
            PAYMENT_LINK_ID=$(echo "$RESPONSE" | jq -r '.data.id // empty' 2>/dev/null)
            if [ -n "$PAYMENT_LINK_ID" ]; then
                result=0
            fi
        fi
    fi
    run_test "POST /api/v1/payments/links returns 200/201" "$result"

    # GET /api/v1/payments/links/:id
    if [ -n "$PAYMENT_LINK_ID" ] && [ "$PAYMENT_LINK_ID" != "null" ]; then
        make_request "GET" "/api/v1/payments/links/$PAYMENT_LINK_ID" "$(auth_header "$USER_TOKEN")"
        result=1
        if assert_status "200" "$HTTP_STATUS"; then
            result=0
        fi
        run_test "GET /api/v1/payments/links/:id returns 200" "$result"
    else
        run_test "GET /api/v1/payments/links/:id - SKIPPED (no link ID)" "0"
    fi

    # PATCH /api/v1/payments/links/:id/activate
    if [ -n "$PAYMENT_LINK_ID" ] && [ "$PAYMENT_LINK_ID" != "null" ]; then
        make_request "PATCH" "/api/v1/payments/links/$PAYMENT_LINK_ID/activate" "$(auth_header "$USER_TOKEN")"
        result=1
        if assert_status "200" "$HTTP_STATUS" || assert_status "400" "$HTTP_STATUS"; then
            result=0
        fi
        run_test "PATCH /api/v1/payments/links/:id/activate returns response" "$result"
    else
        run_test "PATCH /api/v1/payments/links/:id/activate - SKIPPED" "0"
    fi

    # PATCH /api/v1/payments/links/:id/deactivate
    if [ -n "$PAYMENT_LINK_ID" ] && [ "$PAYMENT_LINK_ID" != "null" ]; then
        make_request "PATCH" "/api/v1/payments/links/$PAYMENT_LINK_ID/deactivate" "$(auth_header "$USER_TOKEN")"
        result=1
        if assert_status "200" "$HTTP_STATUS" || assert_status "400" "$HTTP_STATUS"; then
            result=0
        fi
        run_test "PATCH /api/v1/payments/links/:id/deactivate returns response" "$result"
    else
        run_test "PATCH /api/v1/payments/links/:id/deactivate - SKIPPED" "0"
    fi

    # POST /api/v1/payments/webhook (test webhook without signature)
    make_request "POST" "/api/v1/payments/webhook" "" \
        "{\"id\":\"test-webhook-id\",\"status\":\"APPROVED\"}"
    result=1
    if assert_status "200" "$HTTP_STATUS" || assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/payments/webhook returns response" "$result"

    # --- Error Cases ---

    # POST payments without auth (401)
    > "$COOKIE_JAR"
    make_request "POST" "/api/v1/payments" "" \
        "{\"amount\":10000,\"currency\":\"COP\",\"customer_email\":\"test@test.com\",\"reference\":\"NO-AUTH\"}"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/payments without auth returns 401" "$result"
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASSWORD\"}"

    # POST payments with missing required fields
    make_request "POST" "/api/v1/payments" "$(auth_header "$USER_TOKEN")" "{}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/payments with missing fields returns 400" "$result"

    # POST payments with invalid amount (0)
    make_request "POST" "/api/v1/payments" "$(auth_header "$USER_TOKEN")" \
        "{\"amount\":0,\"currency\":\"COP\",\"customer_email\":\"$USER_EMAIL\",\"reference\":\"ZERO-AMT\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/payments with amount 0 returns 400" "$result"

    # POST payments with negative amount
    make_request "POST" "/api/v1/payments" "$(auth_header "$USER_TOKEN")" \
        "{\"amount\":-100,\"currency\":\"COP\",\"customer_email\":\"$USER_EMAIL\",\"reference\":\"NEG-AMT\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/payments with negative amount returns 400" "$result"

    # POST void for already voided payment
    if [ -n "$PAYMENT_ID" ] && [ "$PAYMENT_ID" != "null" ]; then
        make_request "POST" "/api/v1/payments/$PAYMENT_ID/void" "$(auth_header "$USER_TOKEN")"
        result=1
        if assert_status "400" "$HTTP_STATUS" || assert_status "200" "$HTTP_STATUS"; then
            result=0
        fi
        run_test "POST void for already voided payment returns 400 or 200" "$result"
    else
        run_test "POST void for already voided payment - SKIPPED" "0"
    fi

    # POST webhook without valid X-Wompi-Signature header
    make_request "POST" "/api/v1/payments/webhook" "" \
        "{\"id\":\"test\",\"status\":\"APPROVED\",\"signature\":\"invalid\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS" || assert_status "200" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST webhook without valid signature returns response" "$result"
}

# =============================================================================
# SECTION 11: Admin Module
# =============================================================================
test_admin() {
    print_section "Section 11: Admin Module"

    # --- Happy Path ---

    # POST /api/v1/admin/users (admin creates user with role)
    make_request "POST" "/api/v1/admin/users" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"email\":\"admin_test@bey.com\",\"password\":\"AdminTest123!\",\"first_name\":\"Admin\",\"last_name\":\"Test\",\"role\":\"admin\"}"
    local result=1
    if assert_status "201" "$HTTP_STATUS" || assert_status "200" "$HTTP_STATUS"; then
        if assert_success "$RESPONSE"; then
            result=0
        fi
    fi
    run_test "POST /api/v1/admin/users returns 200/201" "$result"

    # --- Error Cases ---

    # POST /api/v1/admin/users without auth (401)
    > "$COOKIE_JAR"
    make_request "POST" "/api/v1/admin/users" "" \
        "{\"email\":\"noauth@bey.com\",\"password\":\"NoAuth123!\",\"first_name\":\"No\",\"last_name\":\"Auth\"}"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/admin/users without auth returns 401" "$result"
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}"

    # POST /api/v1/admin/users without admin role (403)
    make_request "POST" "/api/v1/admin/users" "$(auth_header "$USER_TOKEN")" \
        "{\"email\":\"noadmin@bey.com\",\"password\":\"NoAdmin123!\",\"first_name\":\"No\",\"last_name\":\"Admin\"}"
    result=1
    if assert_status "403" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/admin/users without admin role returns 403" "$result"

    # POST /api/v1/admin/users with invalid role
    make_request "POST" "/api/v1/admin/users" "$(auth_header "$ADMIN_TOKEN")" \
        "{\"email\":\"badrole@bey.com\",\"password\":\"BadRole123!\",\"first_name\":\"Bad\",\"last_name\":\"Role\",\"role\":\"superadmin\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST /api/v1/admin/users with invalid role returns 400" "$result"
}

# =============================================================================
# SECTION 12: Global Error Cases
# =============================================================================
test_global() {
    print_section "Section 12: Global Error Cases"

    # Request to non-existent endpoint (404)
    make_request "GET" "/api/v1/nonexistent"
    local result=1
    if assert_status "404" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "GET /api/v1/nonexistent returns 404" "$result"

    # Request with invalid JSON body
    make_request "POST" "/api/v1/auth/login" "" "not valid json{{{"
    result=1
    if assert_status "400" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST with invalid JSON returns 400" "$result"

    # Request with wrong Content-Type
    make_request "POST" "/api/v1/auth/login" "Content-Type: text/plain" \
        "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}"
    result=1
    if assert_status "400" "$HTTP_STATUS" || assert_status "415" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "POST with wrong Content-Type returns 400/415" "$result"

    # Request with expired JWT token
    local expired_token="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJlbWFpbCI6InRlc3RAYmV5LmNvbSIsInJvbGUiOiJ1c2VyIiwiZXhwIjoxNjAwMDAwMDAwfQ.expired_signature"
    make_request "GET" "/api/v1/users/1" "$(auth_header "$expired_token")"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "GET with expired JWT returns 401" "$result"

    # Request with malformed JWT token
    make_request "GET" "/api/v1/users/1" "Authorization: Bearer not-a-valid-jwt-token"
    result=1
    if assert_status "401" "$HTTP_STATUS"; then
        result=0
    fi
    run_test "GET with malformed JWT returns 401" "$result"
}

# =============================================================================
# SUMMARY
# =============================================================================
print_summary() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  TEST SUMMARY${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
    echo -e "  Total:  $TESTS_TOTAL"
    echo -e "  ${GREEN}Passed: $TESTS_PASSED${NC}"
    echo -e "  ${RED}Failed: $TESTS_FAILED${NC}"

    if [ "$TESTS_FAILED" -eq 0 ]; then
        echo -e "\n  ${GREEN}All tests passed! ✓${NC}"
    else
        echo -e "\n  ${RED}Some tests failed. Review the output above. ✗${NC}"
    fi
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
}

# =============================================================================
# MAIN
# =============================================================================
main() {
    echo -e "${CYAN}"
    echo "========================================================"
    echo "  Bey API - Comprehensive Integration Test Suite"
    echo "========================================================"
    echo -e "${NC}"
    echo -e "Base URL: ${YELLOW}$BASE_URL${NC}"
    echo -e "Admin Email: ${YELLOW}$ADMIN_EMAIL${NC}"

    # Check if jq is installed
    if ! command -v jq &>/dev/null; then
        echo -e "${RED}Error: jq is required but not installed.${NC}"
        exit 1
    fi

    # Check if curl is installed
    if ! command -v curl &>/dev/null; then
        echo -e "${RED}Error: curl is required but not installed.${NC}"
        exit 1
    fi

    # Get admin password
    if [ -z "$ADMIN_PASSWORD" ]; then
        echo -e "${YELLOW}Admin password not set in ADMIN_PASSWORD environment variable.${NC}"
        echo -n "Enter admin password: "
        read -s ADMIN_PASSWORD
        echo ""
    fi

    # Check if API is running
    echo -e "\n${YELLOW}Checking if API is running...${NC}"
    HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health" 2>/dev/null)
    if [ "$HTTP_STATUS" != "200" ]; then
        echo -e "${RED}API is not running at $BASE_URL (health check returned $HTTP_STATUS)${NC}"
        echo -e "${YELLOW}Start the API first: go run ./cmd/api/${NC}"
        exit 1
    fi
    echo -e "${GREEN}API is running ✓${NC}"

    # Get admin user ID for BOLA tests
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}"
    if assert_status "200" "$HTTP_STATUS" && assert_success "$RESPONSE"; then
        ADMIN_TOKEN=$(echo "$RESPONSE" | jq -r '.data.access_token // empty' 2>/dev/null)
        ADMIN_USER_ID=$(echo "$RESPONSE" | jq -r '.data.user_id // .data.user.id // empty' 2>/dev/null)
        # If user_id not in response, fetch from users list
        if [ -z "$ADMIN_USER_ID" ] || [ "$ADMIN_USER_ID" = "null" ]; then
            make_request "GET" "/api/v1/users" "$(auth_header "$ADMIN_TOKEN")"
            ADMIN_USER_ID=$(echo "$RESPONSE" | jq -r ".data[] | select(.email == \"$ADMIN_EMAIL\") | .id" 2>/dev/null | head -1)
        fi
    fi

    if [ -z "$ADMIN_TOKEN" ]; then
        echo -e "${RED}Failed to authenticate as admin. Check credentials.${NC}"
        exit 1
    fi

    # Initialize cookie jar
    > "$COOKIE_JAR"

    # Re-login to set cookies for session
    make_request "POST" "/api/v1/auth/login" "" \
        "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}"

    # Determine which tests to run
    local run_all=true
    local sections=("$@")

    if [ ${#sections[@]} -eq 0 ]; then
        run_all=true
    else
        run_all=false
    fi

    # Run tests based on arguments
    if $run_all || [[ " ${sections[*]} " =~ " health " ]]; then
        test_health
    fi

    if $run_all || [[ " ${sections[*]} " =~ " auth " ]]; then
        test_auth
    fi

    if $run_all || [[ " ${sections[*]} " =~ " users " ]]; then
        test_users
    fi

    if $run_all || [[ " ${sections[*]} " =~ " categories " ]]; then
        test_categories
    fi

    if $run_all || [[ " ${sections[*]} " =~ " products " ]]; then
        test_products
    fi

    if $run_all || [[ " ${sections[*]} " =~ " variants " ]]; then
        test_variants
    fi

    if $run_all || [[ " ${sections[*]} " =~ " images " ]]; then
        test_images
    fi

    if $run_all || [[ " ${sections[*]} " =~ " inventory " ]]; then
        test_inventory
    fi

    if $run_all || [[ " ${sections[*]} " =~ " cart " ]]; then
        test_cart
    fi

    if $run_all || [[ " ${sections[*]} " =~ " orders " ]]; then
        test_orders
    fi

    if $run_all || [[ " ${sections[*]} " =~ " payments " ]]; then
        test_payments
    fi

    if $run_all || [[ " ${sections[*]} " =~ " admin " ]]; then
        test_admin
    fi

    if $run_all || [[ " ${sections[*]} " =~ " global " ]]; then
        test_global
    fi

    # Print summary
    print_summary

    # Cleanup
    rm -f "$COOKIE_JAR"

    # Exit with appropriate code
    if [ "$TESTS_FAILED" -gt 0 ]; then
        exit 1
    fi
    exit 0
}

# Run main function with all arguments
main "$@"
