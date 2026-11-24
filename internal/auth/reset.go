package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"log/slog"
	"strings"
	"time"
)

var ErrResetCode = errors.New("invalid or expired reset code")

const resetLifetime = 15 * time.Minute

// Codes use base32's letters and digits, which are easy to read and type.
var resetEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// ForgotPassword creates a one-time reset code for the account with this
// email and replaces any earlier code. It succeeds whether or not the account
// exists. There is no email delivery yet, so the code is written to the API log.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	email, err := normalizeEmail(email)
	if err != nil {
		return err
	}
	var raw [5]byte
	_, _ = rand.Read(raw[:])
	code := resetEncoding.EncodeToString(raw[:])
	found, err := s.repo.saveResetCode(ctx, email, sha256.Sum256([]byte(code)), resetLifetime)
	if err != nil || !found {
		return err
	}
	slog.Info("password reset code issued", "email", email, "code", code, "expires_in", resetLifetime.String())
	return nil
}

// ResetPassword sets a new password if the code matches the account's
// unexpired code. The code is consumed and the session is revoked.
func (s *Service) ResetPassword(ctx context.Context, email, code, password string) error {
	email, err := normalizeEmail(email)
	if err != nil {
		return err
	}
	code = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
	if code == "" {
		return &ValidationError{Field: "code", Message: "is required"}
	}
	if err := ValidatePassword([]byte(password)); err != nil {
		return err
	}
	// Hashing uses 64 MiB, so share the login slots that bound concurrent work.
	select {
	case s.loginSlots <- struct{}{}:
		defer func() { <-s.loginSlots }()
	default:
		return ErrBusy
	}
	hash, err := HashPassword([]byte(password))
	if err != nil {
		return err
	}
	return s.repo.resetPassword(ctx, email, sha256.Sum256([]byte(code)), hash)
}
