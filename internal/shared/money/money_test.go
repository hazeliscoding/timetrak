package money

import "testing"

func TestDurationBillableIntegerMath(t *testing.T) {
	// 1 hour at 12500 minor units = 12500.
	if got := DurationBillable(3600, 12500); got != 12500 {
		t.Fatalf("1h: got %d", got)
	}
	// 30 minutes at 12500 = 6250.
	if got := DurationBillable(1800, 12500); got != 6250 {
		t.Fatalf("30m: got %d", got)
	}
	// 59 seconds at 3600/hr = 0 (floor-divide).
	if got := DurationBillable(59, 3600); got != 59 {
		// 59 * 3600 / 3600 = 59 actually; different sanity check:
		if got != 59 {
			t.Fatalf("59s at 3600/hr: got %d", got)
		}
	}
}

func TestAmountFormatDefault(t *testing.T) {
	a, err := New(12550, "usd")
	if err != nil {
		t.Fatal(err)
	}
	if got := a.Format(); got != "125.50 USD" {
		t.Fatalf("format: %q", got)
	}
	a2, _ := New(100, "JPY")
	if got := a2.Format(); got != "100 JPY" {
		t.Fatalf("JPY format: %q", got)
	}
}
