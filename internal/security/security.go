package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

func ConstantTimeEqual(a, b string) bool {
	ha := sha256.Sum256([]byte(a))
	hb := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}

func NewOpaqueToken() (raw string, hash [32]byte, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", hash, err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	hash = sha256.Sum256([]byte(raw))
	return raw, hash, nil
}

func HashToken(raw string) [32]byte { return sha256.Sum256([]byte(raw)) }

type attemptState struct {
	Failures    []time.Time
	LockedUntil time.Time
	LastSeen    time.Time
}

type LoginLimiter struct {
	mu        sync.Mutex
	states    map[string]*attemptState
	global    []time.Time
	max       int
	window    time.Duration
	lockout   time.Duration
	globalMax int
}

func NewLoginLimiter() *LoginLimiter {
	return &LoginLimiter{states: map[string]*attemptState{}, max: 5, window: 10 * time.Minute, lockout: 15 * time.Minute, globalMax: 60}
}

func (l *LoginLimiter) RetryAfter(key string, now time.Time) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prune(now)
	if len(l.global) >= l.globalMax {
		return time.Minute
	}
	st := l.states[key]
	if st == nil || !st.LockedUntil.After(now) {
		return 0
	}
	return st.LockedUntil.Sub(now)
}

func (l *LoginLimiter) Failure(key string, now time.Time) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prune(now)
	l.global = append(l.global, now)
	st := l.states[key]
	if st == nil {
		st = &attemptState{}
		l.states[key] = st
	}
	st.LastSeen = now
	st.Failures = append(st.Failures, now)
	if len(st.Failures) >= l.max {
		st.Failures = nil
		st.LockedUntil = now.Add(l.lockout)
		return l.lockout
	}
	return 0
}

func (l *LoginLimiter) Success(key string) {
	l.mu.Lock()
	delete(l.states, key)
	l.mu.Unlock()
}

func (l *LoginLimiter) prune(now time.Time) {
	cut := now.Add(-l.window)
	g := l.global[:0]
	for _, t := range l.global {
		if t.After(cut) {
			g = append(g, t)
		}
	}
	l.global = g
	for k, st := range l.states {
		f := st.Failures[:0]
		for _, t := range st.Failures {
			if t.After(cut) {
				f = append(f, t)
			}
		}
		st.Failures = f
		if !st.LockedUntil.After(now) && len(st.Failures) == 0 && st.LastSeen.Before(cut) {
			delete(l.states, k)
		}
	}
}

func ClientKey(r *http.Request, trustCloudflare bool) string {
	if trustCloudflare {
		if ip := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); ip != "" {
			if parsed := net.ParseIP(ip); parsed != nil {
				return parsed.String()
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}

func RequireOrigin(r *http.Request, publicOrigin string) error {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return nil
	}
	origin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
	if origin == "" || origin != publicOrigin {
		return errors.New("invalid request origin")
	}
	return nil
}
