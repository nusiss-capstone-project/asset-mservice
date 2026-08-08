package market

import (
	"fmt"
	"strings"
)

// CurrenciesForMarket returns USD + local fiat currencies supported for a market.
func CurrenciesForMarket(market string) ([]string, error) {
	market = strings.ToUpper(strings.TrimSpace(market))
	if market == "" {
		return nil, fmt.Errorf("market is required")
	}
	if currencies, ok := marketCurrencies[market]; ok {
		out := make([]string, len(currencies))
		copy(out, currencies)
		return out, nil
	}
	// Known-but-unmapped markets fall back to USD only.
	return []string{"USD"}, nil
}

func SupportsCurrency(market, currency string) bool {
	currencies, err := CurrenciesForMarket(market)
	if err != nil {
		return false
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	for _, c := range currencies {
		if c == currency {
			return true
		}
	}
	return false
}

var marketCurrencies = map[string][]string{
	"SG":  {"USD", "SGD"},
	"HK":  {"USD", "HKD"},
	"US":  {"USD"},
	"EEA": {"USD", "EUR"},
	"UK":  {"USD", "GBP"},
	"AU":  {"USD", "AUD"},
	"JP":  {"USD", "JPY"},
}
