package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

func (s *Store) Rankings(ctx context.Context) ([]Ranking, error) {
	rows, err := s.pool.Query(ctx, `SELECT name, elo FROM players ORDER BY elo DESC, lower(name), name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Ranking{}
	pos := 1
	for rows.Next() {
		var r Ranking
		if err := rows.Scan(&r.Name, &r.Elo); err != nil {
			return nil, err
		}
		r.Position = pos
		pos++
		r.Elo = round2(r.Elo)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) CreatePlayer(ctx context.Context, name string, rating float64) error {
	name = NormalizeName(name)
	if name == "" {
		return errors.New("player name cannot be empty")
	}
	if len([]rune(name)) > 80 {
		return errors.New("player name is too long")
	}
	if rating < -100000 || rating > 100000 {
		return errors.New("invalid elo")
	}
	ct, err := s.pool.Exec(ctx, `INSERT INTO players (name, elo) VALUES ($1,$2) ON CONFLICT DO NOTHING`, name, rating)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrDuplicatePlayer
	}
	return nil
}

func (s *Store) AdjustPlayerElo(ctx context.Context, name string, delta float64) (float64, error) {
	if delta < -10000 || delta > 10000 {
		return 0, errors.New("invalid elo delta")
	}
	var rating float64
	err := s.pool.QueryRow(ctx, `UPDATE players SET elo=elo+$1 WHERE lower(name)=lower($2) RETURNING elo`, delta, NormalizeName(name)).Scan(&rating)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrUnknownPlayer
	}
	return round2(rating), err
}

func (s *Store) DeletePlayer(ctx context.Context, name string) error {
	name = NormalizeName(name)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var hasHistory bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM matches WHERE EXISTS (SELECT 1 FROM unnest(winners || losers) p WHERE lower(p)=lower($1)))`, name).Scan(&hasHistory); err != nil {
		return err
	}
	if hasHistory {
		return ErrPlayerHasHistory
	}
	ct, err := tx.Exec(ctx, `DELETE FROM players WHERE lower(name)=lower($1)`, name)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrUnknownPlayer
	}
	return tx.Commit(ctx)
}
