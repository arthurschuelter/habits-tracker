package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"unicode"
)

var ErrEmailTaken = errors.New("another account already uses that email")

const (
	maxUsernameBytes = 256 // matches the limit Login accepts.
	maxEmailBytes    = 254
)

// ValidationError reports a registration field that cannot be accepted.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// Registration holds the details for a new account.
type Registration struct {
	Username string
	Email    string
	Password string
}

// NormalizeRegistration trims the username and email, lowercases the email,
// and checks each field in form order. The password is used exactly as given.
// Clients may call it for early feedback; Register remains authoritative.
func NormalizeRegistration(r Registration) (Registration, error) {
	r.Username = strings.TrimSpace(r.Username)
	switch {
	case r.Username == "":
		return r, &ValidationError{Field: "username", Message: "is required"}
	case len(r.Username) > maxUsernameBytes:
		return r, &ValidationError{Field: "username", Message: "must be at most 256 bytes"}
	case strings.IndexFunc(r.Username, unicode.IsControl) >= 0:
		return r, &ValidationError{Field: "username", Message: "must not contain control characters"}
	}

	email, err := normalizeEmail(r.Email)
	r.Email = email
	if err != nil {
		return r, err
	}
	return r, ValidatePassword([]byte(r.Password))
}

// normalizeEmail trims and lowercases an address and checks its format.
func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	address, err := mail.ParseAddress(email)
	if email == "" {
		return email, &ValidationError{Field: "email", Message: "is required"}
	}
	// Reject display names and other forms that ParseAddress rewrites.
	if err != nil || address.Address != email || len(email) > maxEmailBytes {
		return email, &ValidationError{Field: "email", Message: "must be a valid email address"}
	}
	return email, nil
}

// Register creates an account that can sign in with Login.
func (s *Service) Register(ctx context.Context, r Registration) error {
	r, err := NormalizeRegistration(r)
	if err != nil {
		return err
	}
	// Hashing uses 64 MiB, so share the login slots that bound concurrent work.
	select {
	case s.loginSlots <- struct{}{}:
		defer func() { <-s.loginSlots }()
	default:
		return ErrBusy
	}
	hash, err := HashPassword([]byte(r.Password))
	if err != nil {
		return err
	}
	return s.repo.createUser(ctx, r.Username, r.Email, hash)
}
