package store

import (
	"errors"
	"github.com/Wolfvus/padelo/internal/elo"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrUnknownPlayer    = errors.New("unknown player")
	ErrDuplicatePlayer  = errors.New("player already exists")
	ErrDuplicateMatch   = errors.New("match already processed")
	ErrPlayerHasHistory = errors.New("player has match history")
)

type Store struct{ pool *pgxpool.Pool }

type NextMatch struct {
	Date      string    `json:"date"`
	Time      string    `json:"time"`
	Location  string    `json:"location"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Ranking struct {
	Position   int      `json:"position"`
	Name       string   `json:"name"`
	Elo        float64  `json:"elo"`
	RankChange int      `json:"rank_change,omitempty"`
	LastDelta  *float64 `json:"last_delta,omitempty"`
}

type SetScore struct {
	Team1 int `json:"team1"`
	Team2 int `json:"team2"`
}

type Match struct {
	ID          string       `json:"id"`
	Winners     []string     `json:"winners"`
	Losers      []string     `json:"losers"`
	WinnerScore int          `json:"winner_score"`
	LoserScore  int          `json:"loser_score"`
	Changes     []elo.Change `json:"changes"`
	Sets        []SetScore   `json:"sets"`
	CreatedAt   time.Time    `json:"created_at"`
}

type RecordInput struct {
	Team1  [2]string
	Team2  [2]string
	Score1 int
	Score2 int
	K      float64
	ID     string
	Sets   []SetScore
}
