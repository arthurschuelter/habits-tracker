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

	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	address, err := mail.ParseAddress(r.Email)
	if r.Email == "" {
		return r, &ValidationError{Field: "email", Message: "is required"}
	}
	// Reject display names and other forms that ParseAddress rewrites.
	if err != nil || address.Address != r.Email || len(r.Email) > maxEmailBytes {
		return r, &ValidationError{Field: "email", Message: "must be a valid email address"}
	}
	return r, ValidatePassword([]byte(r.Password))
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
