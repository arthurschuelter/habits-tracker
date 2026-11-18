package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	return &PostgresRepo{db: db}
}

// SetPassword identifies existing accounts by their unique email and revokes
// their session in the same write.
func (r *PostgresRepo) SetPassword(ctx context.Context, email string, password []byte) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE users SET password_hash = $2, session_token_hash = NULL, session_expires_at = NULL
		WHERE email = $1`, email, hash)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.Constraint == "users_login_username_key" {
			return ErrUsernameTaken
		}
		return fmt.Errorf("set user password: %w", err)
	}
	return requireUserUpdate(result, ErrUserNotFound)
}

func (r *PostgresRepo) credentials(ctx context.Context, username string) (credentials, error) {
	var creds credentials
	err := r.db.QueryRowContext(ctx, `
		SELECT id, password_hash FROM users WHERE username = $1 AND password_hash IS NOT NULL`, username).
		Scan(&creds.userID, &creds.passwordHash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return credentials{}, fmt.Errorf("load login credentials: %w", err)
	}
	return creds, nil
}

func (r *PostgresRepo) saveSession(ctx context.Context, creds credentials, hash [32]byte, lifetime time.Duration) error {
	// Check the verified hash again in case the password was reset during login.
	result, err := r.db.ExecContext(ctx, `
		UPDATE users SET session_token_hash = $3,
			session_expires_at = CURRENT_TIMESTAMP + $4 * INTERVAL '1 second'
		WHERE id = $1 AND password_hash = $2`, creds.userID, creds.passwordHash, hash[:], int64(lifetime/time.Second))
	if err != nil {
		return fmt.Errorf("save login session: %w", err)
	}
	return requireUserUpdate(result, ErrCredentials)
}

func (r *PostgresRepo) sessionUser(ctx context.Context, hash [32]byte) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `
		SELECT id FROM users WHERE session_token_hash = $1 AND session_expires_at > CURRENT_TIMESTAMP`, hash[:]).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrUnauthenticated
	}
	if err != nil {
		return "", fmt.Errorf("look up login session: %w", err)
	}
	return id, nil
}

func requireUserUpdate(result sql.Result, missing error) error {
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check user update: %w", err)
	}
	if n == 0 {
		return missing
	}
	return nil
}
