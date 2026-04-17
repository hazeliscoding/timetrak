// Package money represents monetary amounts as integer minor units.
// Floats are never used for money anywhere in TimeTrak.
package money

import (
	"errors"
	"fmt"
	"strings"
)

// Amount is a monetary value expressed in the minor unit of its currency
// (e.g. USD cents, EUR cents, JPY yen).
type Amount struct {
	MinorUnits   int64  // Can be negative for credits.
	CurrencyCode string // ISO 4217, three uppercase letters.
}

// ErrInvalidCurrency indicates the currency code is not a 3-letter ISO 4217 code.
var ErrInvalidCurrency = errors.New("money: currency code must be 3 uppercase letters")

// New constructs an Amount after validating the currency code.
func New(minor int64, currency string) (Amount, error) {
	c := strings.ToUpper(strings.TrimSpace(currency))
	if len(c) != 3 {
		return Amount{}, ErrInvalidCurrency
	}
	for _, r := range c {
		if r < 'A' || r > 'Z' {
			return Amount{}, ErrInvalidCurrency
		}
	}
	return Amount{MinorUnits: minor, CurrencyCode: c}, nil
}

// fractionDigits returns the number of decimals the given currency typically uses.
// Extend as needed; defaults to 2 which is correct for most.
func fractionDigits(code string) int {
	switch strings.ToUpper(code) {
	case "JPY", "KRW", "VND", "CLP", "ISK", "HUF":
		return 0
	case "BHD", "IQD", "JOD", "KWD", "LYD", "OMR", "TND":
		return 3
	default:
		return 2
	}
}

// Format renders the amount for display as a decimal string followed by the
// currency code, e.g. "125.50 USD". It never uses floats.
func (a Amount) Format() string {
	digits := fractionDigits(a.CurrencyCode)
	if digits == 0 {
		return fmt.Sprintf("%d %s", a.MinorUnits, a.CurrencyCode)
	}
	neg := a.MinorUnits < 0
	v := a.MinorUnits
	if neg {
		v = -v
	}
	var divisor int64 = 1
	for i := 0; i < digits; i++ {
		divisor *= 10
	}
	whole := v / divisor
	frac := v % divisor
	sign := ""
	if neg {
		sign = "-"
	}
	return fmt.Sprintf("%s%d.%0*d %s", sign, whole, digits, frac, a.CurrencyCode)
}

// DurationBillable computes minor-units owed for a duration in seconds
// at a given hourly rate in minor units. Uses integer math only:
//
//	(seconds * rate) / 3600
//
// Truncates toward zero (matches plain integer division for non-negative inputs).
func DurationBillable(seconds int64, hourlyRateMinor int64) int64 {
	return (seconds * hourlyRateMinor) / 3600
}
