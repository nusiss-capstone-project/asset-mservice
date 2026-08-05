package util

import "testing"

func TestToMinorUnits(t *testing.T) {
	got, err := ToMinorUnits("50.00", "USD")
	if err != nil {
		t.Fatal(err)
	}
	if got != 5000 {
		t.Fatalf("got %d want 5000", got)
	}
	got, err = ToMinorUnits("100", "JPY")
	if err != nil {
		t.Fatal(err)
	}
	if got != 100 {
		t.Fatalf("got %d want 100", got)
	}
}
