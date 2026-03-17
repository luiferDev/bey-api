# Payments Specification

## Purpose

This specification defines the payments module for the Bey API, integrating with Wompi (Colombian payment gateway) to enable e-commerce transactions. The module supports payment transactions, payment links, and webhook handling for asynchronous payment status updates.

## Requirements

### Requirement: Wompi Configuration

The system MUST support Wompi configuration for sandbox and production environments.

The configuration MUST include:
- `wompi.presign_key` - Secret key for signature verification
- `wompi.public_key` - Public key for payment link creation
- `wompi.private_key` - Private key for API calls
- `wompi.currency` - Currency code (default: COP)
- `wompi.sandbox` - Boolean for sandbox mode
- `wompi.base_url` - Base URL for Wompi API

#### Scenario: Load Wompi configuration

- GIVEN `config.yaml` contains valid Wompi configuration
- WHEN the application starts
- THEN the payments module loads all Wompi credentials
- AND the system uses sandbox URL when `wompi.sandbox` is true

---

### Requirement: Create Payment Transaction

The system MUST create a payment transaction via Wompi API.

The payment creation MUST:
- Accept amount in cents ( COP, e.g., 10000 = $100 COP)
- Generate a unique idempotency key
- Support payment methods: credit_card, debit_card, nequi, pse
- Return payment ID and redirect URL for checkout
- Validate amount on server-side before sending to Wompi

#### Scenario: Create successful payment

- GIVEN authenticated user with order ID 123 totaling 50000 cents
- WHEN client requests `POST /api/v1/payments` with `{"order_id": 123, "amount": 50000, "payment_method": "credit_card"}`
- THEN system creates payment transaction in database with status "pending"
- AND system calls Wompi API to create payment
- AND response includes `{"payment_id": "wompi-xxx", "redirect_url": "https://checkout.wompi.co/..."}`

---

### Requirement: Query Payment Status

The system MUST query payment status from Wompi.

The status query MUST:
- Accept Wompi transaction ID
- Return current payment status (PENDING, APPROVED, DECLINED, VOIDED)
- Update local payment record with latest status

---

### Requirement: Void Payment

The system MUST allow voiding a pending payment.

The void operation MUST:
- Only allow voiding payments with status PENDING
- Call Wompi API to void the transaction
- Update local payment status to VOIDED

---

### Requirement: Webhook Handling

The system MUST handle Wompi webhooks for async payment updates.

The webhook handler MUST:
- Accept POST requests at `/api/v1/payments/webhook`
- Verify signature using presign_key
- Update payment status based on event type
- Handle event types: transaction.updated, payment_link.updated
- Implement idempotency to prevent duplicate processing

#### Scenario: Process APPROVED webhook

- GIVEN Wompi sends webhook for payment wompi-123 with status APPROVED
- WHEN system verifies signature and processes webhook
- THEN payment status updated to APPROVED
- AND associated order status updated to confirmed
- AND deduplication prevents re-processing same event

---

### Requirement: Payment Links

The system MUST create payment links via Wompi API.

Payment links MUST support:
- Fixed amount or open amount (customer enters)
- Single-use or multi-use
- Expiration date
- Custom reference (e.g., "invoice-123")

#### Scenario: Create fixed-amount payment link

- GIVEN authenticated user requests payment link for 50000 cents
- WHEN client requests `POST /api/v1/payments/links` with `{"amount": 50000, "description": "Order #123"}`
- THEN system creates PaymentLink in database with status ACTIVE
- AND system calls Wompi API to create payment link
- AND response includes `{"id": "wompi-link-xxx", "url": "https://checkout.wompi.co/l/xxx"}`

---

### Requirement: Query Payment Link

The system MUST query payment link status from Wompi.

The query MUST:
- Accept Wompi payment link ID
- Return current status and transactions
- Update local payment link record