package payments

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"

	"bey/internal/shared/response"
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

func (h *PaymentHandler) GetPayment(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.responseHandler.ValidationError(c, "invalid payment ID format")
		return
	}

	payment, err := h.service.GetPayment(id)
	if err != nil {
		h.responseHandler.NotFound(c, "payment not found")
		return
	}

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

func (h *PaymentHandler) VoidPayment(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.responseHandler.ValidationError(c, "invalid payment ID format")
		return
	}

	payment, err := h.service.GetPayment(id)
	if err != nil {
		h.responseHandler.NotFound(c, "payment not found")
		return
	}

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

func (h *PaymentHandler) GetPaymentLink(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.responseHandler.ValidationError(c, "invalid payment link ID format")
		return
	}

	link, err := h.service.GetPaymentLink(id)
	if err != nil {
		h.responseHandler.NotFound(c, "payment link not found")
		return
	}

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

func (h *PaymentHandler) ActivatePaymentLink(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.responseHandler.ValidationError(c, "invalid payment link ID format")
		return
	}

	link, err := h.service.GetPaymentLink(id)
	if err != nil {
		h.responseHandler.NotFound(c, "payment link not found")
		return
	}

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

func (h *PaymentHandler) DeactivatePaymentLink(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		h.responseHandler.ValidationError(c, "invalid payment link ID format")
		return
	}

	link, err := h.service.GetPaymentLink(id)
	if err != nil {
		h.responseHandler.NotFound(c, "payment link not found")
		return
	}

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

func ToPaymentResponse(p *Payment) PaymentResponse {
	return PaymentResponse{
		ID:                 p.ID.String(),
		OrderID:            p.OrderID.String(),
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
		ID:          l.ID.String(),
		OrderID:     l.OrderID.String(),
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
