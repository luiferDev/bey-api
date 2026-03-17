package payments

import "time"

type CreatePaymentRequest struct {
	Amount       int64  `json:"amount" binding:"required,gt=0"`   // Amount in cents
	Currency     string `json:"currency" binding:"required"`      // e.g., "COP"
	PaymentToken string `json:"payment_token" binding:"required"` // Wompi payment token
	RedirectURL  string `json:"redirect_url"`                     // Optional redirect after payment
	Reference    string `json:"reference" binding:"required"`     // Order reference
}

type CreatePaymentLinkRequest struct {
	AmountInCents int64      `json:"amount_in_cents" binding:"required,gt=0"`
	Description   string     `json:"description" binding:"required"`
	Currency      string     `json:"currency" binding:"required"` // default: "COP"
	SingleUse     bool       `json:"single_use"`
	ExpiresAt     *time.Time `json:"expires_at"`
	RedirectURL   string     `json:"redirect_url"`
	Reference     string     `json:"reference" binding:"required"`
	OrderID       uint       `json:"order_id"`
}

type PaymentResponse struct {
	ID                 uint      `json:"id"`
	OrderID            uint      `json:"order_id"`
	WompiTransactionID string    `json:"wompi_transaction_id"`
	Amount             int64     `json:"amount"`
	Currency           string    `json:"currency"`
	Status             string    `json:"status"`
	PaymentMethod      string    `json:"payment_method,omitempty"`
	RedirectURL        string    `json:"redirect_url,omitempty"`
	Reference          string    `json:"reference"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type PaymentLinkResponse struct {
	ID          uint       `json:"id"`
	OrderID     uint       `json:"order_id"`
	WompiLinkID string     `json:"wompi_link_id"`
	URL         string     `json:"url"`
	Amount      int64      `json:"amount"`
	Currency    string     `json:"currency"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	SingleUse   bool       `json:"single_use"`
	ExpiresAt   *time.Time `json:"expires_at"`
	RedirectURL string     `json:"redirect_url,omitempty"`
	Reference   string     `json:"reference"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type WebhookEvent struct {
	Event       string      `json:"event"`
	EventID     string      `json:"event_id"`
	Timestamp   time.Time   `json:"timestamp"`
	Signature   string      `json:"signature"`
	Transaction Transaction `json:"data"` // Wompi sends data directly
}

type Transaction struct {
	ID            string    `json:"id"`
	Status        string    `json:"status"`
	StatusDetail  string    `json:"status_detail"`
	AmountInCents int64     `json:"amount_in_cents"`
	Currency      string    `json:"currency"`
	Reference     string    `json:"reference"`
	PaymentMethod string    `json:"payment_method"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	ErrorCode     *string   `json:"error_code"`
	ErrorMessage  *string   `json:"error_message"`
}

type PaymentLinkData struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	AmountInCents int64     `json:"amount_in_cents"`
	Currency      string    `json:"currency"`
	URL           string    `json:"url"`
	RedirectURL   *string   `json:"redirect_url"`
	Status        string    `json:"status"`
	SingleUse     bool      `json:"single_use"`
	ExpiresAt     *string   `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type WompiTransactionResponse struct {
	Transaction Transaction `json:"data"`
}

type WompiPaymentLinkResponse struct {
	PaymentLink PaymentLinkData `json:"data"`
}

type WompiError struct {
	Error   string `json:"error"`
	Message string `json:"error_description"`
}
