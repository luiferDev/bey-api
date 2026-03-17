package payments

import (
	"log"
	"net/http"

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
	id, err := parseUint(c.Param("id"))
	if err != nil {
		h.responseHandler.ValidationError(c, "invalid payment ID")
		return
	}

	payment, err := h.service.CancelPayment(id)
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

	h.responseHandler.Success(c, ToPaymentLinkResponse(link))
}

func (h *PaymentHandler) ActivatePaymentLink(c *gin.Context) {
	id, err := parseUint(c.Param("id"))
	if err != nil {
		h.responseHandler.ValidationError(c, "invalid payment link ID")
		return
	}

	link, err := h.service.ActivatePaymentLink(id)
	if err != nil {
		h.responseHandler.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	h.responseHandler.Success(c, ToPaymentLinkResponse(link))
}

func (h *PaymentHandler) DeactivatePaymentLink(c *gin.Context) {
	id, err := parseUint(c.Param("id"))
	if err != nil {
		h.responseHandler.ValidationError(c, "invalid payment link ID")
		return
	}

	link, err := h.service.DeactivatePaymentLink(id)
	if err != nil {
		h.responseHandler.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	h.responseHandler.Success(c, ToPaymentLinkResponse(link))
}

func (h *PaymentHandler) Webhook(c *gin.Context) {
	signature := c.GetHeader("X-Wompi-Signature")

	var event WebhookEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		log.Printf("ERROR: Failed to parse webhook: %v", err)
		h.responseHandler.Error(c, http.StatusBadRequest, "invalid webhook payload")
		return
	}

	// Read body for signature verification
	bodyBytes, err := c.GetRawData()
	if err != nil {
		log.Printf("ERROR: Failed to read webhook body: %v", err)
		h.responseHandler.Error(c, http.StatusInternalServerError, "failed to process webhook")
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
