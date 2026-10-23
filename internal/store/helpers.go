package store

import (
	"math"
	"strings"
)

func NormalizeName(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
