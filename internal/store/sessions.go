package store

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateSession(ctx context.Context, tokenHash [sha256.Size]byte, role string, expires time.Time) error {
	if role != "admin" {
		return errors.New("invalid session role")
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO web_sessions (token_hash, role, expires_at) VALUES ($1,$2,$3)`, tokenHash[:], role, expires)
	return err
}

func (s *Store) SessionRole(ctx context.Context, tokenHash [sha256.Size]byte) (string, error) {
	var role string
	err := s.pool.QueryRow(ctx, `SELECT role FROM web_sessions WHERE token_hash=$1 AND expires_at > NOW()`, tokenHash[:]).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return role, err
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash [sha256.Size]byte) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM web_sessions WHERE token_hash=$1`, tokenHash[:])
	return err
}

func (s *Store) PruneSessions(ctx context.Context) {
	_, _ = s.pool.Exec(ctx, `DELETE FROM web_sessions WHERE expires_at <= NOW()`)
}
