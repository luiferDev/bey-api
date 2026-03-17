# Payment Flow - Complete Guide

This document describes the complete payment flow for the Bey API e-commerce application using Wompi as the payment gateway.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Flow Overview](#flow-overview)
3. [Authentication](#authentication)
4. [Cart Operations](#cart-operations)
5. [Order Creation](#order-creation)
6. [Payment Methods](#payment-methods)
   - [Method A: Direct Payment](#method-a-direct-payment)
   - [Method B: Payment Links](#method-b-payment-links)
7. [Webhook Processing](#webhook-processing)
8. [Complete Flow Example](#complete-flow-example)

---

## Prerequisites

### Configuration

Ensure your `config.yaml` has Wompi configured:

```yaml
wompi:
  enabled: true
  environment: sandbox
  public_key: "pub_test_xxxxxxxxxxxxx"
  private_key: "prv_test_xxxxxxxxxxxxx"
  event_key: "test_events_xxxxxxxxxxxxx"
  integrity_key: "test_integrity_xxxxxxxxxxxxx"
  base_url: "https://sandbox.wompi.co/v1"
```

### Test Cards (Sandbox)

Wompi provides test cards for sandbox testing:

| Card Number | Result |
|-------------|--------|
| `4242424242424242` | APPROVED |
| `4000000000000002` | DECLINED |
| `4000000000009995` | INSUFFICIENT_FUNDS |

---

## Flow Overview

### Method A: Direct Payment

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Client    │───→│  Add to    │───→│   Create   │───→│  Create    │───→│ Complete   │
│  (Frontend) │    │    Cart    │    │   Order    │    │ Transaction│    │   Payment  │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                                                                                  │
                                                                                  ↓
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐          │
│   Update    │←───│            │←───│  Webhook    │←───│   Wompi     │          │
│    Order    │    │  Verify    │    │ Processing  │    │   Server    │──────────┘
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
```

### Method B: Payment Links

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Client    │───→│  Add to    │───→│   Create   │───→│   Share     │
│  (Frontend) │    │    Cart    │    │   Order    │    │ Payment Link│
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                                                                    │
                                                                    ↓
                                                             ┌─────────────┐
                                                             │   Client    │
                                                             │  (WhatsApp) │
                                                             └─────────────┘
                                                                    │
                                                                    ↓
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Update    │←───│  Webhook    │←───│   Wompi     │←───│  Completes  │
│    Order    │    │ Processing  │    │   Server    │    │   Payment   │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
```

---

## Authentication

All protected endpoints require a JWT token in the cookie.

### Login

```bash
curl -X POST "http://localhost:8080/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

**Response:**
```json
{
  "message": "Login successful",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "name": "John Doe"
  }
}
```

The JWT token is stored in the `access_token` cookie.

---

## Cart Operations

### 1. Add Item to Cart

```bash
curl -X POST "http://localhost:8080/api/v1/cart/items" \
  -H "Content-Type: application/json" \
  -b "access_token=YOUR_JWT_TOKEN" \
  -d '{
    "variant_id": 1,
    "quantity": 2
  }'
```

**Response:**
```json
{
  "user_id": 1,
  "items": [
    {
      "variant_id": 1,
      "quantity": 2
    }
  ],
  "created_at": "2026-03-16T10:00:00Z",
  "updated_at": "2026-03-16T10:00:00Z"
}
```

### 2. Get Cart

```bash
curl -X GET "http://localhost:8080/api/v1/cart" \
  -b "access_token=YOUR_JWT_TOKEN"
```

**Response:**
```json
{
  "user_id": 1,
  "items": [
    {
      "variant_id": 1,
      "quantity": 2
    }
  ],
  "created_at": "2026-03-16T10:00:00Z",
  "updated_at": "2026-03-16T10:00:00Z"
}
```

---

## Order Creation

### 3. Create Order from Cart

```bash
curl -X POST "http://localhost:8080/api/v1/orders" \
  -H "Content-Type: application/json" \
  -b "access_token=YOUR_JWT_TOKEN" \
  -d '{
    "shipping_address": "Calle 123 #45-67, Bogotá, Colombia",
    "notes": "Por favor tocar el timbre"
  }'
```

**Response:**
```json
{
  "id": 1,
  "user_id": 1,
  "status": "pending_payment",
  "total_price": 150000,
  "shipping_address": "Calle 123 #45-67, Bogotá, Colombia",
  "notes": "Por favor tocar el timbre",
  "items": [
    {
      "product_id": 1,
      "variant_id": 1,
      "quantity": 2,
      "unit_price": 75000
    }
  ],
  "payment_status": "pending",
  "payment_transaction_id": null,
  "created_at": "2026-03-16T10:00:00Z"
}
```

**Note:** The order is created with status `pending_payment`. The payment must be completed to confirm the order.

---

## Payment Methods

### Method A: Direct Payment

#### 4. Create Payment Transaction

```bash
curl -X POST "http://localhost:8080/api/v1/payments" \
  -H "Content-Type: application/json" \
  -b "access_token=YOUR_JWT_TOKEN" \
  -d '{
    "amount": "150000",
    "currency": "COP",
    "payment_token": "tok_test_xxxxxxxxxxxxx",
    "redirect_url": "http://localhost:3000/payment/return",
    "reference": "ORDER-1"
  }'
```

**Payload Details:**

| Field | Type | Description |
|-------|------|-------------|
| `amount` | string | Amount in COP (pesos) |
| `currency` | string | Currency code (COP) |
| `payment_token` | string | Token from Wompi widget |
| `redirect_url` | string | URL to redirect after payment |
| `reference` | string | Your order ID |

**Response:**
```json
{
  "transaction_id": " txn_abc123xyz",
  "status": "PENDING",
  "redirect_url": "https://sandbox.wompi.co/pay/txn_abc123xyz"
}
```

#### 5. Redirect to Wompi

The client should be redirected to the `redirect_url` to complete the payment on Wompi's secure page.

```bash
# Example redirect URL to open in browser
https://sandbox.wompi.co/pay/txn_abc123xyz
```

#### 6. After Payment Completion

Once the payment is processed, Wompi will:
1. Redirect to your `redirect_url` with the transaction ID
2. Send a webhook to your server with the payment status

---

### Method B: Payment Links

#### 4. Create Payment Link

```bash
curl -X POST "http://localhost:8080/api/v1/payments/links" \
  -H "Content-Type: application/json" \
  -b "access_token=YOUR_JWT_TOKEN" \
  -d '{
    "amount_in_cents": 15000000,
    "description": "Pago ORDER-1",
    "reference": "ORDER-1",
    "redirect_url": "http://localhost:3000/payment/return"
  }'
```

**Payload Details:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `amount_in_cents` | int | No* | Amount in cents (COP). If not provided, client chooses amount |
| `description` | string | Yes | Description of the payment |
| `reference` | string | Yes | Your order ID |
| `redirect_url` | string | No | URL after payment |
| `single_use` | boolean | No | If true, link expires after one use (default: false) |
| `expires_at` | string | No | Expiration date (ISO 8601) |

*If `amount_in_cents` is not provided, it's an "open amount" link

**Response:**
```json
{
  "id": 1,
  "order_id": 1,
  "wompi_link_id": "abc123xyz",
  "url": "https://checkout.wompi.co/l/abc123xyz",
  "amount": 150000,
  "status": "active",
  "expires_at": null,
  "created_at": "2026-03-16T10:00:00Z"
}
```

#### 5. Share Payment Link

The payment link can be shared with the customer:

```
https://checkout.wompi.co/l/abc123xyz
```

The customer can pay by:
- Opening the link in a browser
- Scanning a QR code
- Clicking a link in WhatsApp/Email

---

## Webhook Processing

### 6. Wompi Sends Webhook

Wompi will send a webhook to your server when payment status changes:

```bash
# Webhook endpoint (no auth required)
POST http://localhost:8080/api/v1/payments/webhook
```

**Webhook Payload:**

```json
{
  "event": "transaction.updated",
  "data": {
    "transaction": {
      "id": "txn_abc123xyz",
      "status": "APPROVED",
      "status_message": "approved",
      "amount": 15000000,
      "currency": "COP",
      "reference": "ORDER-1",
      "payment_source": {
        "type": "CARD",
        "token": "tok_test_xxxxxxxxxxxxx"
      },
      "created_at": "2026-03-16T10:05:00Z",
      "updated_at": "2026-03-16T10:05:00Z"
    }
  },
  "timestamp": "2026-03-16T10:05:00Z",
  "signature": "sha256=abc123..."
}
```

### 7. Server Processes Webhook

The server:
1. Verifies the webhook signature
2. Looks up the order by `reference`
3. Updates the order status based on transaction status

**Status Mapping:**

| Wompi Status | Order Status | Payment Status |
|---------------|--------------|----------------|
| `APPROVED` | `confirmed` | `paid` |
| `DECLINED` | `cancelled` | `failed` |
| `VOIDED` | `cancelled` | `failed` |
| `PENDING` | `pending_payment` | `pending` |
| `ERROR` | `cancelled` | `failed` |

---

## Complete Flow Example

### Complete cURL Sequence (Method A: Direct Payment)

```bash
# 1. Login
curl -X POST "http://localhost:8080/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}'

# Save the cookie for subsequent requests
COOKIE_FILE=$(mktemp)

# 2. Add item to cart
curl -X POST "http://localhost:8080/api/v1/cart/items" \
  -H "Content-Type: application/json" \
  -b "access_token=YOUR_JWT_TOKEN" \
  -c $COOKIE_FILE \
  -d '{"variant_id": 1, "quantity": 2}'

# 3. Get cart to verify
curl -X GET "http://localhost:8080/api/v1/cart" \
  -b "access_token=YOUR_JWT_TOKEN" \
  -c $COOKIE_FILE

# 4. Create order from cart
curl -X POST "http://localhost:8080/api/v1/orders" \
  -H "Content-Type: application/json" \
  -b "access_token=YOUR_JWT_TOKEN" \
  -c $COOKIE_FILE \
  -d '{"shipping_address": "Calle 123, Bogota", "notes": "Entregar en porteria"}'

# 5. Create payment transaction
curl -X POST "http://localhost:8080/api/v1/payments" \
  -H "Content-Type: application/json" \
  -b "access_token=YOUR_JWT_TOKEN" \
  -c $COOKIE_FILE \
  -d '{
    "amount": "150000",
    "currency": "COP",
    "payment_token": "tok_test_xxxxxxxxxxxxx",
    "redirect_url": "http://localhost:3000/payment/return",
    "reference": "ORDER-1"
  }'

# Response includes redirect_url to Wompi:
# {"transaction_id": "txn_abc123", "status": "PENDING", "redirect_url": "https://sandbox.wompi.co/pay/txn_abc123"}

# 6. User completes payment at Wompi (manually in browser)
# URL: https://sandbox.wompi.co/pay/txn_abc123

# 7. Webhook updates order automatically
# Check order status:
curl -X GET "http://localhost:8080/api/v1/orders/1" \
  -b "access_token=YOUR_JWT_TOKEN"
```

### Complete cURL Sequence (Method B: Payment Links)

```bash
# 1-4. Same as above (Login → Cart → Order)

# 5. Create payment link
curl -X POST "http://localhost:8080/api/v1/payments/links" \
  -H "Content-Type: application/json" \
  -b "access_token=YOUR_JWT_TOKEN" \
  -d '{
    "amount_in_cents": 15000000,
    "description": "Pago ORDER-1",
    "reference": "ORDER-1",
    "redirect_url": "http://localhost:3000/payment/return"
  }'

# Response:
# {
#   "id": 1,
#   "url": "https://checkout.wompi.co/l/abc123xyz",
#   "amount": 150000,
#   "status": "active"
# }

# 6. Share link with customer (via WhatsApp, Email, etc.)
# URL: https://checkout.wompi.co/l/abc123xyz

# 7. Customer completes payment

# 8. Webhook updates order automatically

# 9. Check order status
curl -X GET "http://localhost:8080/api/v1/orders/1" \
  -b "access_token=YOUR_JWT_TOKEN"
```

---

## Order Status Flow

```
┌─────────────┐
│ pending_    │ ─── Payment Created
│   payment   │     (Order created, awaiting payment)
└─────────────┘
      │
      │ APPROVED (Webhook)
      ↓
┌─────────────┐
│  confirmed  │ ─── Payment Successful
│             │     (Order confirmed, ready for shipping)
└─────────────┘

OR

      │ DECLINED/VOIDED/ERROR (Webhook)
      ↓
┌─────────────┐
│ cancelled  │ ─── Payment Failed
│            │    (Order cancelled)
└─────────────┘
```

---

## Error Handling

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `400 - invalid amount` | Amount must be > 0 | Check amount field |
| `400 - insufficient stock` | Product out of stock | Check inventory |
| `401 - unauthorized` | Missing/invalid JWT | Login first |
| `404 - order not found` | Invalid reference | Check order ID |
| `Wompi: DECLINED` | Card declined | Use test card 4242... |
| `Wompi: INSUFFICIENT_FUNDS` | No funds | Use test card 9999... |

### Check Payment Status

```bash
curl -X GET "http://localhost:8080/api/v1/payments/txn_abc123xyz" \
  -b "access_token=YOUR_JWT_TOKEN"
```

**Response:**
```json
{
  "id": "txn_abc123xyz",
  "status": "APPROVED",
  "amount": 15000000,
  "currency": "COP",
  "reference": "ORDER-1",
  "payment_method": "CARD",
  "created_at": "2026-03-16T10:00:00Z"
}
```

---

## Testing in Sandbox

### Test Scenarios

1. **Successful Payment:**
   - Use card: `4242424242424242`
   - Any future expiry date
   - Any 3-digit CVV

2. **Declined Payment:**
   - Use card: `4000000000000002`

3. **Insufficient Funds:**
   - Use card: `4000000000009995`

4. **Payment Link - Open Amount:**
   - Create link without `amount_in_cents`
   - Customer enters any amount

5. **Payment Link - Single Use:**
   - Create link with `"single_use": true`
   - Link expires after first payment

---

## Security Notes

1. **Signature Verification:** All webhooks are verified using HMAC-SHA256
2. **Amount Validation:** Server validates amount matches order total
3. **Idempotency:** Duplicate webhook events are ignored
4. **HTTPS:** Always use HTTPS in production
5. **Never expose private keys:** Only use in server-side code

---

*Document updated: March 2026*
*For Bey API E-commerce Project*
