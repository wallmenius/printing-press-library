package pricerunner

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSearch(t *testing.T) {
	html, err := os.ReadFile("testdata/search.html")
	require.NoError(t, err)
	sr, err := ParseSearch(html, "iphone 15")
	require.NoError(t, err)
	assert.Equal(t, "iphone 15", sr.Query)
	assert.NotEmpty(t, sr.Products, "search results should not be empty")
	for _, p := range sr.Products {
		assert.NotEmpty(t, p.ID)
		assert.NotEmpty(t, p.Name)
		assert.NotEmpty(t, p.URL)
	}
}

func TestParseProduct(t *testing.T) {
	html, err := os.ReadFile("testdata/product.html")
	require.NoError(t, err)
	p, err := ParseProduct(html)
	require.NoError(t, err)
	assert.NotEmpty(t, p.ID)
	assert.NotEmpty(t, p.Name)
	assert.NotEmpty(t, p.Offers, "product should have offers")
	for _, o := range p.Offers {
		assert.NotEmpty(t, o.Merchant)
		assert.Greater(t, o.PriceSEK, float64(0))
	}
}

func TestExtractInitialPayload_NoMatch(t *testing.T) {
	_, err := ExtractInitialPayload([]byte("<html><body>nothing</body></html>"))
	assert.Error(t, err)
}

func TestMoneyFloat(t *testing.T) {
	cases := []struct {
		amount string
		want   float64
	}{
		{"7245.00", 7245.0},
		{"100", 100.0},
		{"", 0.0},
		{"abc", 0.0},
	}
	for _, tc := range cases {
		m := money{Amount: tc.amount, Currency: "SEK"}
		assert.Equal(t, tc.want, m.Float(), "amount: %s", tc.amount)
	}
}

func TestParseDeals_NoMatch(t *testing.T) {
	// /deals page payload may be missing in test fixtures; verify graceful empty return
	_, err := ParseDeals([]byte("<html></html>"))
	assert.NoError(t, err, "should not error on empty HTML")
}

func TestExtractFallbackLinks(t *testing.T) {
	html := []byte(`<html><body>
<a href="/pl/1-3208336567/Mobiltelefoner/Apple-iPhone-15-Pro-Max-256GB-Natural-Titanium-priser">Apple iPhone 15 Pro Max</a>
<a href="/pl/1-3208336570/Mobiltelefoner/Apple-iPhone-15-128GB-Blue-priser">Apple iPhone 15 128GB Blue</a>
</body></html>`)
	got := extractFallbackLinks(html)
	require.Len(t, got, 2)
	assert.Equal(t, "3208336567", got[0].ID)
	assert.Contains(t, got[0].Name, "iPhone 15 Pro Max")
	assert.Equal(t, HostPriceRunner+"/pl/1-3208336567/Mobiltelefoner/Apple-iPhone-15-Pro-Max-256GB-Natural-Titanium-priser", got[0].URL)
}
