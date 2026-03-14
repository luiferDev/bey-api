# Technical Design: Email Verification & Password Reset

## Architecture

### New Modules

1. **Email Module** (`internal/modules/email/`)
   - `service.go` - Email sending service using go-mail
   - `token.go` - Token generation and hashing utilities
   - `templates/` - Email templates (HTML and plain text)

2. **Auth Module Updates** (`internal/modules/auth/`)
   - New endpoints: `/auth/verify-email`, `/auth/forgot-password`, `/auth/reset-password`
   - Service layer updates for verification and reset logic

### User Model Updates

New fields added to User model:
- `EmailVerified` (bool) - Whether email has been verified
- `VerificationToken` (string, nullable) - Token for email verification
- `ResetToken` (string, nullable) - Token for password reset
- `ResetExpiresAt` (*time.Time, nullable) - Expiration for reset token

### Configuration

New email configuration in `config.yaml`:
```yaml
email:
  smtp_host: "smtp.gmail.com"
  smtp_port: 587
  username: "your-email@gmail.com"
  password: "your-app-password"
  from_email: "noreply@beyapi.com"
  from_name: "Bey API"
```

## Security Considerations

1. **Token Security**
   - 32-byte cryptographically secure random tokens
   - SHA-256 hashing before database storage
   - Tokens never logged or exposed in errors

2. **Email Safety**
   - No email existence confirmation in forgot-password response
   - Rate limiting recommended for production

3. **Password Reset**
   - Tokens expire after 1 hour
   - Single-use tokens (invalidated after use)
