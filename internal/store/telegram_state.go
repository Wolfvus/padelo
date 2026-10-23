package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

func (s *Store) SetActiveTeams(ctx context.Context, key string, team1, team2 [2]string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO contexts (context_key,team1,team2) VALUES ($1,$2,$3) ON CONFLICT(context_key) DO UPDATE SET team1=excluded.team1,team2=excluded.team2`, key, team1[:], team2[:])
	return err
}

func (s *Store) ActiveTeams(ctx context.Context, key string) ([2]string, [2]string, error) {
	var a, b []string
	err := s.pool.QueryRow(ctx, `SELECT team1,team2 FROM contexts WHERE context_key=$1`, key).Scan(&a, &b)
	if errors.Is(err, pgx.ErrNoRows) {
		return [2]string{}, [2]string{}, ErrNotFound
	}
	if err != nil {
		return [2]string{}, [2]string{}, err
	}
	if len(a) != 2 || len(b) != 2 {
		return [2]string{}, [2]string{}, errors.New("invalid stored teams")
	}
	return [2]string{a[0], a[1]}, [2]string{b[0], b[1]}, nil
}

func (s *Store) TryMarkTelegramUpdate(ctx context.Context, updateID int64) (bool, error) {
	ct, err := s.pool.Exec(ctx, `INSERT INTO telegram_updates(update_id) VALUES($1) ON CONFLICT DO NOTHING`, updateID)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() == 1, nil
}
