// Package clock provides an injectable clock so that time-sensitive logic
// (rate resolution, timer start/stop) can be deterministic in tests.
package clock

import "time"

// Clock is the interface services depend on instead of time.Now.
type Clock interface {
	Now() time.Time
}

// System is the production clock.
type System struct{}

// Now returns the current wall-clock time in UTC.
func (System) Now() time.Time { return time.Now().UTC() }

// Fixed returns a clock that always reports the given instant. Intended for tests.
type Fixed struct{ T time.Time }

// Now returns the fixed instant.
func (f Fixed) Now() time.Time { return f.T }
