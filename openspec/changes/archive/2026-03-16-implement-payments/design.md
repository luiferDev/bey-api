# Technical Design: Payments Module

## 1. Executive Summary

**Project**: bey_api  
**Module**: Payments (Wompi Integration)  
**Change**: implement-payments

### Goals
- Integrate Wompi payment gateway for Colombian market
- Support transactions, tokens, payment links, and webhooks
- Integrate with orders module for payment flow

### Scope
- Wompi client for API communication
- Payment service with business logic
- Payment links (fixed/open amount, single/multi use)
- Webhook handling with signature verification
- Order-payment integration

---

## 2. Technical Approach

### Architecture Pattern
**Hexagonal Architecture** with clear separation:
- **Domain**: Payment entity, status tracking
- **Application**: PaymentService, business logic
- **Infrastructure**: WompiClient (external API), Repository (DB)

### Module Structure
```
internal/modules/payments/
├── model.go          # Payment entity, enums
├── dto.go            # Request/Response DTOs  
├── repository.go     # DB operations
├── client.go         # Wompi API client
├── service.go        # Business logic
├── handler.go        # HTTP handlers
└── routes.go         # Route definitions
```

---

## 3. Architecture Decisions

### Decision 1: Payment Storage
**Option**: Store payments in database with status tracking  
**Chosen**: Yes - enables reconciliation, webhook updates, order integration  
**Tradeoff**: Adds DB writes but provides audit trail

### Decision 2: Payment Flow
**Option A**: Create order first, then payment  
**Option B**: Create payment first, then order on approval  
**Chosen**: Option B - order only confirmed on payment approval  

### Decision 3: Webhook Processing
**Option A**: Synchronous - process in request thread  
**Option B**: Async via task queue  
**Chosen**: Option A with error handling - status updates are simple

### Decision 4: Payment Links
**Chosen**: Support both fixed and open amount, single and multi-use

---

## 4. Data Model

### Payment Entity
- ID, OrderID, Amount, Currency
- WompiTransactionID, WompiStatus
- PaymentMethod, Status (PENDING/APPROVED/DECLINED/VOIDED)
- IdempotencyKey, RedirectURL, CreatedAt, UpdatedAt

### PaymentLink Entity
- ID, OrderID, Amount (nullable for open)
- WompiLinkID, WompiLinkURL
- IsSingleUse, IsActive, ExpiresAt
- Reference, Description, Status

---

## 5. API Endpoints

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | /api/v1/payments | JWT | Create payment |
| GET | /api/v1/payments/:id | JWT | Get payment |
| GET | /api/v1/payments/wompi/:id/status | JWT | Query Wompi status |
| POST | /api/v1/payments/:id/void | JWT | Void pending payment |
| POST | /api/v1/payments/webhook | None | Wompi webhook |
| POST | /api/v1/payments/links | JWT | Create payment link |
| GET | /api/v1/payments/links/:id | JWT | Get payment link |
| PATCH | /api/v1/payments/links/:id/activate | JWT | Activate link |
| PATCH | /api/v1/payments/links/:id/deactivate | JWT | Deactivate link |

---

## 6. Integration Points

### Orders Module
- PaymentService calls OrderService.UpdatePaymentStatus()
- Updates order status to confirmed on APPROVED
- Updates payment_status to failed on DECLINED/VOIDED

### Config
- WompiConfig added to config.yaml
- GetBaseURL() helper for sandbox/production URLs