package cli

import (
	"testing"

	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/store"
)

// TestComputeArbitrage_NoDuplicateOnEANAndNameOverlap pins the fix for the
// Greptile finding on sep_arbitrage.go:132. Before the fix, two independent
// maps (byEAN, byName) could each emit a pair for what is conceptually the
// same product when one source carried an EAN and the other did not. The
// product appeared twice in the result. This test arranges exactly that
// shape and asserts the dedup leaves one row.
func TestComputeArbitrage_NoDuplicateOnEANAndNameOverlap(t *testing.T) {
	products := []store.SEProduct{
		// Prisjakt entry carrying both an EAN and a name.
		{Source: "prisjakt", SourceID: "10001", Name: "Apple iPhone 15 128GB", EAN: "0194253433927", LowestSEK: 7900},
		// PriceRunner entry for the same product, ALSO carrying both keys.
		{Source: "pricerunner", SourceID: "1-3208336567", Name: "Apple iPhone 15 128GB", EAN: "0194253433927", LowestSEK: 7245},
		// Another PriceRunner entry with the same name but no EAN — this is
		// the case that previously triggered duplicate emission via byName.
		{Source: "pricerunner", SourceID: "1-9999999999", Name: "Apple iPhone 15 128GB", EAN: "", LowestSEK: 7600},
	}
	rows := computeArbitrage(products, 1, 0)
	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 dedup'd arbitrage row, got %d: %+v", len(rows), rows)
	}
	r := rows[0]
	if r.PrisjaktID != "10001" {
		t.Errorf("expected prisjakt id 10001, got %q", r.PrisjaktID)
	}
	if r.PrisjaktPrice != 7900 || r.PriceRunnerPrice != 7245 {
		t.Errorf("expected pj=7900 pr=7245, got pj=%v pr=%v", r.PrisjaktPrice, r.PriceRunnerPrice)
	}
	if r.CheaperSite != "pricerunner" {
		t.Errorf("expected cheaper_site=pricerunner, got %q", r.CheaperSite)
	}
}

// TestComputeArbitrage_NameOnlyMatchStillWorks ensures the name-fallback
// path still produces a pair when neither product carries an EAN. The dedup
// fix must not regress this case.
func TestComputeArbitrage_NameOnlyMatchStillWorks(t *testing.T) {
	products := []store.SEProduct{
		{Source: "prisjakt", SourceID: "20001", Name: "Sony WH-1000XM5", LowestSEK: 3199},
		{Source: "pricerunner", SourceID: "1-2222222222", Name: "Sony WH-1000XM5", LowestSEK: 2899},
	}
	rows := computeArbitrage(products, 1, 0)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row from name-only match, got %d", len(rows))
	}
}

// TestComputeArbitrage_ThresholdsHonored asserts both --min-gap and --min-sek
// still filter correctly after the dedup refactor.
func TestComputeArbitrage_ThresholdsHonored(t *testing.T) {
	products := []store.SEProduct{
		{Source: "prisjakt", SourceID: "30001", Name: "tiny gap product", LowestSEK: 1000},
		{Source: "pricerunner", SourceID: "1-3333333333", Name: "tiny gap product", LowestSEK: 990},
	}
	if rows := computeArbitrage(products, 5, 0); len(rows) != 0 {
		t.Errorf("expected min-gap=5 to filter out 1%% gap, got %d rows", len(rows))
	}
	if rows := computeArbitrage(products, 0.5, 0); len(rows) != 1 {
		t.Errorf("expected min-gap=0.5 to surface 1%% gap, got %d rows", len(rows))
	}
	if rows := computeArbitrage(products, 0.5, 100); len(rows) != 0 {
		t.Errorf("expected min-sek=100 to filter out 10 SEK gap, got %d rows", len(rows))
	}
}
