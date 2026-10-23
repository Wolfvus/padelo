package httpapp

import (
	"math"
	"strings"
)

func containsFold(names []string, lower string) bool {
	for _, n := range names {
		if strings.ToLower(n) == lower {
			return true
		}
	}
	return false
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
