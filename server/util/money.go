package util

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// ToMinorUnits converts a decimal amount string to the smallest currency unit (e.g. cents).
func ToMinorUnits(amount, currency string) (int64, error) {
	d, err := decimal.NewFromString(strings.TrimSpace(amount))
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q: %w", amount, err)
	}
	if d.IsNegative() || d.IsZero() {
		return 0, fmt.Errorf("amount must be positive")
	}
	exp := currencyExponent(currency)
	scaled := d.Shift(int32(exp))
	if !scaled.Equal(scaled.Truncate(0)) {
		return 0, fmt.Errorf("amount has too many decimal places for currency %s", currency)
	}
	return scaled.IntPart(), nil
}

func currencyExponent(currency string) int {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "JPY", "KRW":
		return 0
	default:
		return 2
	}
}
