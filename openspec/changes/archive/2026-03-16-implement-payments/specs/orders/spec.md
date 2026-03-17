# Orders Integration Specification

## Purpose

Define integration points between payments and orders modules.

## Requirements

### Requirement: Order Status Update on Payment

When a payment is approved, the associated order MUST be updated to "confirmed" status.

#### Scenario: Payment approved via webhook

- GIVEN order has status "pending" with payment ID 456
- WHEN payment webhook received with status APPROVED
- THEN order status updated to "confirmed"
- AND order.payment_status updated to "completed"

### Requirement: Order Status Update on Payment Failure

When a payment is declined or voided, the order payment status MUST reflect the failure.

#### Scenario: Payment declined

- GIVEN order has status "pending" with payment ID 456
- WHEN payment webhook received with status DECLINED
- THEN order.payment_status updated to "failed"
- AND order status remains "pending" (customer can retry)