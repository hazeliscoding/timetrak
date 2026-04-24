package datetime

import (
	"errors"
	"testing"
	"time"
)

func TestParseLocalDateTime_HappyPaths(t *testing.T) {
	tests := []struct {
		name   string
		date   string
		time   string
		tz     string
		wantUS string // expected UTC time, RFC3339
	}{
		{"UTC", "2026-04-24", "09:00", "UTC", "2026-04-24T09:00:00Z"},
		{"UTC via empty tz", "2026-04-24", "09:00", "", "2026-04-24T09:00:00Z"},
		{"New York (EDT, -04)", "2026-04-24", "10:00", "America/New_York", "2026-04-24T14:00:00Z"},
		{"Los Angeles (PDT, -07)", "2026-04-24", "09:00", "America/Los_Angeles", "2026-04-24T16:00:00Z"},
		{"Tokyo (JST, +09)", "2026-04-24", "18:00", "Asia/Tokyo", "2026-04-24T09:00:00Z"},
		{"Midnight local", "2026-04-24", "00:00", "America/New_York", "2026-04-24T04:00:00Z"},
		{"End of day local", "2026-04-24", "23:59", "America/New_York", "2026-04-25T03:59:00Z"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseLocalDateTime(tc.date, tc.time, tc.tz)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Location() != time.UTC {
				t.Fatalf("location: got %v, want UTC", got.Location())
			}
			want, _ := time.Parse(time.RFC3339, tc.wantUS)
			if !got.Equal(want) {
				t.Fatalf("got %s, want %s", got.Format(time.RFC3339), tc.wantUS)
			}
		})
	}
}

func TestParseLocalDateTime_InvalidInputs(t *testing.T) {
	tests := []struct {
		name      string
		date      string
		time      string
		tz        string
		wantField string
	}{
		{"empty date", "", "09:00", "UTC", "date"},
		{"empty time", "2026-04-24", "", "UTC", "time"},
		{"malformed date", "not-a-date", "09:00", "UTC", "date"},
		{"malformed time (25:99)", "2026-04-24", "25:99", "UTC", "time"},
		{"malformed time (partial)", "2026-04-24", "9", "UTC", "time"},
		{"non-existent month", "2026-13-01", "09:00", "UTC", "date"},
		{"invalid tz", "2026-04-24", "09:00", "Not/A/Zone", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseLocalDateTime(tc.date, tc.time, tc.tz)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			var fe *FieldError
			if !errors.As(err, &fe) {
				t.Fatalf("expected *FieldError, got %T: %v", err, err)
			}
			if fe.Field != tc.wantField {
				t.Fatalf("field: got %q, want %q", fe.Field, tc.wantField)
			}
			if fe.Reason == "" {
				t.Fatal("reason: empty")
			}
		})
	}
}

// TestParseLocalDateTime_SpringForwardNYC pins the documented stdlib
// behavior for a non-existent clock time. 02:30 on 2026-03-08 does NOT
// exist in America/New_York; stdlib resolves it to the post-transition
// equivalent.
func TestParseLocalDateTime_SpringForwardNYC(t *testing.T) {
	got, err := ParseLocalDateTime("2026-03-08", "02:30", "America/New_York")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Stdlib behavior: Go's ParseInLocation applies the post-transition
	// offset to the non-existent wall clock. Result: 02:30 interpreted
	// with the EDT offset (-04) → 06:30 UTC. This is a known stdlib
	// convention; test pins it so a future Go version regression would
	// surface here.
	want, _ := time.Parse(time.RFC3339, "2026-03-08T06:30:00Z")
	if !got.Equal(want) {
		t.Fatalf("DST spring-forward: got %s, want %s (stdlib behavior)",
			got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

// TestParseLocalDateTime_FallBackNYC pins the documented stdlib behavior
// for an ambiguous clock time. 01:30 on 2026-11-01 exists twice in
// America/New_York (once in EDT, once in EST). Stdlib returns the first
// occurrence (EDT).
func TestParseLocalDateTime_FallBackNYC(t *testing.T) {
	got, err := ParseLocalDateTime("2026-11-01", "01:30", "America/New_York")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// First occurrence is EDT (-04): 01:30 local → 05:30 UTC.
	want, _ := time.Parse(time.RFC3339, "2026-11-01T05:30:00Z")
	if !got.Equal(want) {
		t.Fatalf("DST fall-back: got %s, want %s (stdlib returns first occurrence)",
			got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}
