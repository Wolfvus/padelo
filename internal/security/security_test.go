package security

import "testing"

func TestOpaqueToken(t *testing.T) {
	raw, hash, err := NewOpaqueToken()
	if err != nil {
		t.Fatal(err)
	}
	if raw == "" {
		t.Fatal("empty token")
	}
	if HashToken(raw) != hash {
		t.Fatal("hash mismatch")
	}
}

func TestConstantTimeEqual(t *testing.T) {
	if !ConstantTimeEqual("correct", "correct") {
		t.Fatal("equal values rejected")
	}
	if ConstantTimeEqual("correct", "wrong") {
		t.Fatal("different values accepted")
	}
}
