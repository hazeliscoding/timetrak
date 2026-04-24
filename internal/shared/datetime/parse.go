// Package datetime exposes a small, focused helper for parsing split
// date + time inputs under a named IANA timezone.
//
// Background: TimeTrak stores timestamps as UTC (`timestamptz`), but
// users interact with clock times in their workspace's reporting
// timezone. The tracking handlers (manual create, entry edit) submit
// date and time as two separate form fields — one `type="date"` and
// one `type="time"`. This package is the single source of truth for
// converting that pair into a UTC `time.Time`.
//
// See openspec/specs/tracking/spec.md (Datetime input parse and display
// is workspace-timezone-aware).
package datetime

import (
	"errors"
	"fmt"
	"time"
)

// FieldError names which input the handler should attach the validation
// message to. Handlers use Field to drive `aria-describedby` / per-field
// error surfacing via the existing tracking_error partial path.
type FieldError struct {
	Field  string // "start_date", "start_time", "end_date", "end_time", or ""
	Reason string
}

func (e *FieldError) Error() string {
	if e.Field == "" {
		return e.Reason
	}
	return e.Field + ": " + e.Reason
}

// ErrEmptyInput is the sentinel reason for an empty date or time string.
var ErrEmptyInput = errors.New("empty input")

// ParseLocalDateTime parses a YYYY-MM-DD `date` and a HH:MM `timeStr`
// as a clock time in the named IANA `tz`, returning the equivalent UTC
// `time.Time`.
//
// Both `date` and `timeStr` MUST be non-empty. The `tz` MUST be an IANA
// zone name known to the runtime's tzdata; empty `tz` is treated as
// `"UTC"` so callers with a default-UTC workspace do not need to
// special-case.
//
// On failure the returned error wraps `*FieldError` so the handler can
// identify which input to mark invalid. Parse failures do NOT leak the
// raw input value into the error message — error strings are safe to
// render back to the user without escaping.
//
// DST behavior is delegated to `time.ParseInLocation`:
//   - Spring-forward (ambiguous non-existent time, e.g. 02:30 EST→EDT):
//     stdlib returns the post-transition time (03:30 local).
//   - Fall-back (ambiguous doubly-existent time, e.g. 01:30 EDT→EST):
//     stdlib returns the first occurrence. This is a documented
//     trade-off, not a bug.
func ParseLocalDateTime(date, timeStr, tz string) (time.Time, error) {
	if date == "" {
		return time.Time{}, &FieldError{Field: "date", Reason: "date is required"}
	}
	if timeStr == "" {
		return time.Time{}, &FieldError{Field: "time", Reason: "time is required"}
	}
	if tz == "" {
		tz = "UTC"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, &FieldError{Field: "", Reason: fmt.Sprintf("unknown timezone %q", tz)}
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", date+" "+timeStr, loc)
	if err != nil {
		// Disambiguate which field failed by re-parsing each independently.
		if _, dateErr := time.Parse("2006-01-02", date); dateErr != nil {
			return time.Time{}, &FieldError{Field: "date", Reason: "must be YYYY-MM-DD"}
		}
		if _, timeErr := time.Parse("15:04", timeStr); timeErr != nil {
			return time.Time{}, &FieldError{Field: "time", Reason: "must be HH:MM (24-hour)"}
		}
		// Both parse independently but the combined parse failed —
		// surface a generic field-less error so the caller can still
		// render something useful.
		return time.Time{}, &FieldError{Field: "", Reason: "invalid date or time"}
	}
	return t.UTC(), nil
}
