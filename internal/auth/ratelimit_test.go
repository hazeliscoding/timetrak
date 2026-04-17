package auth

import (
	"testing"
	"time"
)

func TestRateLimiterBurstAndRefill(t *testing.T) {
	l := NewRateLimiter()
	now := time.Now()
	l.now = func() time.Time { return now }
	for i := 0; i < rateBurst; i++ {
		if !l.Allow("1.2.3.4") {
			t.Fatalf("burst %d should be allowed", i)
		}
	}
	if l.Allow("1.2.3.4") {
		t.Fatalf("expected throttle after burst exhausted")
	}
	// Advance time by a full refill interval and try again.
	now = now.Add(rateRefill)
	if !l.Allow("1.2.3.4") {
		t.Fatalf("expected refill to permit another request")
	}
	// Other IPs independent.
	if !l.Allow("9.9.9.9") {
		t.Fatalf("other ip should be allowed")
	}
}
