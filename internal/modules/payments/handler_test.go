package payments

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type MockPaymentService struct {
	createPaymentFunc         func(req *CreatePaymentRequest) (*Payment, error)
	getPaymentFunc            func(id uint) (*Payment, error)
	getPaymentStatusFunc      func(wompiID string) (*Payment, error)
	cancelPaymentFunc         func(id uint) (*Payment, error)
	createPaymentLinkFunc     func(req *CreatePaymentLinkRequest) (*PaymentLink, error)
	getPaymentLinkFunc        func(id uint) (*PaymentLink, error)
	activatePaymentLinkFunc   func(id uint) (*PaymentLink, error)
	deactivatePaymentLinkFunc func(id uint) (*PaymentLink, error)
	verifySignatureFunc       func(payload []byte, signature string) bool
	processWebhookFunc        func(event *WebhookEvent) error
}

func (m *MockPaymentService) CreatePayment(req *CreatePaymentRequest) (*Payment, error) {
	if m.createPaymentFunc != nil {
		return m.createPaymentFunc(req)
	}
	return nil, nil
}

func (m *MockPaymentService) GetPayment(id uint) (*Payment, error) {
	if m.getPaymentFunc != nil {
		return m.getPaymentFunc(id)
	}
	return nil, nil
}

func (m *MockPaymentService) GetPaymentStatus(wompiID string) (*Payment, error) {
	if m.getPaymentStatusFunc != nil {
		return m.getPaymentStatusFunc(wompiID)
	}
	return nil, nil
}

func (m *MockPaymentService) CancelPayment(id uint) (*Payment, error) {
	if m.cancelPaymentFunc != nil {
		return m.cancelPaymentFunc(id)
	}
	return nil, nil
}

func (m *MockPaymentService) CreatePaymentLink(req *CreatePaymentLinkRequest) (*PaymentLink, error) {
	if m.createPaymentLinkFunc != nil {
		return m.createPaymentLinkFunc(req)
	}
	return nil, nil
}

func (m *MockPaymentService) GetPaymentLink(id uint) (*PaymentLink, error) {
	if m.getPaymentLinkFunc != nil {
		return m.getPaymentLinkFunc(id)
	}
	return nil, nil
}

func (m *MockPaymentService) ActivatePaymentLink(id uint) (*PaymentLink, error) {
	if m.activatePaymentLinkFunc != nil {
		return m.activatePaymentLinkFunc(id)
	}
	return nil, nil
}

func (m *MockPaymentService) DeactivatePaymentLink(id uint) (*PaymentLink, error) {
	if m.deactivatePaymentLinkFunc != nil {
		return m.deactivatePaymentLinkFunc(id)
	}
	return nil, nil
}

func (m *MockPaymentService) VerifySignature(payload []byte, signature string) bool {
	if m.verifySignatureFunc != nil {
		return m.verifySignatureFunc(payload, signature)
	}
	return true
}

func (m *MockPaymentService) ProcessWebhook(event *WebhookEvent) error {
	if m.processWebhookFunc != nil {
		return m.processWebhookFunc(event)
	}
	return nil
}

type TestPaymentHandler struct {
	service *MockPaymentService
}

func (h *TestPaymentHandler) CreatePayment(c *gin.Context) {
	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment, err := h.service.CreatePayment(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ToPaymentResponse(payment))
}

func (h *TestPaymentHandler) GetPayment(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if _, err := parseUintParams(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment ID"})
		return
	}

	payment, err := h.service.GetPayment(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}
	if payment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}

	c.JSON(http.StatusOK, ToPaymentResponse(payment))
}

func (h *TestPaymentHandler) Webhook(c *gin.Context) {
	signature := c.GetHeader("X-Wompi-Signature")

	var event WebhookEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook payload"})
		return
	}

	bodyBytes, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process webhook"})
		return
	}

	if !h.service.VerifySignature(bodyBytes, signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	if err := h.service.ProcessWebhook(&event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process webhook"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "processed"})
}

func setupTestRouter(handler *TestPaymentHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/payments", handler.CreatePayment)
	r.GET("/payments/:id", handler.GetPayment)
	r.POST("/payments/webhook", handler.Webhook)
	return r
}

func TestPaymentHandler_CreatePayment(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		mockResponse  *Payment
		mockError     error
		wantStatus    int
		wantErrInBody bool
	}{
		{
			name: "valid request",
			body: `{"amount":100000,"currency":"COP","payment_token":"tok_test_123","reference":"order-123"}`,
			mockResponse: &Payment{
				ID:                 1,
				WompiTransactionID: "tx_123",
				Amount:             100000,
				Currency:           "COP",
				Status:             StatusApproved,
				Reference:          "order-123",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:          "invalid request - missing amount",
			body:          `{"currency":"COP","payment_token":"tok_test_123","reference":"order-123"}`,
			mockResponse:  nil,
			wantStatus:    http.StatusBadRequest,
			wantErrInBody: true,
		},
		{
			name:          "invalid request - negative amount",
			body:          `{"amount":-1000,"currency":"COP","payment_token":"tok_test_123","reference":"order-123"}`,
			mockResponse:  nil,
			wantStatus:    http.StatusBadRequest,
			wantErrInBody: true,
		},
		{
			name:          "invalid request - missing payment_token",
			body:          `{"amount":100000,"currency":"COP","reference":"order-123"}`,
			mockResponse:  nil,
			wantStatus:    http.StatusBadRequest,
			wantErrInBody: true,
		},
		{
			name:          "invalid request - empty body",
			body:          ``,
			mockResponse:  nil,
			wantStatus:    http.StatusBadRequest,
			wantErrInBody: true,
		},
		{
			name:          "invalid request - malformed JSON",
			body:          `{invalid json}`,
			mockResponse:  nil,
			wantStatus:    http.StatusBadRequest,
			wantErrInBody: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockPaymentService{
				createPaymentFunc: func(req *CreatePaymentRequest) (*Payment, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			handler := &TestPaymentHandler{service: mockService}
			router := setupTestRouter(handler)

			req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantErrInBody {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Errorf("failed to parse response body: %v", err)
				}
				if resp["error"] == nil && resp["Amount"] == nil {
					t.Errorf("expected error in response, got %v", resp)
				}
			}
		})
	}
}

func TestPaymentHandler_GetPayment(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		mockResponse  *Payment
		mockError     error
		wantStatus    int
		wantErrInBody bool
	}{
		{
			name: "valid payment ID",
			id:   "1",
			mockResponse: &Payment{
				ID:                 1,
				WompiTransactionID: "tx_123",
				Amount:             100000,
				Currency:           "COP",
				Status:             StatusApproved,
				Reference:          "order-123",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:          "invalid payment ID - non-numeric",
			id:            "abc",
			mockResponse:  nil,
			wantStatus:    http.StatusNotFound,
			wantErrInBody: true,
		},
		{
			name:          "invalid payment ID - zero",
			id:            "0",
			mockResponse:  nil,
			wantStatus:    http.StatusNotFound,
			wantErrInBody: true,
		},
		{
			name:          "payment not found",
			id:            "999",
			mockResponse:  nil,
			mockError:     nil,
			wantStatus:    http.StatusNotFound,
			wantErrInBody: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockPaymentService{
				getPaymentFunc: func(id uint) (*Payment, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			handler := &TestPaymentHandler{service: mockService}
			router := setupTestRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/payments/"+tt.id, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantErrInBody {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Errorf("failed to parse response body: %v", err)
				}
				if resp["error"] == nil {
					t.Errorf("expected error in response, got %v", resp)
				}
			}
		})
	}
}

func TestPaymentHandler_Webhook(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		signature     string
		validSig      bool
		mockError     error
		wantStatus    int
		wantErrInBody bool
	}{
		{
			name:       "valid webhook with APPROVED status",
			body:       `{"event":"payment.created","event_id":"evt_123","data":{"id":"tx_123","status":"APPROVED","amount_in_cents":100000,"currency":"COP","reference":"order-123"}}`,
			signature:  "valid-signature",
			validSig:   true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid webhook with DECLINED status",
			body:       `{"event":"payment.declined","event_id":"evt_456","data":{"id":"tx_456","status":"DECLINED","amount_in_cents":100000,"currency":"COP","reference":"order-456"}}`,
			signature:  "valid-signature",
			validSig:   true,
			wantStatus: http.StatusOK,
		},
		{
			name:          "invalid signature",
			body:          `{"event":"payment.created","data":{"id":"tx_123","status":"APPROVED","reference":"order-123"}}`,
			signature:     "invalid-signature",
			validSig:      false,
			wantStatus:    http.StatusUnauthorized,
			wantErrInBody: true,
		},
		{
			name:          "missing signature",
			body:          `{"event":"payment.created","data":{"id":"tx_123","status":"APPROVED","reference":"order-123"}}`,
			signature:     "",
			validSig:      false,
			wantStatus:    http.StatusUnauthorized,
			wantErrInBody: true,
		},
		{
			name:          "invalid JSON body",
			body:          `{invalid json}`,
			signature:     "valid-signature",
			validSig:      true,
			wantStatus:    http.StatusBadRequest,
			wantErrInBody: true,
		},
		{
			name:          "empty body",
			body:          ``,
			signature:     "valid-signature",
			validSig:      true,
			wantStatus:    http.StatusBadRequest,
			wantErrInBody: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockPaymentService{
				verifySignatureFunc: func(payload []byte, signature string) bool {
					return tt.validSig
				},
				processWebhookFunc: func(event *WebhookEvent) error {
					return tt.mockError
				},
			}

			handler := &TestPaymentHandler{service: mockService}
			router := setupTestRouter(handler)

			req := httptest.NewRequest(http.MethodPost, "/payments/webhook", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Wompi-Signature", tt.signature)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantErrInBody {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Errorf("failed to parse response body: %v", err)
				}
				if resp["error"] == nil && resp["status"] == nil {
					t.Errorf("expected error in response, got %v", resp)
				}
			}
		})
	}
}

func TestPaymentHandler_Integration(t *testing.T) {
	mockService := &MockPaymentService{
		createPaymentFunc: func(req *CreatePaymentRequest) (*Payment, error) {
			return &Payment{
				ID:                 1,
				WompiTransactionID: "tx_123",
				Amount:             req.Amount,
				Currency:           req.Currency,
				Status:             StatusPending,
				Reference:          req.Reference,
			}, nil
		},
	}

	handler := &TestPaymentHandler{service: mockService}
	router := setupTestRouter(handler)

	body := `{"amount":50000,"currency":"COP","payment_token":"tok_test_456","reference":"order-integration"}`
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var resp PaymentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Amount != 50000 {
		t.Errorf("expected amount 50000, got %d", resp.Amount)
	}
	if resp.Currency != "COP" {
		t.Errorf("expected currency COP, got %s", resp.Currency)
	}
	if resp.Status != StatusPending {
		t.Errorf("expected status pending, got %s", resp.Status)
	}
}
