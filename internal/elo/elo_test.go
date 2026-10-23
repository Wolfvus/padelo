package elo

import "testing"

func TestApplyIsZeroSum(t *testing.T) {
	w := [2]Player{{Name: "A", Elo: 1500}, {Name: "B", Elo: 1500}}
	l := [2]Player{{Name: "C", Elo: 1500}, {Name: "D", Elo: 1500}}
	changes, err := Apply(w, l, 6, 4, 32)
	if err != nil {
		t.Fatal(err)
	}
	var total float64
	for _, c := range changes {
		total += c.Delta
	}
	if total < -0.01 || total > 0.01 {
		t.Fatalf("expected zero-sum deltas, got %.2f", total)
	}
}

func TestRejectsDuplicatePlayers(t *testing.T) {
	_, err := Apply([2]Player{{Name: "A", Elo: 1500}, {Name: "A", Elo: 1500}}, [2]Player{{Name: "C", Elo: 1500}, {Name: "D", Elo: 1500}}, 6, 4, 32)
	if err == nil {
		t.Fatal("expected duplicate player error")
	}
}
