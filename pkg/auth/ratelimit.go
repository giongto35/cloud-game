package auth

import (
	"sync"
	"time"
)

// Limiter is a small in-memory token bucket keyed by an arbitrary string (IP).
type Limiter struct {
	mu      sync.Mutex
	rate    float64 // tokens per second
	burst   float64
	buckets map[string]*bucket
	last    time.Time
}

type bucket struct {
	tokens float64
	at     time.Time
}

func NewLimiter(perMinute int, burst int) *Limiter {
	return &Limiter{rate: float64(perMinute) / 60, burst: float64(burst), buckets: map[string]*bucket{}}
}

// Allow consumes one token for the key and reports whether it was available.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if now.Sub(l.last) > 10*time.Minute {
		l.gc(now)
		l.last = now
	}
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, at: now}
		l.buckets[key] = b
	}
	b.tokens = min(l.burst, b.tokens+now.Sub(b.at).Seconds()*l.rate)
	b.at = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *Limiter) gc(now time.Time) {
	for k, b := range l.buckets {
		if now.Sub(b.at) > 30*time.Minute {
			delete(l.buckets, k)
		}
	}
}
