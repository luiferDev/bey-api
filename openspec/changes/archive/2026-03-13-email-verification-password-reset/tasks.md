## Tasks: Email Verification & Password Reset

### Phase 1: Infrastructure (Config & Dependencies)
- [x] 1.1 Add go-mail to go.mod
- [x] 1.2 Add email config to config.yaml
- [x] 1.3 Add EmailConfig struct to internal/config/config.go
- [x] 1.4 Add user model fields: email_verified, verification_token, reset_token, reset_expires_at
- [x] 1.5 Run database migration to add new columns

### Phase 2: Email Service Core
- [x] 2.1 Create internal/modules/email/service.go - email service with go-mail client
- [x] 2.2 Create internal/modules/email/templates/verification.html
- [x] 2.3 Create internal/modules/email/templates/password_reset.html
- [x] 2.4 Create internal/modules/email/templates/password_reset_plain.txt
- [x] 2.5 Implement token generation (32-byte crypto random, hex encoded)
- [x] 2.6 Implement token hashing (SHA-256, hex encoded)
- [x] 2.7 Implement token validation
- [x] 2.8 Implement SendVerificationEmail function
- [x] 2.9 Implement SendPasswordResetEmail function

### Phase 3: Auth Integration
- [x] 3.1 Add POST /auth/verify-email endpoint
- [x] 3.2 Add POST /auth/forgot-password endpoint
- [x] 3.3 Add POST /auth/reset-password endpoint
- [x] 3.4 Update user registration to generate verification token
- [x] 3.5 Wire routes in cmd/api/main.go
- [x] 3.6 Add request DTOs for forgot/reset password

### Phase 4: Testing
- [x] 4.1 Unit tests for token generation
- [x] 4.2 Unit tests for token hashing
- [x] 4.3 Unit tests for token validation
- [x] 4.4 Unit tests for email service
- [x] 4.5 Integration tests for /auth/verify-email
- [x] 4.6 Integration tests for /auth/forgot-password
- [x] 4.7 Integration tests for /auth/reset-password

**Total: 22/22 tasks complete**
