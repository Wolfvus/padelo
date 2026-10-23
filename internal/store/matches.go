package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Wolfvus/padelo/internal/elo"
	"github.com/jackc/pgx/v5"
	"sort"
	"strings"
	"time"
)

func (s *Store) RenamePlayer(ctx context.Context, oldName, newName string) (string, error) {
	oldName, newName = NormalizeName(oldName), NormalizeName(newName)
	if newName == "" || len([]rune(newName)) > 80 {
		return "", errors.New("invalid player name")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var canonical string
	if err := tx.QueryRow(ctx, `SELECT name FROM players WHERE lower(name)=lower($1) FOR UPDATE`, oldName).Scan(&canonical); errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUnknownPlayer
	} else if err != nil {
		return "", err
	}
	var collision bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM players WHERE lower(name)=lower($1) AND lower(name)<>lower($2))`, newName, canonical).Scan(&collision); err != nil {
		return "", err
	}
	if collision {
		return "", ErrDuplicatePlayer
	}

	rows, err := tx.Query(ctx, `SELECT id, changes_json FROM matches WHERE EXISTS (SELECT 1 FROM unnest(winners || losers) p WHERE lower(p)=lower($1)) FOR UPDATE`, canonical)
	if err != nil {
		return "", err
	}
	type changeRow struct{ id, raw string }
	var affected []changeRow
	for rows.Next() {
		var r changeRow
		if err := rows.Scan(&r.id, &r.raw); err != nil {
			rows.Close()
			return "", err
		}
		affected = append(affected, r)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	rows.Close()

	if _, err := tx.Exec(ctx, `UPDATE players SET name=$1 WHERE name=$2`, newName, canonical); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE matches SET winners=array_replace(winners,$1,$2), losers=array_replace(losers,$1,$2) WHERE $1=ANY(winners) OR $1=ANY(losers)`, canonical, newName); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE contexts SET team1=array_replace(team1,$1,$2), team2=array_replace(team2,$1,$2) WHERE $1=ANY(team1) OR $1=ANY(team2)`, canonical, newName); err != nil {
		return "", err
	}
	for _, r := range affected {
		var changes []elo.Change
		if err := json.Unmarshal([]byte(r.raw), &changes); err != nil {
			return "", fmt.Errorf("decode match changes: %w", err)
		}
		for i := range changes {
			if strings.EqualFold(changes[i].Name, canonical) {
				changes[i].Name = newName
			}
		}
		b, _ := json.Marshal(changes)
		if _, err := tx.Exec(ctx, `UPDATE matches SET changes_json=$1 WHERE id=$2`, string(b), r.id); err != nil {
			return "", err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return newName, nil
}

func (s *Store) RecordMatch(ctx context.Context, in RecordInput) (Match, []Ranking, error) {
	if in.K == 0 {
		in.K = 32
	}
	if in.Score1 < 0 || in.Score1 > 99 || in.Score2 < 0 || in.Score2 > 99 || in.Score1 == in.Score2 {
		return Match{}, nil, errors.New("invalid match score")
	}
	all := []string{NormalizeName(in.Team1[0]), NormalizeName(in.Team1[1]), NormalizeName(in.Team2[0]), NormalizeName(in.Team2[1])}
	seen := map[string]struct{}{}
	for _, n := range all {
		k := strings.ToLower(n)
		if n == "" {
			return Match{}, nil, errors.New("player names cannot be empty")
		}
		if _, ok := seen[k]; ok {
			return Match{}, nil, errors.New("a match must contain four distinct players")
		}
		seen[k] = struct{}{}
	}
	lockOrder := append([]string(nil), all...)
	sort.Slice(lockOrder, func(i, j int) bool { return strings.ToLower(lockOrder[i]) < strings.ToLower(lockOrder[j]) })
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return Match{}, nil, err
	}
	defer tx.Rollback(ctx)
	players := map[string]elo.Player{}
	for _, n := range lockOrder {
		var p elo.Player
		err := tx.QueryRow(ctx, `SELECT name, elo FROM players WHERE lower(name)=lower($1) FOR UPDATE`, n).Scan(&p.Name, &p.Elo)
		if errors.Is(err, pgx.ErrNoRows) {
			return Match{}, nil, fmt.Errorf("%w: %s", ErrUnknownPlayer, n)
		}
		if err != nil {
			return Match{}, nil, err
		}
		players[strings.ToLower(n)] = p
	}
	if in.ID == "" {
		in.ID = MatchID(all, in.Score1, in.Score2)
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM matches WHERE id=$1)`, in.ID).Scan(&exists); err != nil {
		return Match{}, nil, err
	}
	if exists {
		return Match{}, nil, ErrDuplicateMatch
	}

	team1 := [2]elo.Player{players[strings.ToLower(all[0])], players[strings.ToLower(all[1])]}
	team2 := [2]elo.Player{players[strings.ToLower(all[2])], players[strings.ToLower(all[3])]}
	winners, losers := team1, team2
	winnerScore, loserScore := in.Score1, in.Score2
	if in.Score2 > in.Score1 {
		winners, losers = team2, team1
		winnerScore, loserScore = in.Score2, in.Score1
	}
	changes, err := elo.Apply(winners, losers, winnerScore, loserScore, in.K)
	if err != nil {
		return Match{}, nil, err
	}
	for _, c := range changes {
		if _, err := tx.Exec(ctx, `UPDATE players SET elo=$1 WHERE name=$2`, c.After, c.Name); err != nil {
			return Match{}, nil, err
		}
	}
	changesJSON, _ := json.Marshal(changes)
	setsJSON, _ := json.Marshal(in.Sets)
	created := time.Now().UTC()
	winnerNames := []string{winners[0].Name, winners[1].Name}
	loserNames := []string{losers[0].Name, losers[1].Name}
	_, err = tx.Exec(ctx, `INSERT INTO matches (id,winners,losers,winner_score,loser_score,changes_json,sets_json,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, in.ID, winnerNames, loserNames, winnerScore, loserScore, string(changesJSON), string(setsJSON), created)
	if err != nil {
		return Match{}, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Match{}, nil, err
	}
	rankings, err := s.Rankings(ctx)
	if err != nil {
		return Match{}, nil, err
	}
	return Match{ID: in.ID, Winners: winnerNames, Losers: loserNames, WinnerScore: winnerScore, LoserScore: loserScore, Changes: changes, Sets: in.Sets, CreatedAt: created}, rankings, nil
}

func (s *Store) RecentMatches(ctx context.Context, limit int) ([]Match, error) {
	if limit < 1 || limit > 10000 {
		return nil, errors.New("invalid match limit")
	}
	rows, err := s.pool.Query(ctx, `SELECT id,winners,losers,winner_score,loser_score,changes_json,sets_json,created_at FROM matches ORDER BY created_at DESC,id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Match{}
	for rows.Next() {
		var m Match
		var changesRaw, setsRaw string
		if err := rows.Scan(&m.ID, &m.Winners, &m.Losers, &m.WinnerScore, &m.LoserScore, &changesRaw, &setsRaw, &m.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(changesRaw), &m.Changes); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(setsRaw), &m.Sets); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) RollbackLatest(ctx context.Context) (Match, []Ranking, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return Match{}, nil, err
	}
	defer tx.Rollback(ctx)
	var m Match
	var changesRaw, setsRaw string
	err = tx.QueryRow(ctx, `SELECT id,winners,losers,winner_score,loser_score,changes_json,sets_json,created_at FROM matches ORDER BY created_at DESC,id DESC LIMIT 1 FOR UPDATE`).Scan(&m.ID, &m.Winners, &m.Losers, &m.WinnerScore, &m.LoserScore, &changesRaw, &setsRaw, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Match{}, nil, ErrNotFound
	}
	if err != nil {
		return Match{}, nil, err
	}
	if err := json.Unmarshal([]byte(changesRaw), &m.Changes); err != nil {
		return Match{}, nil, err
	}
	_ = json.Unmarshal([]byte(setsRaw), &m.Sets)
	for _, c := range m.Changes {
		if _, err := tx.Exec(ctx, `UPDATE players SET elo=$1 WHERE name=$2`, c.Before, c.Name); err != nil {
			return Match{}, nil, err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM matches WHERE id=$1`, m.ID); err != nil {
		return Match{}, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Match{}, nil, err
	}
	rankings, err := s.Rankings(ctx)
	return m, rankings, err
}
