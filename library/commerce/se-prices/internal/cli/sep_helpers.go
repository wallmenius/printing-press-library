// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: shared helpers for the cross-site novel commands.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/client"
	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/source/pricerunner"
	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/source/prisjakt"
	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/store"
)

// sepDBPath returns the SQLite path used by all novel commands. Mirrors what
// the generated sync command writes to.
func sepDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "se-prices-pp-cli.db"
	}
	return filepath.Join(home, ".cache", "se-prices-pp-cli", "se-prices-pp-cli.db")
}

// openSEPStore opens the local store, creating its directory if needed.
func openSEPStore(ctx context.Context) (*store.Store, error) {
	path := sepDBPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating store directory: %w", err)
	}
	st, err := store.OpenWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("opening store: %w", err)
	}
	if err := st.EnsureSEPricesSchema(ctx); err != nil {
		_ = st.Close()
		return nil, err
	}
	return st, nil
}

// printJSON writes a value as pretty JSON honoring --select / --compact / etc.
func printJSON(w io.Writer, v any, flags *rootFlags) error {
	return printJSONFiltered(w, v, flags)
}

// fetchPrisjaktSearch hits Prisjakt's /search and returns a SearchListing.
func fetchPrisjaktSearch(c *client.Client, query string) (*prisjakt.SearchListing, error) {
	html, err := c.Get("/search", map[string]string{"search": query})
	if err != nil {
		return nil, err
	}
	return prisjakt.ParseSearch(html, query)
}

// fetchPriceRunnerSearch hits PriceRunner's /results and returns a SearchListing.
func fetchPriceRunnerSearch(c *client.Client, query string) (*pricerunner.SearchListing, error) {
	html, err := c.Get("https://www.pricerunner.se/results", map[string]string{"q": query})
	if err != nil {
		return nil, err
	}
	return pricerunner.ParseSearch(html, query)
}

// normalizeName lowercases and strips punctuation for cross-site title matching.
func normalizeName(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == ' ' || r == '\t' {
			b.WriteByte(' ')
		}
	}
	parts := strings.Fields(b.String())
	return strings.Join(parts, " ")
}

// JSONClone returns a deep copy of v via JSON round-trip. Useful for stripping
// type metadata so output renders cleanly.
func jsonClone(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}
