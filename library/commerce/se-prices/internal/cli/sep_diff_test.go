package cli

import (
	"testing"

	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/store"
)

// TestComputeCatalogueDiff_RejectsUnknownOnly pins the fix for the silent
// fallback finding (Greptile P1 on sep_diff.go:70). The cobra RunE rejects
// unknown --only values before calling this helper, but the helper itself
// must not silently default to "prisjakt" if a caller skips that gate;
// returning nil makes the misuse loud.
func TestComputeCatalogueDiff_RejectsUnknownOnly(t *testing.T) {
	products := []store.SEProduct{
		{Source: "prisjakt", Name: "Apple iPhone 15"},
		{Source: "pricerunner", Name: "Samsung Galaxy S26"},
	}
	for _, only := range []string{"", "amazon", "PRISJAKT", "prisjakt1", " prisjakt"} {
		if got := computeCatalogueDiff(products, only); got != nil {
			t.Errorf("computeCatalogueDiff(%q) = %d rows, want nil (unknown source)", only, len(got))
		}
	}
}

// TestComputeCatalogueDiff_HappyPath confirms the dedup logic still works
// end-to-end after the validation tightening.
func TestComputeCatalogueDiff_HappyPath(t *testing.T) {
	products := []store.SEProduct{
		{Source: "prisjakt", Name: "Apple iPhone 15", EAN: "0194253433927"},
		{Source: "pricerunner", Name: "Apple iPhone 15", EAN: "0194253433927"},
		{Source: "prisjakt", Name: "Niche Swedish Brand X"}, // only on prisjakt
		{Source: "pricerunner", Name: "Klarna Exclusive Y"}, // only on pricerunner
	}
	got := computeCatalogueDiff(products, "prisjakt")
	if len(got) != 1 || got[0].Name != "Niche Swedish Brand X" {
		t.Errorf("only=prisjakt: expected just 'Niche Swedish Brand X', got %+v", got)
	}
	got = computeCatalogueDiff(products, "pricerunner")
	if len(got) != 1 || got[0].Name != "Klarna Exclusive Y" {
		t.Errorf("only=pricerunner: expected just 'Klarna Exclusive Y', got %+v", got)
	}
}

// TestLooksLikeEAN exercises the GTIN length guard. The watchlist add command
// no longer auto-detects EANs from positionals (--ean must be explicit), but
// the helper is still used to validate that an --ean value is plausibly a
// barcode rather than free text.
func TestLooksLikeEAN(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		// Real GTIN lengths.
		{"12345678", true},      // GTIN-8
		{"012345678901", true},  // GTIN-12 (UPC-A)
		{"1234567890123", true}, // GTIN-13 (EAN-13)
		{"12345678901234", true},
		// Too short or too long.
		{"1234567", false},
		{"123456789012345", false},
		{"", false},
		// Non-digits.
		{"123abc4567", false},
		{"1-3208336567", false},
	}
	for _, tc := range cases {
		if got := looksLikeEAN(tc.in); got != tc.want {
			t.Errorf("looksLikeEAN(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
