package store

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

func MatchID(names []string, score1, score2 int) string {
	raw := strings.Join(append(append([]string{}, names...), fmt.Sprintf("%d", score1), fmt.Sprintf("%d", score2)), "|")
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", h[:8])
}
