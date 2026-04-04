package payments

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"bey/internal/config"
	"bey/internal/modules/orders"

	"github.com/gofrs/uuid/v5"
)

type PaymentService struct {
	wompiClient       *WompiClient
	paymentRepo       *PaymentRepository
	paymentLinkRepo   *PaymentLinkRepository
	orderService      *orders.OrderService
	integrityKey      string
	webhookEventsSeen map[string]bool
}

func NewPaymentService(
	cfg *config.WompiConfig,
	paymentRepo *PaymentRepository,
	paymentLinkRepo *PaymentLinkRepository,
	orderService *orders.OrderService,
) *PaymentService {
	return &PaymentService{
		wompiClient:       NewWompiClient(cfg),
		paymentRepo:       paymentRepo,
		paymentLinkRepo:   paymentLinkRepo,
		orderService:      orderService,
		integrityKey:      cfg.IntegrityKey,
		webhookEventsSeen: make(map[string]bool),
	}
}

func (s *PaymentService) CreatePayment(req *CreatePaymentRequest) (*Payment, error) {
	resp, err := s.wompiClient.CreateTransaction(
		req.Amount,
		req.Currency,
		req.PaymentToken,
		req.Reference,
		req.RedirectURL,
	)
	if err != nil {
		return nil, fmt.Errorf("create wompi transaction: %w", err)
	}

	payment := &Payment{
		OrderID:            uuid.Nil,
		WompiTransactionID: resp.Transaction.ID,
		Amount:             resp.Transaction.AmountInCents,
		Currency:           resp.Transaction.Currency,
		Status:             s.mapWompiStatus(resp.Transaction.Status),
		PaymentMethod:      resp.Transaction.PaymentMethod,
		PaymentToken:       req.PaymentToken,
		RedirectURL:        req.RedirectURL,
		Reference:          req.Reference,
	}

	if err := s.paymentRepo.Create(payment); err != nil {
		return nil, fmt.Errorf("save payment: %w", err)
	}

	log.Printf("Created payment %d with Wompi ID %s, status: %s", payment.ID, payment.WompiTransactionID, payment.Status)

	return payment, nil
}

func (s *PaymentService) GetPayment(id uuid.UUID) (*Payment, error) {
	payment, err := s.paymentRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("find payment: %w", err)
	}
	if payment == nil {
		return nil, errors.New("payment not found")
	}
	return payment, nil
}

func (s *PaymentService) GetOrderByPaymentID(orderID uuid.UUID) (*orders.Order, error) {
	if s.orderService == nil {
		return nil, errors.New("order service not available")
	}
	return s.orderService.GetOrderByID(orderID)
}

func (s *PaymentService) GetPaymentStatus(wompiTransactionID string) (*Payment, error) {
	resp, err := s.wompiClient.GetTransaction(wompiTransactionID)
	if err != nil {
		return nil, fmt.Errorf("get wompi transaction: %w", err)
	}

	payment, err := s.paymentRepo.FindByWompiTransactionID(wompiTransactionID)
	if err != nil {
		return nil, fmt.Errorf("find payment: %w", err)
	}
	if payment == nil {
		return nil, errors.New("payment not found")
	}

	payment.Status = s.mapWompiStatus(resp.Transaction.Status)
	if err := s.paymentRepo.Update(payment); err != nil {
		log.Printf("Failed to update payment status: %v", err)
	}

	return payment, nil
}

func (s *PaymentService) CancelPayment(id uuid.UUID) (*Payment, error) {
	payment, err := s.paymentRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("find payment: %w", err)
	}
	if payment == nil {
		return nil, errors.New("payment not found")
	}

	if payment.Status == StatusApproved || payment.Status == StatusVoided {
		return nil, fmt.Errorf("cannot void payment with status: %s", payment.Status)
	}

	resp, err := s.wompiClient.VoidTransaction(payment.WompiTransactionID)
	if err != nil {
		return nil, fmt.Errorf("void wompi transaction: %w", err)
	}

	payment.Status = s.mapWompiStatus(resp.Transaction.Status)
	if err := s.paymentRepo.Update(payment); err != nil {
		return nil, fmt.Errorf("update payment: %w", err)
	}

	log.Printf("Voided payment %d, status: %s", payment.ID, payment.Status)

	return payment, nil
}

func (s *PaymentService) CreatePaymentLink(req *CreatePaymentLinkRequest) (*PaymentLink, error) {
	orderID, err := uuid.FromString(req.OrderID)
	if err != nil {
		return nil, errors.New("invalid order ID")
	}
	activeLink, err := s.paymentLinkRepo.FindActiveByOrderID(orderID)
	if err != nil {
		log.Printf("Warning: failed to check existing payment link: %v", err)
	}
	if activeLink != nil {
		log.Printf("Active payment link already exists for order %s: %s", req.OrderID, activeLink.WompiLinkID)
		return activeLink, nil
	}

	currency := req.Currency
	if currency == "" {
		currency = CurrencyCOP
	}

	resp, err := s.wompiClient.CreatePaymentLink(
		req.AmountInCents,
		currency,
		req.Description,
		req.Reference,
		req.RedirectURL,
		req.SingleUse,
		req.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create wompi payment link: %w", err)
	}

	expiresAt := parseWompiTime(resp.PaymentLink.ExpiresAt)

	link := &PaymentLink{
		OrderID:     orderID,
		WompiLinkID: resp.PaymentLink.ID,
		URL:         resp.PaymentLink.URL,
		Amount:      resp.PaymentLink.AmountInCents,
		Currency:    resp.PaymentLink.Currency,
		Description: resp.PaymentLink.Description,
		Status:      s.mapWompiPaymentLinkStatus(resp.PaymentLink.Status),
		SingleUse:   resp.PaymentLink.SingleUse,
		ExpiresAt:   expiresAt,
		RedirectURL: orString(resp.PaymentLink.RedirectURL),
		Reference:   req.Reference,
	}

	if err := s.paymentLinkRepo.Create(link); err != nil {
		return nil, fmt.Errorf("save payment link: %w", err)
	}

	log.Printf("Created payment link %d with Wompi ID %s", link.ID, link.WompiLinkID)

	return link, nil
}

func (s *PaymentService) GetPaymentLink(id uuid.UUID) (*PaymentLink, error) {
	link, err := s.paymentLinkRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("find payment link: %w", err)
	}
	if link == nil {
		return nil, errors.New("payment link not found")
	}
	return link, nil
}

func (s *PaymentService) GetPaymentLinkByWompiID(wompiID string) (*PaymentLink, error) {
	link, err := s.paymentLinkRepo.FindByWompiLinkID(wompiID)
	if err != nil {
		return nil, fmt.Errorf("find payment link: %w", err)
	}
	if link == nil {
		return nil, errors.New("payment link not found")
	}
	return link, nil
}

func (s *PaymentService) ActivatePaymentLink(id uuid.UUID) (*PaymentLink, error) {
	link, err := s.paymentLinkRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("find payment link: %w", err)
	}
	if link == nil {
		return nil, errors.New("payment link not found")
	}

	if link.Status == StatusUsed {
		return nil, errors.New("payment link already used")
	}

	resp, err := s.wompiClient.UpdatePaymentLink(link.WompiLinkID, "ACTIVE")
	if err != nil {
		return nil, fmt.Errorf("activate wompi payment link: %w", err)
	}

	link.Status = s.mapWompiPaymentLinkStatus(resp.PaymentLink.Status)
	if err := s.paymentLinkRepo.Update(link); err != nil {
		return nil, fmt.Errorf("update payment link: %w", err)
	}

	log.Printf("Activated payment link %d", link.ID)

	return link, nil
}

func (s *PaymentService) DeactivatePaymentLink(id uuid.UUID) (*PaymentLink, error) {
	link, err := s.paymentLinkRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("find payment link: %w", err)
	}
	if link == nil {
		return nil, errors.New("payment link not found")
	}

	resp, err := s.wompiClient.UpdatePaymentLink(link.WompiLinkID, "INACTIVE")
	if err != nil {
		return nil, fmt.Errorf("deactivate wompi payment link: %w", err)
	}

	link.Status = s.mapWompiPaymentLinkStatus(resp.PaymentLink.Status)
	if err := s.paymentLinkRepo.Update(link); err != nil {
		return nil, fmt.Errorf("update payment link: %w", err)
	}

	log.Printf("Deactivated payment link %d", link.ID)

	return link, nil
}

func (s *PaymentService) ProcessWebhook(event *WebhookEvent) error {
	if event.EventID != "" {
		if _, seen := s.webhookEventsSeen[event.EventID]; seen {
			log.Printf("Webhook event %s already processed, skipping", event.EventID)
			return nil
		}
		s.webhookEventsSeen[event.EventID] = true
	}

	transaction := event.Transaction
	amount := transaction.AmountInCents

	if !s.ValidateAmount(amount, transaction.Reference) {
		log.Printf("Warning: Amount mismatch for reference %s", transaction.Reference)
	}

	payment, err := s.paymentRepo.FindByReference(transaction.Reference)
	if err != nil {
		return fmt.Errorf("find payment by reference: %w", err)
	}

	if payment == nil {
		link, err := s.paymentLinkRepo.FindByReference(transaction.Reference)
		if err != nil {
			return fmt.Errorf("find payment link by reference: %w", err)
		}
		if link == nil {
			log.Printf("No payment or link found for reference %s, creating new payment record", transaction.Reference)
			payment = &Payment{
				WompiTransactionID: transaction.ID,
				Amount:             amount,
				Currency:           transaction.Currency,
				Status:             s.mapWompiStatus(transaction.Status),
				Reference:          transaction.Reference,
			}
			if err := s.paymentRepo.Create(payment); err != nil {
				return fmt.Errorf("create payment: %w", err)
			}
		} else {
			oldStatus := link.Status
			link.Status = s.mapWompiStatus(transaction.Status)
			if err := s.paymentLinkRepo.Update(link); err != nil {
				return fmt.Errorf("update payment link: %w", err)
			}
			log.Printf("Updated payment link %d status from %s to %s", link.ID, oldStatus, link.Status)

			if s.orderService != nil && link.OrderID != uuid.Nil {
				switch link.Status {
				case StatusApproved:
					if err := s.orderService.UpdatePaymentStatus(link.OrderID, "paid", transaction.ID); err != nil {
						log.Printf("Failed to update order payment status: %v", err)
					}
				case StatusDeclined, StatusVoided:
					if err := s.orderService.UpdatePaymentStatus(link.OrderID, "failed", transaction.ID); err != nil {
						log.Printf("Failed to update order payment status: %v", err)
					}
				}
			}

			if link.SingleUse && link.Status == StatusApproved {
				if err := s.paymentLinkRepo.MarkAsUsed(link.ID); err != nil {
					log.Printf("Failed to mark payment link as used: %v", err)
				}
			}
		}
	} else {
		oldStatus := payment.Status
		payment.Status = s.mapWompiStatus(transaction.Status)
		if err := s.paymentRepo.Update(payment); err != nil {
			return fmt.Errorf("update payment: %w", err)
		}
		log.Printf("Updated payment %d status from %s to %s", payment.ID, oldStatus, payment.Status)

		if s.orderService != nil && payment.OrderID != uuid.Nil {
			switch payment.Status {
			case StatusApproved:
				if err := s.orderService.UpdatePaymentStatus(payment.OrderID, "paid", transaction.ID); err != nil {
					log.Printf("Failed to update order payment status: %v", err)
				}
			case StatusDeclined, StatusVoided:
				if err := s.orderService.UpdatePaymentStatus(payment.OrderID, "failed", transaction.ID); err != nil {
					log.Printf("Failed to update order payment status: %v", err)
				}
			}
		}
	}

	return nil
}

func (s *PaymentService) VerifySignature(payload []byte, signature string) bool {
	if s.integrityKey == "" {
		log.Printf("Warning: No integrity key configured, skipping signature verification")
		return true
	}

	mac := hmac.New(sha256.New, []byte(s.integrityKey))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

const maxAmountLimit int64 = 100000000

func (s *PaymentService) ValidateAmount(amount int64, reference string) bool {
	if amount <= 0 {
		log.Printf("Invalid amount: %d - amount must be greater than 0", amount)
		return false
	}

	if amount > maxAmountLimit {
		log.Printf("Invalid amount: %d - amount exceeds maximum limit of %d", amount, maxAmountLimit)
		return false
	}

	if reference == "" {
		return true
	}

	if s.paymentLinkRepo == nil && s.paymentRepo == nil {
		return true
	}

	if s.paymentLinkRepo != nil {
		link, err := s.paymentLinkRepo.FindByReference(reference)
		if err != nil {
			log.Printf("Warning: failed to find payment link by reference %s: %v", reference, err)
			return true
		}
		if link != nil {
			if link.Amount != amount {
				log.Printf("Amount mismatch for reference %s: expected %d, got %d", reference, link.Amount, amount)
				return false
			}
			return true
		}
	}

	if s.paymentRepo != nil {
		payment, err := s.paymentRepo.FindByReference(reference)
		if err != nil {
			log.Printf("Warning: failed to find payment by reference %s: %v", reference, err)
			return true
		}
		if payment != nil {
			if payment.Amount != amount {
				log.Printf("Amount mismatch for reference %s: expected %d, got %d", reference, payment.Amount, amount)
				return false
			}
			return true
		}
	}

	return true
}

func (s *PaymentService) mapWompiStatus(status string) string {
	switch status {
	case "APPROVED":
		return StatusApproved
	case "DECLINED":
		return StatusDeclined
	case "VOIDED":
		return StatusVoided
	case "PENDING", "PENDING_WALLET":
		return StatusPending
	case "FAILED":
		return StatusFailed
	default:
		return StatusPending
	}
}

func (s *PaymentService) mapWompiPaymentLinkStatus(status string) string {
	switch status {
	case "ACTIVE":
		return StatusActive
	case "INACTIVE":
		return StatusInactive
	case "EXPIRED":
		return StatusExpired
	case "USED":
		return StatusUsed
	default:
		return StatusActive
	}
}

func parseWompiTime(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		log.Printf("Failed to parse wompi time %s: %v", *s, err)
		return nil
	}
	return &t
}

func orString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
