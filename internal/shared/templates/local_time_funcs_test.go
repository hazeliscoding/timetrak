package templates

import (
	"testing"
	"time"
)

// TestFormatLocalDateTime covers the new formatLocalDate /
// formatLocalTime template funcs. The funcs are registered in
// baseFuncs() and exercised here at package level via toLocation so
// the FuncMap entries stay pure proxies.
func TestFormatLocalDate(t *testing.T) {
	// 2026-04-24T14:00:00Z → 10:00 in New York, 07:00 in Los Angeles.
	utc, _ := time.Parse(time.RFC3339, "2026-04-24T14:00:00Z")

	cases := []struct {
		name string
		tz   string
		want string
	}{
		{"UTC explicit", "UTC", "2026-04-24"},
		{"UTC via empty tz", "", "2026-04-24"},
		{"New York (EDT)", "America/New_York", "2026-04-24"},
		{"Tokyo (crosses day line)", "Asia/Tokyo", "2026-04-24"},          // 23:00 local, still same day
		{"Auckland (crosses day line)", "Pacific/Auckland", "2026-04-25"}, // UTC+12, next day
		{"Invalid tz falls back to UTC", "Not/A/Zone", "2026-04-24"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := toLocation(utc, tc.tz).Format("2006-01-02")
			if got != tc.want {
				t.Fatalf("formatLocalDate: got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatLocalTime(t *testing.T) {
	utc, _ := time.Parse(time.RFC3339, "2026-04-24T14:00:00Z")

	cases := []struct {
		name string
		tz   string
		want string
	}{
		{"UTC", "UTC", "14:00"},
		{"empty tz", "", "14:00"},
		{"New York (EDT, -04)", "America/New_York", "10:00"},
		{"Los Angeles (PDT, -07)", "America/Los_Angeles", "07:00"},
		{"Tokyo (+09)", "Asia/Tokyo", "23:00"},
		{"Invalid tz falls back to UTC", "Not/A/Zone", "14:00"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := toLocation(utc, tc.tz).Format("15:04")
			if got != tc.want {
				t.Fatalf("formatLocalTime: got %q, want %q", got, tc.want)
			}
		})
	}
}
