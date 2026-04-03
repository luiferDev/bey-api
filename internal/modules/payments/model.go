package payments

import (
	"time"

	"bey/internal/shared/uuidutil"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type Payment struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	OrderID            uuid.UUID      `gorm:"index" json:"order_id"`
	WompiTransactionID string         `gorm:"size:255;uniqueIndex" json:"wompi_transaction_id"`
	Amount             int64          `gorm:"not null" json:"amount"`
	Currency           string         `gorm:"size:3;default:COP" json:"currency"`
	Status             string         `gorm:"size:50;default:pending" json:"status"`
	PaymentMethod      string         `gorm:"size:50" json:"payment_method"`
	PaymentToken       string         `gorm:"size:255" json:"payment_token,omitempty"`
	RedirectURL        string         `gorm:"size:500" json:"redirect_url,omitempty"`
	Reference          string         `gorm:"size:255;index" json:"reference"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuidutil.GenerateV7()
	}
	return nil
}

type PaymentLink struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	OrderID     uuid.UUID      `gorm:"index" json:"order_id"`
	WompiLinkID string         `gorm:"size:255;uniqueIndex" json:"wompi_link_id"`
	URL         string         `gorm:"size:500;not null" json:"url"`
	Amount      int64          `gorm:"not null" json:"amount"`
	Currency    string         `gorm:"size:3;default:COP" json:"currency"`
	Description string         `gorm:"size:500" json:"description"`
	Status      string         `gorm:"size:50;default:active" json:"status"`
	SingleUse   bool           `gorm:"default:false" json:"single_use"`
	ExpiresAt   *time.Time     `gorm:"index" json:"expires_at"`
	RedirectURL string         `gorm:"size:500" json:"redirect_url,omitempty"`
	Reference   string         `gorm:"size:255;index" json:"reference"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (pl *PaymentLink) BeforeCreate(tx *gorm.DB) error {
	if pl.ID == uuid.Nil {
		pl.ID = uuidutil.GenerateV7()
	}
	return nil
}

const (
	StatusPending       = "pending"
	StatusApproved      = "approved"
	StatusDeclined      = "declined"
	StatusVoided        = "voided"
	StatusCancelled     = "cancelled"
	StatusFailed        = "failed"
	StatusPendingWallet = "pending_wallet"

	StatusActive   = "active"
	StatusInactive = "inactive"
	StatusExpired  = "expired"
	StatusUsed     = "used"

	CurrencyCOP = "COP"
)
