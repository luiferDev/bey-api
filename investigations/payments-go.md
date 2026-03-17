# Wompi Payments Integration - Go Implementation Guide

## Overview

This document provides technical documentation for implementing Wompi payments in the Bey API (Go/Gin project).

**Wompi** is a payment gateway for Colombia supporting:
- Credit cards
- Debit cards
- Nequi (digital wallet)
- PSE (bank transfers)
- Daviplata

---

## Table of Contents

1. [Authentication & Keys](#authentication--keys)
2. [Environments](#environments)
3. [API Endpoints](#api-endpoints)
4. [Payment Flow](#payment-flow)
5. [Go Implementation](#go-implementation)
6. [Webhook Handling](#webhook-handling)
7. [Error Codes](#error-codes)

---

## Authentication & Keys

Wompi uses two types of keys:

| Key Type | Prefix | Purpose |
|----------|--------|---------|
| Public Key | `pub_prod_` / `pub_test_` | Client-side (widget) |
| Private Key | `prv_prod_` / `prv_test_` | Server-side (API calls) |

Additional secrets:
- **Event Key**: `prod_events_*` - For webhook signature verification
- **Integrity Key**: `prod_integrity_*` - For checksum validation

### Getting Keys

1. Register at [comercios.wompi.co](https://comercios.wompi.co/)
2. Get your API keys from the merchant dashboard

---

## Environments

| Environment | Base URL |
|-------------|----------|
| **Sandbox** | `https://sandbox.wompi.co/v1` |
| **Production** | `https://production.wompi.co/v1` |

**Important**: Always use the correct key prefix for each environment:
- Sandbox: `pub_test_*`, `prv_test_*`
- Production: `pub_prod_*`, `prv_prod_*`

---

## API Endpoints

### 1. Transactions

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/transactions` | Create a transaction |
| GET | `/transactions/{transaction_id}` | Get transaction status |
| GET | `/transactions` | Search transactions |
| POST | `/transactions/{transaction_id}/void` | Cancel a transaction |

### 2. Tokens

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/tokens/cards` | Tokenize credit card |
| POST | `/tokens/nequi` | Tokenize Nequi account |
| GET | `/tokens/nequi/{token_id}` | Get Nequi token info |

### 3. Payment Sources

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/payment_sources` | Create payment source |
| GET | `/payment_sources/{payment_source_id}` | Get payment source |

### 4. Payment Links

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/payment_links` | Create payment link |
| GET | `/payment_links/{payment_link_id}` | Get payment link |
| PATCH | `/payment_links/{payment_link_id}` | Activate/deactivate link |

### 5. Merchants

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/merchants/{merchant_public_key}` | Get merchant info & acceptance token |

---

## Payment Flow

### Option A: Direct API (Recommended for this project)

```
1. Client requests payment
2. Server creates Transaction with payment source
3. Server returns redirect URL to client
4. Client completes payment on Wompi
5. Wompi redirects back to your site
6. Server verifies transaction status via webhook or API
```

### Option B: Widget/Checkout Web

```
1. Server generates acceptance token
2. Client initializes Wompi Widget
3. User enters payment info in widget
4. Widget handles redirect to Wompi
5. Wompi redirects back with transaction ID
```

---

## Go Implementation

### Configuration (config.yaml)

```yaml
wompi:
  enabled: true
  environment: sandbox  # or production
  public_key: "pub_test_xxxxxxxxxxxxx"
  private_key: "prv_test_xxxxxxxxxxxxx"
  event_key: "test_events_xxxxxxxxxxxxx"
  integrity_key: "test_integrity_xxxxxxxxxxxxx"
  base_url: "https://sandbox.wompi.co/v1"
```

### Config Struct (internal/config/config.go)

```go
type WompiConfig struct {
    Enabled       bool   `yaml:"enabled"`
    Environment   string `yaml:"environment"`
    PublicKey     string `yaml:"public_key"`
    PrivateKey    string `yaml:"private_key"`
    EventKey      string `yaml:"event_key"`
    IntegrityKey  string `yaml:"integrity_key"`
    BaseURL       string `yaml:"base_url"`
}
```

### Wompi Client (internal/modules/payments/client.go)

```go
package payments

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "bey/internal/config"
)

type WompiClient struct {
    config    *config.WompiConfig
    client    *http.Client
}

func NewWompiClient(cfg *config.WompiConfig) *WompiClient {
    return &WompiClient{
        config: cfg,
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

// CreateTransaction creates a payment transaction
func (c *WompiClient) CreateTransaction(req *CreateTransactionRequest) (*Transaction, error) {
    endpoint := fmt.Sprintf("%s/transactions", c.config.BaseURL)
    
    body, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.PrivateKey))
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("failed to execute request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
        return nil, fmt.Errorf("wompi API error: status %d", resp.StatusCode)
    }

    var result TransactionResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return &result.Data.Transaction, nil
}

// GetTransaction retrieves transaction status
func (c *WompiClient) GetTransaction(transactionID string) (*Transaction, error) {
    endpoint := fmt.Sprintf("%s/transactions/%s", c.config.BaseURL, transactionID)
    
    httpReq, err := http.NewRequest("GET", endpoint, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.PrivateKey))

    resp, err := c.client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("failed to execute request: %w", err)
    }
    defer resp.Body.Close()

    var result TransactionResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return &result.Data.Transaction, nil
}

// VoidTransaction cancels a transaction
func (c *WompiClient) VoidTransaction(transactionID string) (*Transaction, error) {
    endpoint := fmt.Sprintf("%s/transactions/%s/void", c.config.BaseURL, transactionID)
    
    httpReq, err := http.NewRequest("POST", endpoint, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.PrivateKey))

    resp, err := c.client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("failed to execute request: %w", err)
    }
    defer resp.Body.Close()

    var result TransactionResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return &result.Data.Transaction, nil
}

// CreateToken creates a card token (for recurring payments)
func (c *WompiClient) CreateToken(req *TokenRequest) (*Token, error) {
    endpoint := fmt.Sprintf("%s/tokens/cards", c.config.BaseURL)
    
    body, _ := json.Marshal(req)
    
    httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
    if err != nil {
        return nil, err
    }

    httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.PrivateKey))
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result TokenResponse
    json.NewDecoder(resp.Body).Decode(&result)

    return &result.Data.Token, nil
}
```

### Request/Response Types (internal/modules/payments/dto.go)

```go
package payments

// CreateTransactionRequest - Request to create a transaction
type CreateTransactionRequest struct {
    Amount        int64         `json:"amount"`         // Amount in cents
    Currency      string        `json:"currency"`      // "COP"
    CustomerEmail string        `json:"customer_email"`
    PaymentSource PaymentSource `json:"payment_source"`
    PaymentMethod PaymentMethod `json:"payment_method,omitempty"`
    RedirectURL   string        `json:"redirect_url"` // Return URL after payment
    Reference     string        `json:"reference"`    // Your order ID
}

// PaymentSource - Payment source details
type PaymentSource struct {
    Type string `json:"type"` // "CARD", "NEQUI", "PSE"
    Token string `json:"token,omitempty"` // Tokenized card/account
    DVV string `json:"dvv,omitempty"` // For cards
    CVV string `json:"cvv,omitempty"` // For cards
    PhoneNumber string `json:"phone_number,omitempty"` // For Nequi
    DocumentType string `json:"document_type,omitempty"` // "CC", "CE", "TI", "NIT"
    Document string `json:"document,omitempty"` // Document number
}

// PaymentMethod - Payment method type
type PaymentMethod struct {
    Type string `json:"type"` // "CARD", "NEQUI", "PSE", "BANCOLOMBIA_TRANSFER", "DAVIPLATA"
    Installments int `json:"installments,omitempty"` // 1-36
}

// Transaction - Transaction response
type Transaction struct {
    ID            string `json:"id"`
    Status        string `json:"status"` // "PENDING", "APPROVED", "DECLINED", "VOIDED", "ERROR"
    StatusMessage string `json:"status_message,omitempty"`
    Amount        int64  `json:"amount"`
    Currency      string `json:"currency"`
    Reference     string `json:"reference"`
    PaymentSource PaymentSource `json:"payment_source"`
    CreatedAt     string `json:"created_at"`
    UpdatedAt     string `json:"updated_at"`
    RedirectURL   string `json:"redirect_url,omitempty"`
}

// TransactionResponse - API response wrapper
type TransactionResponse struct {
    Data struct {
        Transaction Transaction `json:"transaction"`
    } `json:"data"`
}

// TokenRequest - Request to create a card token
type TokenRequest struct {
    CardNumber string `json:"card_number"`
    CVC        string `json:"cvc"`
    ExpMonth   string `json:"exp_month"`
    ExpYear    string `json:"exp_year"`
    CardHolder string `json:"card_holder"`
}

// Token - Token response
type Token struct {
    ID         string `json:"id"`
    Status     string `json:"status"`
    CreatedAt  string `json:"created_at"`
}

// TokenResponse - Token API response wrapper
type TokenResponse struct {
    Data struct {
        Token Token `json:"token"`
    } `json:"data"`
}
```

### Payment Service (internal/modules/payments/service.go)

```go
package payments

import (
    "errors"
    "fmt"
    "strconv"
    "time"

    "bey/internal/config"
)

var (
    ErrInvalidAmount       = errors.New("amount must be greater than 0")
    ErrInvalidCurrency     = errors.New("currency must be COP")
    ErrPaymentFailed       = errors.New("payment failed")
    ErrInsufficientFunds   = errors.New("insufficient funds")
    ErrCardDeclined        = errors.New("card declined")
    ErrTransactionNotFound = errors.New("transaction not found")
)

type PaymentService struct {
    client *WompiClient
    config *config.WompiConfig
}

func NewPaymentService(cfg *config.Config) *PaymentService {
    return &PaymentService{
        client: NewWompiClient(&cfg.Wompi),
        config: &cfg.Wompi,
    }
}

// CreatePayment creates a new payment transaction
func (s *PaymentService) CreatePayment(orderID string, amount float64, currency, customerEmail, paymentToken string, redirectURL string) (*Transaction, error) {
    // Validate amount (convert to cents)
    amountInCents := int64(amount * 100)
    if amountInCents <= 0 {
        return nil, ErrInvalidAmount
    }

    // Create transaction request
    req := &CreateTransactionRequest{
        Amount:        amountInCents,
        Currency:      currency,
        CustomerEmail: customerEmail,
        RedirectURL:   redirectURL,
        Reference:     orderID,
        PaymentSource: PaymentSource{
            Type:  "CARD",
            Token: paymentToken,
        },
    }

    // Create transaction
    transaction, err := s.client.CreateTransaction(req)
    if err != nil {
        return nil, fmt.Errorf("failed to create transaction: %w", err)
    }

    return transaction, nil
}

// GetPaymentStatus checks transaction status
func (s *PaymentService) GetPaymentStatus(transactionID string) (*Transaction, error) {
    transaction, err := s.client.GetTransaction(transactionID)
    if err != nil {
        return nil, ErrTransactionNotFound
    }
    return transaction, nil
}

// CancelPayment voids/cancels a transaction
func (s *PaymentService) CancelPayment(transactionID string) (*Transaction, error) {
    transaction, err := s.client.VoidTransaction(transactionID)
    if err != nil {
        return nil, fmt.Errorf("failed to void transaction: %w", err)
    }
    return transaction, nil
}

// IsPaymentSuccessful checks if transaction was approved
func (s *PaymentService) IsPaymentSuccessful(transaction *Transaction) bool {
    return transaction.Status == "APPROVED"
}

// ConvertAmountToCents converts decimal amount to cents
func ConvertAmountToCents(amount float64) int64 {
    return int64(amount * 100)
}

// ConvertCentsToAmount converts cents to decimal
func ConvertCentsToAmount(cents int64) float64 {
    return float64(cents) / 100
}
```

### Handler (internal/modules/payments/handler.go)

```go
package payments

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
)

type PaymentHandler struct {
    service *PaymentService
}

func NewPaymentHandler(service *PaymentService) *PaymentHandler {
    return &PaymentHandler{service: service}
}

// CreatePaymentRequest - HTTP request body
type CreatePaymentRequest struct {
    OrderID       string `json:"order_id" binding:"required"`
    Amount        string `json:"amount" binding:"required"`
    Currency      string `json:"currency" binding:"required"`
    PaymentToken  string `json:"payment_token" binding:"required"`
    CustomerEmail string `json:"customer_email" binding:"required,email"`
    RedirectURL   string `json:"redirect_url" binding:"required"`
}

// CreatePayment - POST /api/v1/payments
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
    var req CreatePaymentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Parse amount
    amount, err := strconv.ParseFloat(req.Amount, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
        return
    }

    // Create payment
    transaction, err := h.service.CreatePayment(
        req.OrderID,
        amount,
        req.Currency,
        req.CustomerEmail,
        req.PaymentToken,
        req.RedirectURL,
    )
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "transaction_id": transaction.ID,
        "status":        transaction.Status,
        "redirect_url":  transaction.RedirectURL,
    })
}

// GetPaymentStatus - GET /api/v1/payments/:transaction_id
func (h *PaymentHandler) GetPaymentStatus(c *gin.Context) {
    transactionID := c.Param("transaction_id")

    transaction, err := h.service.GetPaymentStatus(transactionID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "transaction_id": transaction.ID,
        "status":        transaction.Status,
        "amount":        transaction.Amount,
        "currency":      transaction.Currency,
        "reference":     transaction.Reference,
    })
}
```

### Routes (internal/modules/payments/routes.go)

```go
package payments

import (
    "github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, handler *PaymentHandler) {
    payments := router.Group("/payments")
    {
        payments.POST("", handler.CreatePayment)
        payments.GET("/:transaction_id", handler.GetPaymentStatus)
    }
}
```

---

## Webhook Handling

Wompi sends webhooks for transaction status changes. You need to verify the signature.

### Webhook Handler

```go
package payments

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "io"
    "log"
    "net/http"

    "github.com/gin-gonic/gin"
)

type WebhookHandler struct {
    eventKey string
}

func NewWebhookHandler(eventKey string) *WebhookHandler {
    return &WebhookHandler{eventKey: eventKey}
}

// VerifySignature verifies the webhook signature
func (h *WebhookHandler) VerifySignature(payload []byte, signature string) bool {
    mac := hmac.New(sha256.New, []byte(h.eventKey))
    mac.Write(payload)
    expectedSignature := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

// HandleWebhook - POST /api/v1/payments/webhook
func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
    // Read body
    body, err := io.ReadAll(c.Request.Body)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
        return
    }

    // Verify signature
    signature := c.GetHeader("Wompi-Signature")
    if signature == "" || !h.VerifySignature(body, signature) {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
        return
    }

    // Parse webhook event
    var event WebhookEvent
    if err := json.Unmarshal(body, &event); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
        return
    }

    // Process event based on type
    switch event.Event {
    case "transaction.updated":
        h.handleTransactionUpdate(event.Data.Transaction)
    default:
        log.Printf("Unknown event type: %s", event.Event)
    }

    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *WebhookHandler) handleTransactionUpdate(transaction Transaction) {
    // Update order status in database based on transaction.Status
    // "APPROVED" -> Order completed
    // "DECLINED" -> Order failed
    // "VOIDED" -> Order cancelled
}

// WebhookEvent - Webhook event structure
type WebhookEvent struct {
    Event       string      `json:"event"`
    Data        WebhookData `json:"data"`
    Timestamp   string      `json:"timestamp"`
    Signature   string      `json:"signature"`
}

type WebhookData struct {
    Transaction Transaction `json:"transaction"`
}
```

---

## Error Codes

### Transaction Statuses

| Status | Description |
|--------|-------------|
| `PENDING` | Transaction is pending (awaiting payment) |
| `APPROVED` | Transaction was approved |
| `DECLINED` | Transaction was declined |
| `VOIDED` | Transaction was cancelled/voided |
| `ERROR` | An error occurred |

### Common Error Codes

| Code | Description |
|------|-------------|
| `PAYMENT_DECLINED` | Card was declined |
| `INSUFFICIENT_FUNDS` | Insufficient funds |
| `INVALID_CARD` | Invalid card number |
| `EXPIRED_CARD` | Card has expired |
| `INVALID_CVV` | Invalid CVV |
| `CARD_NOT_SUPPORTED` | Card type not supported |
| `BANK_ERROR` | Bank error |
| `AMOUNT_TOO_HIGH` | Amount exceeds limit |

---

## Integration with Orders Module

The payment module integrates with the orders module as follows:

```go
// OrderService - After payment is approved
func (s *OrderService) CreateOrderFromCart(cart *Cart, shippingAddress string, notes string, paymentToken string) (*Order, error) {
    // 1. Calculate total from cart items
    total := s.calculateCartTotal(cart)
    
    // 2. Create payment
    payment, err := s.paymentService.CreatePayment(
        orderID,           // Reference
        total,            // Amount
        "COP",            // Currency  
        user.Email,       // Customer email
        paymentToken,     // Payment token
        "https://yourapp.com/payment/return", // Redirect URL
    )
    if err != nil {
        return nil, err
    }
    
    // 3. Wait for payment approval (via webhook or polling)
    // 4. Create order only if payment approved
    order := &Order{
        UserID:          user.ID,
        Status:          "pending_payment",
        TotalPrice:      total,
        ShippingAddress: shippingAddress,
        Notes:           notes,
        PaymentID:       payment.ID,
        Items:           cart.Items,
    }
    
    return s.orderRepo.Create(order)
}
```

---

## Testing

### Sandbox Testing

Use the sandbox environment with test keys:
- Base URL: `https://sandbox.wompi.co/v1`
- Keys: `pub_test_*`, `prv_test_*`

### Test Cards

Wompi provides test card numbers:
- Approved: `4242424242424242`
- Declined: `4000000000000002`

---

## Security Best Practices

1. **Never expose private key** on client side
2. **Verify webhook signatures** before processing
3. **Use HTTPS** in production
4. **Validate amounts** on server (don't trust client)
5. **Store transaction IDs** for reconciliation
6. **Implement idempotency** for payment requests

---

## References

- [Wompi Documentation](https://docs.wompi.co/docs/colombia/inicio-rapido/)
- [API Reference](https://app.swaggerhub.com/apis-docs/waybox/wompi/1.2.0)
- [Merchant Dashboard](https://comercios.wompi.co/)
- [Wompi Support](https://soporte.wompi.co/hc/es-419)

---

*Last updated: March 2026*
*For Bey API - E-commerce Project*
