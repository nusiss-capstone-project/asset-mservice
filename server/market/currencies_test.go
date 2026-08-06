package market

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCurrenciesForMarket(t *testing.T) {
	got, err := CurrenciesForMarket("sg")
	require.NoError(t, err)
	require.Equal(t, []string{"USD", "SGD"}, got)

	got, err = CurrenciesForMarket("XX")
	require.NoError(t, err)
	require.Equal(t, []string{"USD"}, got)

	_, err = CurrenciesForMarket("")
	require.ErrorContains(t, err, "market is required")
}

func TestSupportsCurrency(t *testing.T) {
	require.True(t, SupportsCurrency("HK", "hkd"))
	require.False(t, SupportsCurrency("US", "SGD"))
	require.False(t, SupportsCurrency("", "USD"))
}
