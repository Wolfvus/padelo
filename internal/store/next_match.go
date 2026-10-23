package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (s *Store) NextMatch(ctx context.Context) (NextMatch, error) {
	var next NextMatch
	err := s.pool.QueryRow(ctx, `
SELECT to_char(match_date, 'YYYY-MM-DD'), to_char(match_time, 'HH24:MI'), location, updated_at
FROM next_match
WHERE singleton = TRUE
`).Scan(&next.Date, &next.Time, &next.Location, &next.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return NextMatch{}, ErrNotFound
	}
	return next, err
}

func (s *Store) SaveNextMatch(ctx context.Context, date, matchTime, location string) (NextMatch, error) {
	var next NextMatch
	err := s.pool.QueryRow(ctx, `
INSERT INTO next_match (singleton, match_date, match_time, location)
VALUES (TRUE, $1::date, $2::time, $3)
ON CONFLICT (singleton) DO UPDATE SET
  match_date = EXCLUDED.match_date,
  match_time = EXCLUDED.match_time,
  location = EXCLUDED.location,
  updated_at = NOW()
RETURNING to_char(match_date, 'YYYY-MM-DD'), to_char(match_time, 'HH24:MI'), location, updated_at
`, date, matchTime, location).Scan(&next.Date, &next.Time, &next.Location, &next.UpdatedAt)
	return next, err
}

func (s *Store) ClearNextMatch(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM next_match WHERE singleton = TRUE`)
	return err
}
