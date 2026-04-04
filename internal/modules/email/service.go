package email

import (
	"bey/internal/config"
	"bey/internal/modules/users"
	"fmt"
	"strings"
	"time"

	"github.com/wneessen/go-mail"
)

type EmailService struct {
	cfg    *config.Config
	client *mail.Client
}

func NewEmailService(cfg *config.Config) (*EmailService, error) {
	client, err := mail.NewClient(
		cfg.Email.SMTP.Host,
		mail.WithPort(cfg.Email.SMTP.Port),
		mail.WithUsername(cfg.Email.SMTP.Username),
		mail.WithPassword(cfg.Email.SMTP.Password),
		mail.WithTLSConfig(nil),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create email client: %w", err)
	}

	return &EmailService{
		cfg:    cfg,
		client: client,
	}, nil
}

func (s *EmailService) SendVerificationEmail(toEmail, token string) error {
	verifyURL := fmt.Sprintf("http://localhost:3000/verify-email?token=%s", token)

	body := strings.ReplaceAll(VerificationEmailTemplate, "{{.URL}}", verifyURL)

	msg := mail.NewMsg()
	if err := msg.From(s.cfg.Email.FromEmail); err != nil {
		return fmt.Errorf("failed to set from address: %w", err)
	}
	if err := msg.AddTo(toEmail); err != nil {
		return fmt.Errorf("failed to set to address: %w", err)
	}

	msg.Subject("Verify Your Email - Bey API")
	msg.SetBodyString(mail.TypeTextHTML, body)

	if err := s.client.Send(msg); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	return nil
}

func (s *EmailService) SendPasswordResetEmail(toEmail, token string) error {
	resetURL := fmt.Sprintf("http://localhost:3000/reset-password?token=%s", token)

	body := strings.ReplaceAll(PasswordResetEmailTemplate, "{{.URL}}", resetURL)

	msg := mail.NewMsg()
	if err := msg.From(s.cfg.Email.FromEmail); err != nil {
		return fmt.Errorf("failed to set from address: %w", err)
	}
	if err := msg.AddTo(toEmail); err != nil {
		return fmt.Errorf("failed to set to address: %w", err)
	}

	msg.Subject("Password Reset - Bey API")
	msg.SetBodyString(mail.TypeTextHTML, body)

	if err := s.client.Send(msg); err != nil {
		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	return nil
}

func VerifyVerificationToken(user *users.User, token string) bool {
	if user == nil || user.VerificationToken == "" || user.VerificationExpires == nil {
		return false
	}

	if time.Now().After(*user.VerificationExpires) {
		return false
	}

	expectedHash := HashToken(token)
	return user.VerificationToken == expectedHash
}

func VerifyResetToken(user *users.User, token string) bool {
	if user == nil || user.ResetToken == "" || user.ResetExpires == nil {
		return false
	}

	if time.Now().After(*user.ResetExpires) {
		return false
	}

	expectedHash := HashToken(token)
	return user.ResetToken == expectedHash
}
