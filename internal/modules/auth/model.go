package auth

import (
	"time"
)

type RefreshToken struct {
	ID        uint      `gorm:"primaryKey"`
	Token     string    `gorm:"uniqueIndex;not null"`
	UserID    uint      `gorm:"not null;index"`
	ExpiresAt time.Time `gorm:"not null"`
	Revoked   bool      `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
