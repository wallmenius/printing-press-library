package prisjakt

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCategory(t *testing.T) {
	html, err := os.ReadFile("testdata/category.html")
	require.NoError(t, err)
	cat, err := ParseCategory(html)
	require.NoError(t, err)
	assert.Equal(t, "Mobiltelefoner", cat.Name)
	assert.NotEmpty(t, cat.Products, "trendingProducts should populate the listing")
	for _, p := range cat.Products {
		assert.Greater(t, p.ID, 0, "product id should be set")
		assert.NotEmpty(t, p.Name)
		assert.Greater(t, p.LowestPriceSEK, float64(0), "trending products should have prices")
	}
}

func TestParseProduct(t *testing.T) {
	html, err := os.ReadFile("testdata/product.html")
	require.NoError(t, err)
	p, err := ParseProduct(html)
	require.NoError(t, err)
	assert.Greater(t, p.ID, 0)
	assert.NotEmpty(t, p.Name)
	assert.NotNil(t, p.Brand)
	assert.NotNil(t, p.Category)
	assert.NotEmpty(t, p.Offers, "product should have offers")
	for _, o := range p.Offers {
		assert.NotEmpty(t, o.Merchant)
		assert.Greater(t, o.PriceSEK, float64(0))
	}
}

func TestExtractReactQueryState_NoMatch(t *testing.T) {
	_, err := ExtractReactQueryState([]byte("<html><body>just html</body></html>"))
	assert.Error(t, err, "should error on HTML without React Query state")
}

func TestFlexID_Unmarshal(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{`"103"`, "103"},
		{`142`, "142"},
		{`"abc"`, "abc"},
	}
	for _, tc := range cases {
		var f FlexID
		require.NoError(t, f.UnmarshalJSON([]byte(tc.input)))
		assert.Equal(t, tc.expected, string(f), "input: %s", tc.input)
	}
}

func TestDecodeJSStringLiteral(t *testing.T) {
	cases := []struct {
		input, expected string
	}{
		{`hello\nworld`, "hello\nworld"},
		{`\xE4`, "\xE4"},
		{`ä`, "\xc3\xa4"},
		{`escaped \' quote`, `escaped ' quote`},
	}
	for _, tc := range cases {
		got := decodeJSStringLiteral(tc.input)
		assert.Equal(t, tc.expected, got, "input: %s", tc.input)
	}
}
