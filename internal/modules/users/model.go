package users

import (
	"time"

	"bey/internal/shared/uuidutil"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type User struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primarykey" json:"id"`
	Email               string         `gorm:"uniqueIndex;size:255" json:"email"`
	Password            string         `gorm:"size:255" json:"-"`
	FirstName           string         `gorm:"size:100" json:"first_name"`
	LastName            string         `gorm:"size:100" json:"last_name"`
	Phone               string         `gorm:"size:20" json:"phone,omitempty"`
	Role                string         `gorm:"size:50;default:customer" json:"role"`
	Active              bool           `gorm:"default:true" json:"active"`
	EmailVerified       bool           `gorm:"default:false" json:"email_verified"`
	VerificationToken   string         `gorm:"size:64" json:"-"`
	VerificationExpires *time.Time     `gorm:"index" json:"-"`
	ResetToken          string         `gorm:"size:64" json:"-"`
	ResetExpires        *time.Time     `gorm:"index" json:"-"`
	TwoFASecret         string         `gorm:"size:255" json:"-"`
	TwoFAEnabled        bool           `gorm:"default:false" json:"two_fa_enabled"`
	TwoFABackupCodes    string         `gorm:"type:text" json:"-"`
	OAuthProvider       string         `gorm:"size:50" json:"oauth_provider,omitempty"`
	OAuthProviderID     string         `gorm:"size:255" json:"oauth_provider_id,omitempty"`
	AvatarURL           string         `gorm:"size:500" json:"avatar_url,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuidutil.GenerateV7()
	}
	return nil
}

type UpdateUserRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Phone     *string `json:"phone"`
	Active    *bool   `json:"active"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Phone     string    `json:"phone,omitempty"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
