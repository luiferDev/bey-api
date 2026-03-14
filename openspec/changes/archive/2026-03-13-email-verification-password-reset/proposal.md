# Proposal: Email Verification and Password Reset

## Intent

Address critical authentication gaps in Bey API by implementing:
1. **Email verification** - Ensure user emails are valid and belong to the user
2. **Password reset** - Allow users to recover account access when password is forgotten

## Scope

### In Scope
- Email verification flow with secure token generation
- Password reset flow with secure token generation
- Token hashing for security (SHA-256)
- SMTP email sending via go-mail library
- Expiration for reset tokens (1 hour)

### Out of Scope
- OAuth2/Social login
- Two-factor authentication (2FA)
- Rate limiting

## Approach

Implement email verification and password reset as new functionality in the auth module, with a separate email module for reusable email sending capabilities. Use crypto/rand for secure token generation and SHA-256 for token hashing.
