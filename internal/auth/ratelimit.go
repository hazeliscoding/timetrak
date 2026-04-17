package auth

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Documented rate-limit policy for /login and /signup:
//
//	At most 10 POST attempts per remote IP per 10-minute rolling window.
//	Exceeding the threshold yields HTTP 429 until the window elapses.
//
// Implementation: in-memory per-IP token bucket refilled at 1 token / 60s with a burst cap of 10.
// This is adequate for a single-process MVP; swap for a Redis-backed bucket when scaling out.
const (
	rateBurst      = 10
	rateRefill     = time.Minute
	rateMaxIdleAge = 30 * time.Minute
)

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// RateLimiter is a very small thread-safe IP-keyed token bucket.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	now     func() time.Time
}

// NewRateLimiter returns an empty limiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{buckets: map[string]*bucket{}, now: time.Now}
}

// Allow reports whether a request from ip should be allowed; it also consumes one token if so.
func (l *RateLimiter) Allow(ip string) bool {
	if ip == "" {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b, ok := l.buckets[ip]
	if !ok {
		b = &bucket{tokens: rateBurst, lastSeen: now}
		l.buckets[ip] = b
	}
	// Refill.
	elapsed := now.Sub(b.lastSeen)
	if elapsed > 0 {
		refill := float64(elapsed) / float64(rateRefill)
		b.tokens += refill
		if b.tokens > rateBurst {
			b.tokens = rateBurst
		}
	}
	b.lastSeen = now
	// Opportunistic GC of idle IPs.
	if len(l.buckets) > 512 {
		for k, v := range l.buckets {
			if now.Sub(v.lastSeen) > rateMaxIdleAge {
				delete(l.buckets, k)
			}
		}
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// ClientIP extracts the remote IP from the request, honoring X-Forwarded-For if present.
func ClientIP(r *http.Request) string {
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		if i := strings.IndexByte(xf, ','); i >= 0 {
			return strings.TrimSpace(xf[:i])
		}
		return strings.TrimSpace(xf)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
