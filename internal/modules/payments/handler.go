package payments

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"bey/internal/shared/response"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	service         *PaymentService
	responseHandler *response.ResponseHandler
}

func NewPaymentHandler(service *PaymentService) *PaymentHandler {
	return &PaymentHandler{
		service:         service,
		responseHandler: response.NewResponseHandler(),
	}
}

// CreatePayment godoc
// @Summary Create a new payment
// @Description Creates a new payment transaction through Wompi gateway for the authenticated user
// @Tags Payments
// @Accept json
// @Produce json
// @Param request body CreatePaymentRequest true "Payment creation request"
// @Success 201 {object} response.ApiResponse{data=PaymentResponse} "Payment created successfully"
// @Failure 400 {object} response.ApiResponse "Bad request - invalid payment data"
// @Failure 401 {object} response.ApiResponse "Unauthorized - invalid or missing token"
// @Failure 500 {object} response.ApiResponse "Internal server error - payment processing failed"
// @Security BearerAuth
// @Router /api/v1/payments [post]
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responseHandler.ValidationError(c, err.Error())
		return
	}

	payment, err := h.service.CreatePayment(&req)
	if err != nil {
		log.Printf("ERROR: CreatePayment failed: %v", err)
		h.responseHandler.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	h.responseHandler.Created(c, ToPaymentResponse(payment))
}

// GetPayment godoc
// @Summary Get payment details
// @Description Retrieves details of a specific payment by ID
// @Tags Payments
// @Accept json
// @Produce json
// @Param id path int true "Payment ID"
// @Success 200 {object} response.ApiResponse{data=PaymentResponse} "Payment details retrieved successfully"
// @Failure 400 {object} response.ApiResponse "Bad request - invalid payment ID"
// @Failure 401 {object} response.ApiResponse "Unauthorized - invalid or missing token"
// @Failure 404 {object} response.ApiResponse "Payment not found"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/payments/{id} [get]
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	id, err := parseUint(c.Param("id"))
	if err != nil {
		h.responseHandler.ValidationError(c, "invalid payment ID")
		return
	}

	payment, err := h.service.GetPayment(id)
	if err != nil {
		h.responseHandler.NotFound(c, "payment not found")
		return
	}

	// Check ownership via order
	userID := c.GetUint("user_id")
	userRole := c.GetString("user_role")
	if userRole != "admin" {
		order, err := h.service.GetOrderByPaymentID(payment.OrderID)
		if err != nil || order == nil || order.UserID != userID {
			h.responseHandler.Error(c, http.StatusForbidden, "you don't have access to this payment")
			return
		}
	}

	h.responseHandler.Success(c, ToPaymentResponse(payment))
}

// GetPaymentStatus godoc
// @Summary Get payment status by Wompi ID
// @Description Retrieves payment status using Wompi transaction ID
// @Tags Payments
// @Accept json
// @Produce json
// @Param wompi_id path string true "Wompi transaction ID"
// @Success 200 {object} response.ApiResponse{data=PaymentResponse} "Payment status retrieved successfully"
// @Failure 400 {object} response.ApiResponse "Bad request - invalid Wompi ID or payment error"
// @Failure 401 {object} response.ApiResponse "Unauthorized - invalid or missing token"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/payments/wompi/{wompi_id}/status [get]
func (h *PaymentHandler) GetPaymentStatus(c *gin.Context) {
	wompiID := c.Param("wompi_id")
	if wompiID == "" {
		h.responseHandler.ValidationError(c, "wompi transaction ID required")
		return
	}

	payment, err := h.service.GetPaymentStatus(wompiID)
	if err != nil {
		h.responseHandler.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	h.responseHandler.Success(c, ToPaymentResponse(payment))
}

// VoidPayment godoc
// @Summary Void/cancel a payment
// @Description Cancels an existing payment transaction
// @Tags Payments
// @Accept json
// @Produce json
// @Param id path int true "Payment ID to cancel"
// @Success 200 {object} response.ApiResponse{data=PaymentResponse} "Payment cancelled successfully"
// @Failure 400 {object} response.ApiResponse "Bad request - payment cannot be cancelled"
// @Failure 401 {object} response.ApiResponse "Unauthorized - invalid or missing token"
// @Failure 404 {object} response.ApiResponse "Payment not found"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/payments/{id}/void [post]
func (h *PaymentHandler) VoidPayment(c *gin.Context) {
	id, err := parseUint(c.Param("id"))
	if err != nil {
		h.responseHandler.ValidationError(c, "invalid payment ID")
		return
	}

	payment, err := h.service.GetPayment(id)
	if err != nil {
		h.responseHandler.NotFound(c, "payment not found")
		return
	}

	// Check ownership via order
	userID := c.GetUint("user_id")
	userRole := c.GetString("user_role")
	if userRole != "admin" {
		order, err := h.service.GetOrderByPaymentID(payment.OrderID)
		if err != nil || order == nil || order.UserID != userID {
			h.responseHandler.Error(c, http.StatusForbidden, "you don't have access to this payment")
			return
		}
	}

	payment, err = h.service.CancelPayment(id)
	if err != nil {
		log.Printf("ERROR: VoidPayment failed: %v", err)
		h.responseHandler.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	h.responseHandler.Success(c, ToPaymentResponse(payment))
}

// CreatePaymentLink godoc
// @Summary Create a payment link
// @Description Creates a new payment link through Wompi gateway that can be shared with customers
// @Tags Payments
// @Accept json
// @Produce json
// @Param request body CreatePaymentLinkRequest true "Payment link creation request"
// @Success 201 {object} response.ApiResponse{data=PaymentLinkResponse} "Payment link created successfully"
// @Failure 400 {object} response.ApiResponse "Bad request - invalid payment link data"
// @Failure 401 {object} response.ApiResponse "Unauthorized - invalid or missing token"
// @Failure 500 {object} response.ApiResponse "Internal server error - payment link creation failed"
// @Security BearerAuth
// @Router /api/v1/payments/links [post]
func (h *PaymentHandler) CreatePaymentLink(c *gin.Context) {
	var req CreatePaymentLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responseHandler.ValidationError(c, err.Error())
		return
	}

	link, err := h.service.CreatePaymentLink(&req)
	if err != nil {
		log.Printf("ERROR: CreatePaymentLink failed: %v", err)
		h.responseHandler.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	h.responseHandler.Created(c, ToPaymentLinkResponse(link))
}

// GetPaymentLink godoc
// @Summary Get payment link details
// @Description Retrieves details of a specific payment link by ID
// @Tags Payments
// @Accept json
// @Produce json
// @Param id path int true "Payment link ID"
// @Success 200 {object} response.ApiResponse{data=PaymentLinkResponse} "Payment link retrieved successfully"
// @Failure 400 {object} response.ApiResponse "Bad request - invalid payment link ID"
// @Failure 401 {object} response.ApiResponse "Unauthorized - invalid or missing token"
// @Failure 404 {object} response.ApiResponse "Payment link not found"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/payments/links/{id} [get]
func (h *PaymentHandler) GetPaymentLink(c *gin.Context) {
	id, err := parseUint(c.Param("id"))
	if err != nil {
		h.responseHandler.ValidationError(c, "invalid payment link ID")
		return
	}

	link, err := h.service.GetPaymentLink(id)
	if err != nil {
		h.responseHandler.NotFound(c, "payment link not found")
		return
	}

	// Check ownership via order
	userID := c.GetUint("user_id")
	userRole := c.GetString("user_role")
	if userRole != "admin" {
		order, err := h.service.GetOrderByPaymentID(link.OrderID)
		if err != nil || order == nil || order.UserID != userID {
			h.responseHandler.Error(c, http.StatusForbidden, "you don't have access to this payment link")
			return
		}
	}

	h.responseHandler.Success(c, ToPaymentLinkResponse(link))
}

// ActivatePaymentLink godoc
// @Summary Activate a payment link
// @Description Activates a previously deactivated payment link
// @Tags Payments
// @Accept json
// @Produce json
// @Param id path int true "Payment link ID to activate"
// @Success 200 {object} response.ApiResponse{data=PaymentLinkResponse} "Payment link activated successfully"
// @Failure 400 {object} response.ApiResponse "Bad request - payment link cannot be activated"
// @Failure 401 {object} response.ApiResponse "Unauthorized - invalid or missing token"
// @Failure 404 {object} response.ApiResponse "Payment link not found"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/payments/links/{id}/activate [patch]
func (h *PaymentHandler) ActivatePaymentLink(c *gin.Context) {
	id, err := parseUint(c.Param("id"))
	if err != nil {
		h.responseHandler.ValidationError(c, "invalid payment link ID")
		return
	}

	link, err := h.service.GetPaymentLink(id)
	if err != nil {
		h.responseHandler.NotFound(c, "payment link not found")
		return
	}

	// Check ownership via order
	userID := c.GetUint("user_id")
	userRole := c.GetString("user_role")
	if userRole != "admin" {
		order, err := h.service.GetOrderByPaymentID(link.OrderID)
		if err != nil || order == nil || order.UserID != userID {
			h.responseHandler.Error(c, http.StatusForbidden, "you don't have access to this payment link")
			return
		}
	}

	link, err = h.service.ActivatePaymentLink(id)
	if err != nil {
		h.responseHandler.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	h.responseHandler.Success(c, ToPaymentLinkResponse(link))
}

// DeactivatePaymentLink godoc
// @Summary Deactivate a payment link
// @Description Deactivates an active payment link to prevent further payments
// @Tags Payments
// @Accept json
// @Produce json
// @Param id path int true "Payment link ID to deactivate"
// @Success 200 {object} response.ApiResponse{data=PaymentLinkResponse} "Payment link deactivated successfully"
// @Failure 400 {object} response.ApiResponse "Bad request - payment link cannot be deactivated"
// @Failure 401 {object} response.ApiResponse "Unauthorized - invalid or missing token"
// @Failure 404 {object} response.ApiResponse "Payment link not found"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/payments/links/{id}/deactivate [patch]
func (h *PaymentHandler) DeactivatePaymentLink(c *gin.Context) {
	id, err := parseUint(c.Param("id"))
	if err != nil {
		h.responseHandler.ValidationError(c, "invalid payment link ID")
		return
	}

	link, err := h.service.GetPaymentLink(id)
	if err != nil {
		h.responseHandler.NotFound(c, "payment link not found")
		return
	}

	// Check ownership via order
	userID := c.GetUint("user_id")
	userRole := c.GetString("user_role")
	if userRole != "admin" {
		order, err := h.service.GetOrderByPaymentID(link.OrderID)
		if err != nil || order == nil || order.UserID != userID {
			h.responseHandler.Error(c, http.StatusForbidden, "you don't have access to this payment link")
			return
		}
	}

	link, err = h.service.DeactivatePaymentLink(id)
	if err != nil {
		h.responseHandler.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	h.responseHandler.Success(c, ToPaymentLinkResponse(link))
}

// Webhook godoc
// @Summary Handle Wompi webhook
// @Description Receives and processes webhook events from Wompi payment gateway
// @Tags Payments
// @Accept json
// @Produce json
// @Param request body WebhookEvent true "Webhook event payload"
// @Success 200 {object} response.ApiResponse "Webhook processed successfully"
// @Failure 400 {object} response.ApiResponse "Bad request - invalid payload"
// @Failure 401 {object} response.ApiResponse "Unauthorized - invalid webhook signature"
// @Failure 500 {object} response.ApiResponse "Internal server error - webhook processing failed"
// @Router /api/v1/payments/webhook [post]
func (h *PaymentHandler) Webhook(c *gin.Context) {
	contentType := c.GetHeader("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "application/json") {
		h.responseHandler.Error(c, http.StatusUnsupportedMediaType, "unsupported media type")
		return
	}

	signature := c.GetHeader("X-Wompi-Signature")

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read webhook body: %v", err)
		h.responseHandler.Error(c, http.StatusBadRequest, "failed to read request body")
		return
	}

	var event WebhookEvent
	if err := json.Unmarshal(bodyBytes, &event); err != nil {
		log.Printf("ERROR: Failed to parse webhook: %v", err)
		h.responseHandler.Error(c, http.StatusBadRequest, "invalid webhook payload")
		return
	}

	if !h.service.VerifySignature(bodyBytes, signature) {
		log.Printf("WARNING: Invalid webhook signature")
		h.responseHandler.Error(c, http.StatusUnauthorized, "invalid signature")
		return
	}

	if err := h.service.ProcessWebhook(&event); err != nil {
		log.Printf("ERROR: Webhook processing failed: %v", err)
		h.responseHandler.Error(c, http.StatusInternalServerError, "failed to process webhook")
		return
	}

	h.responseHandler.Success(c, gin.H{"status": "processed"})
}

func parseUint(s string) (uint, error) {
	var id uint
	if _, err := parseUintParams(s, &id); err != nil {
		return 0, err
	}
	return id, nil
}

func parseUintParams(s string, v *uint) (bool, error) {
	var u uint
	if _, err := parseUintBytes([]byte(s), &u); err != nil {
		return false, err
	}
	*v = u
	return true, nil
}

func parseUintBytes(s []byte, v *uint) (bool, error) {
	var n uint = 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return false, nil
		}
		n = n*10 + uint(c-'0')
	}
	*v = n
	return true, nil
}

func ToPaymentResponse(p *Payment) PaymentResponse {
	return PaymentResponse{
		ID:                 p.ID,
		OrderID:            p.OrderID,
		WompiTransactionID: p.WompiTransactionID,
		Amount:             p.Amount,
		Currency:           p.Currency,
		Status:             p.Status,
		PaymentMethod:      p.PaymentMethod,
		RedirectURL:        p.RedirectURL,
		Reference:          p.Reference,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}

func ToPaymentLinkResponse(l *PaymentLink) PaymentLinkResponse {
	return PaymentLinkResponse{
		ID:          l.ID,
		OrderID:     l.OrderID,
		WompiLinkID: l.WompiLinkID,
		URL:         l.URL,
		Amount:      l.Amount,
		Currency:    l.Currency,
		Description: l.Description,
		Status:      l.Status,
		SingleUse:   l.SingleUse,
		ExpiresAt:   l.ExpiresAt,
		RedirectURL: l.RedirectURL,
		Reference:   l.Reference,
		CreatedAt:   l.CreatedAt,
		UpdatedAt:   l.UpdatedAt,
	}
}
