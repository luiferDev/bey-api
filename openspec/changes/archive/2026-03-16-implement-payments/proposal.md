# Proposal: Implement Payments Module with Wompi

## Intent

Enable e-commerce payments for the Bey API using Wompi, a Colombian payment gateway. The current orders module lacks payment processing capability - users can create orders but cannot pay for them. This change adds a standalone payments module that integrates with the existing orders flow.

## Scope

### In Scope
- Wompi integration for Colombian payment methods (credit cards, debit cards, Nequi, PSE)
- Payment creation, status query, and cancellation via Wompi API
- Webhook endpoint for async payment status updates
- Configuration for sandbox/production environments
- Integration points with orders module (update order status based on payment)
- Signature verification for webhook security
- **Payment Links** - Generate shareable payment URLs for WhatsApp/Email/Social media

### Out of Scope
- Recurring/subscription payments
- Refunds (future enhancement)
- Payment widget/frontend integration (client-side)
- Multiple currency support beyond COP

## Approach

Create a standalone `payments` module following existing module patterns (model/repository/service/handler/routes). The module will:

1. **Configuration**: Add `WompiConfig` to config.yaml with keys for sandbox/production
2. **Client**: HTTP client wrapping Wompi API (transactions, tokens, payment_links, events)
3. **Repository**: Database operations for Payment and PaymentLink entities
4. **Service**: Business logic for payment creation, validation, webhook processing
5. **Handler**: HTTP endpoints for payment operations
6. **Routes**: Register under `/api/v1/payments`

### Integration Points

- **Orders Module**: When payment status changes to APPROVED, update order status to "confirmed"
- **Inventory**: On payment approval, confirm variant stock sales (already handled by async task)
- **Cart**: Payment completes the cart → order → payment flow