package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

var (
	ErrUnauthenticated = errors.New("authentication required")
	ErrCredentials     = errors.New("invalid username or password")
	ErrBusy            = errors.New("too many login attempts; try again shortly")
	ErrUserNotFound    = errors.New("no user with that email")
	ErrUsernameTaken   = errors.New("another login already uses that username")
)

const sessionLifetime = 24 * time.Hour
const tokenSize = 32

type credentials struct {
	userID       string
	passwordHash string
}

type Service struct {
	// Mailer delivers password reset codes. When nil, codes are written to
	// the log instead, which is only suitable for local development.
	Mailer     Mailer
	repo       *PostgresRepo
	loginSlots chan struct{}
}

func NewService(repo *PostgresRepo) *Service {
	return &Service{repo: repo, loginSlots: make(chan struct{}, 2)}
}

func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" || len(username) > 256 || password == "" || len(password) > maxPasswordBytes {
		return "", ErrCredentials
	}
	select {
	case s.loginSlots <- struct{}{}:
		defer func() { <-s.loginSlots }()
	default:
		return "", ErrBusy
	}
	creds, err := s.repo.credentials(ctx, username)
	if err != nil {
		return "", err
	}
	if !verifyPassword(password, creds.passwordHash) || creds.userID == "" {
		return "", ErrCredentials
	}
	var token [tokenSize]byte
	_, _ = rand.Read(token[:])
	if err := s.repo.saveSession(ctx, creds, sha256.Sum256(token[:]), sessionLifetime); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(token[:]), nil
}

func (s *Service) SessionUser(ctx context.Context, token string) (string, error) {
	if len(token) != base64.RawURLEncoding.EncodedLen(tokenSize) {
		return "", ErrUnauthenticated
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil {
		return "", ErrUnauthenticated
	}
	return s.repo.sessionUser(ctx, sha256.Sum256(raw))
}
