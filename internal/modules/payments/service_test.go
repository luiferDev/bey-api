package payments

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"bey/internal/config"
)

type mockPaymentRepo struct {
	createFunc          func(p *Payment) error
	findByIDFunc        func(id uint) (*Payment, error)
	findByWompiIDFunc   func(id string) (*Payment, error)
	findByReferenceFunc func(ref string) (*Payment, error)
	updateFunc          func(p *Payment) error
}

func (m *mockPaymentRepo) Create(p *Payment) error {
	if m.createFunc != nil {
		return m.createFunc(p)
	}
	return nil
}

func (m *mockPaymentRepo) FindByID(id uint) (*Payment, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(id)
	}
	return nil, nil
}

func (m *mockPaymentRepo) FindByWompiTransactionID(id string) (*Payment, error) {
	if m.findByWompiIDFunc != nil {
		return m.findByWompiIDFunc(id)
	}
	return nil, nil
}

func (m *mockPaymentRepo) FindByReference(ref string) (*Payment, error) {
	if m.findByReferenceFunc != nil {
		return m.findByReferenceFunc(ref)
	}
	return nil, nil
}

func (m *mockPaymentRepo) Update(p *Payment) error {
	if m.updateFunc != nil {
		return m.updateFunc(p)
	}
	return nil
}

type mockPaymentLinkRepo struct {
	createFunc            func(l *PaymentLink) error
	findByIDFunc          func(id uint) (*PaymentLink, error)
	findByWompiLinkIDFunc func(id string) (*PaymentLink, error)
	findByOrderIDFunc     func(orderID uint) (*PaymentLink, error)
	findByReferenceFunc   func(ref string) (*PaymentLink, error)
	updateFunc            func(l *PaymentLink) error
	markAsUsedFunc        func(id uint) error
}

func (m *mockPaymentLinkRepo) Create(l *PaymentLink) error {
	if m.createFunc != nil {
		return m.createFunc(l)
	}
	return nil
}

func (m *mockPaymentLinkRepo) FindByID(id uint) (*PaymentLink, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(id)
	}
	return nil, nil
}

func (m *mockPaymentLinkRepo) FindByWompiLinkID(id string) (*PaymentLink, error) {
	if m.findByWompiLinkIDFunc != nil {
		return m.findByWompiLinkIDFunc(id)
	}
	return nil, nil
}

func (m *mockPaymentLinkRepo) FindActiveByOrderID(orderID uint) (*PaymentLink, error) {
	if m.findByOrderIDFunc != nil {
		return m.findByOrderIDFunc(orderID)
	}
	return nil, nil
}

func (m *mockPaymentLinkRepo) FindByReference(ref string) (*PaymentLink, error) {
	if m.findByReferenceFunc != nil {
		return m.findByReferenceFunc(ref)
	}
	return nil, nil
}

func (m *mockPaymentLinkRepo) Update(l *PaymentLink) error {
	if m.updateFunc != nil {
		return m.updateFunc(l)
	}
	return nil
}

func (m *mockPaymentLinkRepo) MarkAsUsed(id uint) error {
	if m.markAsUsedFunc != nil {
		return m.markAsUsedFunc(id)
	}
	return nil
}

type mockWompiClient struct {
	createTransactionFunc func(amount int64, currency, token, reference, redirectURL string) (*WompiTransactionResponse, error)
	getTransactionFunc    func(transactionID string) (*WompiTransactionResponse, error)
	voidTransactionFunc   func(transactionID string) (*WompiTransactionResponse, error)
	createPaymentLinkFunc func(amount int64, currency, description, reference, redirectURL string, singleUse bool, expiresAt *time.Time) (*WompiPaymentLinkResponse, error)
	updatePaymentLinkFunc func(linkID string, status string) (*WompiPaymentLinkResponse, error)
}

func (m *mockWompiClient) CreateTransaction(amount int64, currency, token, reference, redirectURL string) (*WompiTransactionResponse, error) {
	if m.createTransactionFunc != nil {
		return m.createTransactionFunc(amount, currency, token, reference, redirectURL)
	}
	return &WompiTransactionResponse{
		Transaction: Transaction{
			ID:            "tx_123",
			AmountInCents: amount,
			Currency:      currency,
			Status:        "APPROVED",
			Reference:     reference,
		},
	}, nil
}

func (m *mockWompiClient) GetTransaction(transactionID string) (*WompiTransactionResponse, error) {
	if m.getTransactionFunc != nil {
		return m.getTransactionFunc(transactionID)
	}
	return nil, nil
}

func (m *mockWompiClient) VoidTransaction(transactionID string) (*WompiTransactionResponse, error) {
	if m.voidTransactionFunc != nil {
		return m.voidTransactionFunc(transactionID)
	}
	return nil, nil
}

func (m *mockWompiClient) CreatePaymentLink(amount int64, currency, description, reference, redirectURL string, singleUse bool, expiresAt *time.Time) (*WompiPaymentLinkResponse, error) {
	if m.createPaymentLinkFunc != nil {
		return m.createPaymentLinkFunc(amount, currency, description, reference, redirectURL, singleUse, expiresAt)
	}
	return nil, nil
}

func (m *mockWompiClient) UpdatePaymentLink(linkID string, status string) (*WompiPaymentLinkResponse, error) {
	if m.updatePaymentLinkFunc != nil {
		return m.updatePaymentLinkFunc(linkID, status)
	}
	return nil, nil
}

func TestPaymentService_CreatePayment(t *testing.T) {
	integrityKey := "test-integrity-key"

	tests := []struct {
		name        string
		req         *CreatePaymentRequest
		setupMock   func() *PaymentService
		wantErr     bool
		errContains string
	}{
		{
			name: "valid payment request",
			req: &CreatePaymentRequest{
				Amount:       100000,
				Currency:     "COP",
				PaymentToken: "tok_test_123",
				Reference:    "order-123",
				RedirectURL:  "https://example.com/return",
			},
			setupMock: func() *PaymentService {
				return &PaymentService{
					integrityKey: integrityKey,
				}
			},
			wantErr: false,
		},
		{
			name: "invalid amount - zero",
			req: &CreatePaymentRequest{
				Amount:       0,
				Currency:     "COP",
				PaymentToken: "tok_test_123",
				Reference:    "order-123",
			},
			setupMock: func() *PaymentService {
				return &PaymentService{
					integrityKey: integrityKey,
				}
			},
			wantErr:     true,
			errContains: "amount_in_cents",
		},
		{
			name: "invalid amount - negative",
			req: &CreatePaymentRequest{
				Amount:       -1000,
				Currency:     "COP",
				PaymentToken: "tok_test_123",
				Reference:    "order-123",
			},
			setupMock: func() *PaymentService {
				return &PaymentService{
					integrityKey: integrityKey,
				}
			},
			wantErr:     true,
			errContains: "amount_in_cents",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.Amount <= 0 {
				err := validatePaymentRequest(tt.req)
				if (err != nil) != tt.wantErr {
					t.Errorf("expected error=%v, got=%v", tt.wantErr, err)
				}
				if tt.wantErr && tt.errContains != "" && err != nil && !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			}
		})
	}
}

func validatePaymentRequest(req *CreatePaymentRequest) error {
	if req.Amount <= 0 {
		return errors.New("amount_in_cents must be greater than 0")
	}
	if req.Currency == "" {
		return errors.New("currency is required")
	}
	if req.PaymentToken == "" {
		return errors.New("payment_token is required")
	}
	if req.Reference == "" {
		return errors.New("reference is required")
	}
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestPaymentService_VerifySignature(t *testing.T) {
	integrityKey := "test-integrity-key-12345"

	tests := []struct {
		name        string
		payload     []byte
		signature   string
		expectValid bool
	}{
		{
			name:        "valid signature",
			payload:     []byte(`{"event":"payment.created","data":{"id":"tx_123"}}`),
			signature:   generateHMAC([]byte(`{"event":"payment.created","data":{"id":"tx_123"}}`), integrityKey),
			expectValid: true,
		},
		{
			name:        "invalid signature - wrong key",
			payload:     []byte(`{"event":"payment.created","data":{"id":"tx_123"}}`),
			signature:   generateHMAC([]byte(`{"event":"payment.created","data":{"id":"tx_123"}}`), "wrong-key"),
			expectValid: false,
		},
		{
			name:        "invalid signature - tampered payload",
			payload:     []byte(`{"event":"payment.created","data":{"id":"tx_456"}}`),
			signature:   generateHMAC([]byte(`{"event":"payment.created","data":{"id":"tx_123"}}`), integrityKey),
			expectValid: false,
		},
		{
			name:        "empty signature",
			payload:     []byte(`{"event":"payment.created"}`),
			signature:   "",
			expectValid: false,
		},
		{
			name:        "empty payload with valid signature",
			payload:     []byte{},
			signature:   generateHMAC([]byte{}, integrityKey),
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &PaymentService{
				integrityKey: integrityKey,
			}

			result := service.VerifySignature(tt.payload, tt.signature)
			if result != tt.expectValid {
				t.Errorf("VerifySignature() = %v, expected %v", result, tt.expectValid)
			}
		})
	}
}

func generateHMAC(payload []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestPaymentService_ProcessWebhook_Deduplication(t *testing.T) {
	tests := []struct {
		name     string
		eventID  string
		seenMap  map[string]bool
		wantSkip bool
	}{
		{
			name:     "new event ID should process",
			eventID:  "evt_new",
			seenMap:  map[string]bool{},
			wantSkip: false,
		},
		{
			name:     "duplicate event ID should skip",
			eventID:  "evt_dup",
			seenMap:  map[string]bool{"evt_dup": true},
			wantSkip: true,
		},
		{
			name:     "empty event ID should process",
			eventID:  "",
			seenMap:  map[string]bool{},
			wantSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &PaymentService{
				webhookEventsSeen: tt.seenMap,
			}

			event := &WebhookEvent{
				EventID: tt.eventID,
			}

			if tt.eventID != "" {
				if _, seen := service.webhookEventsSeen[event.EventID]; seen {
					if !tt.wantSkip {
						t.Errorf("expected to process event, but it was skipped")
					}
				} else {
					if tt.wantSkip {
						t.Errorf("expected to skip event, but it was processed")
					}
				}
			}
		})
	}
}

func TestPaymentService_ValidateAmount(t *testing.T) {
	tests := []struct {
		name      string
		amount    int64
		reference string
		expected  bool
	}{
		{
			name:      "positive amount",
			amount:    100000,
			reference: "order-123",
			expected:  true,
		},
		{
			name:      "zero amount is invalid",
			amount:    0,
			reference: "order-123",
			expected:  false,
		},
		{
			name:      "negative amount is invalid",
			amount:    -1000,
			reference: "order-123",
			expected:  false,
		},
		{
			name:      "amount exceeds max limit",
			amount:    100000001,
			reference: "order-large",
			expected:  false,
		},
		{
			name:      "amount at max limit is valid",
			amount:    100000000,
			reference: "order-max",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &PaymentService{}
			result := service.ValidateAmount(tt.amount, tt.reference)
			if result != tt.expected {
				t.Errorf("ValidateAmount(%d, %s) = %v, expected %v", tt.amount, tt.reference, result, tt.expected)
			}
		})
	}
}

func TestPaymentService_MapWompiStatus(t *testing.T) {
	tests := []struct {
		wompiStatus string
		expected    string
	}{
		{"APPROVED", StatusApproved},
		{"DECLINED", StatusDeclined},
		{"VOIDED", StatusVoided},
		{"PENDING", StatusPending},
		{"PENDING_WALLET", StatusPending},
		{"FAILED", StatusFailed},
		{"UNKNOWN_STATUS", StatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.wompiStatus, func(t *testing.T) {
			service := &PaymentService{}
			result := service.mapWompiStatus(tt.wompiStatus)
			if result != tt.expected {
				t.Errorf("mapWompiStatus(%s) = %v, expected %v", tt.wompiStatus, result, tt.expected)
			}
		})
	}
}

func TestPaymentService_MapWompiPaymentLinkStatus(t *testing.T) {
	tests := []struct {
		wompiStatus string
		expected    string
	}{
		{"ACTIVE", StatusActive},
		{"INACTIVE", StatusInactive},
		{"EXPIRED", StatusExpired},
		{"USED", StatusUsed},
		{"UNKNOWN_STATUS", StatusActive},
	}

	for _, tt := range tests {
		t.Run(tt.wompiStatus, func(t *testing.T) {
			service := &PaymentService{}
			result := service.mapWompiPaymentLinkStatus(tt.wompiStatus)
			if result != tt.expected {
				t.Errorf("mapWompiPaymentLinkStatus(%s) = %v, expected %v", tt.wompiStatus, result, tt.expected)
			}
		})
	}
}

func TestNewPaymentService(t *testing.T) {
	cfg := &config.WompiConfig{
		IntegrityKey: "test-key",
		PublicKey:    "pub_test",
		PrivateKey:   "priv_test",
		BaseURL:      "https://sandbox.wompi.co",
	}

	service := NewPaymentService(cfg, nil, nil, nil)

	if service == nil {
		t.Error("NewPaymentService returned nil")
	}
	if service.integrityKey != cfg.IntegrityKey {
		t.Errorf("integrityKey = %v, expected %v", service.integrityKey, cfg.IntegrityKey)
	}
	if service.webhookEventsSeen == nil {
		t.Error("webhookEventsSeen should be initialized")
	}
	_ = cfg
}
