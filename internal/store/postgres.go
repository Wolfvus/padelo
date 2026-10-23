package store

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

func New(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = 6
	cfg.MinConns = 0
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	ctxPing, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err := pool.Ping(ctxPing); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) EnsureSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS players (
  name TEXT PRIMARY KEY,
  elo DOUBLE PRECISION NOT NULL CHECK (elo > -100000 AND elo < 100000)
);
CREATE UNIQUE INDEX IF NOT EXISTS players_name_ci_idx ON players (lower(name));
CREATE TABLE IF NOT EXISTS matches (
  id TEXT PRIMARY KEY,
  winners TEXT[] NOT NULL,
  losers TEXT[] NOT NULL,
  winner_score INTEGER NOT NULL CHECK (winner_score >= 0 AND winner_score <= 99),
  loser_score INTEGER NOT NULL CHECK (loser_score >= 0 AND loser_score <= 99),
  changes_json TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE matches ADD COLUMN IF NOT EXISTS sets_json TEXT NOT NULL DEFAULT '[]';
CREATE INDEX IF NOT EXISTS matches_created_at_idx ON matches (created_at DESC);
CREATE TABLE IF NOT EXISTS contexts (
  context_key TEXT PRIMARY KEY,
  team1 TEXT[] NOT NULL,
  team2 TEXT[] NOT NULL
);
CREATE TABLE IF NOT EXISTS web_sessions (
  token_hash BYTEA PRIMARY KEY,
  role TEXT NOT NULL CHECK (role IN ('admin')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS web_sessions_expires_at_idx ON web_sessions (expires_at);
CREATE TABLE IF NOT EXISTS telegram_updates (
  update_id BIGINT PRIMARY KEY,
  received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS next_match (
  singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
  match_date DATE NOT NULL,
  match_time TIME NOT NULL,
  location TEXT NOT NULL CHECK (char_length(location) BETWEEN 1 AND 160),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`)
	return err
}

func (s *Store) Health(ctx context.Context) error { return s.pool.Ping(ctx) }
