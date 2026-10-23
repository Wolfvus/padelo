package elo

import (
	"errors"
	"math"
	"strings"
)

type Player struct {
	Name string  `json:"name"`
	Elo  float64 `json:"elo"`
}

type Change struct {
	Name   string  `json:"name"`
	Before float64 `json:"before"`
	After  float64 `json:"after"`
	Delta  float64 `json:"delta"`
}

func ExpectedScore(a, b float64) float64 {
	return 1 / (1 + math.Pow(10, (b-a)/400))
}

func ScoreFactor(winner, loser int) (float64, error) {
	if winner < 0 || loser < 0 {
		return 0, errors.New("scores cannot be negative")
	}
	if winner <= loser {
		return 0, errors.New("winner score must be greater than loser score")
	}
	total := winner + loser
	if total == 0 {
		return 0, errors.New("total games cannot be zero")
	}
	return 1 + float64(winner-loser)/float64(total), nil
}

func Apply(winners, losers [2]Player, winnerScore, loserScore int, k float64) ([]Change, error) {
	if k <= 0 || k > 128 {
		return nil, errors.New("invalid k factor")
	}
	seen := map[string]struct{}{}
	for _, p := range []Player{winners[0], winners[1], losers[0], losers[1]} {
		key := strings.ToLower(strings.TrimSpace(p.Name))
		if key == "" {
			return nil, errors.New("player names cannot be empty")
		}
		if _, ok := seen[key]; ok {
			return nil, errors.New("a match must contain four distinct players")
		}
		seen[key] = struct{}{}
	}
	factor, err := ScoreFactor(winnerScore, loserScore)
	if err != nil {
		return nil, err
	}
	winnerTeam := (winners[0].Elo + winners[1].Elo) / 2
	loserTeam := (losers[0].Elo + losers[1].Elo) / 2
	expectedWinner := ExpectedScore(winnerTeam, loserTeam)
	winnerDelta := k * factor * (1 - expectedWinner)
	loserDelta := -winnerDelta

	changes := make([]Change, 0, 4)
	for _, p := range winners {
		changes = append(changes, Change{Name: p.Name, Before: round2(p.Elo), After: round2(p.Elo + winnerDelta), Delta: round2(winnerDelta)})
	}
	for _, p := range losers {
		changes = append(changes, Change{Name: p.Name, Before: round2(p.Elo), After: round2(p.Elo + loserDelta), Delta: round2(loserDelta)})
	}
	return changes, nil
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
