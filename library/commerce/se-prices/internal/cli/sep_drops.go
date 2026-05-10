// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored novel command: cross-site price-drop digest from snapshots.

package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/store"
)

type dropRow struct {
	Source         string  `json:"source"`
	SourceID       string  `json:"source_id"`
	Name           string  `json:"name,omitempty"`
	URL            string  `json:"url,omitempty"`
	StartPriceSEK  float64 `json:"start_price_sek"`
	LatestPriceSEK float64 `json:"latest_price_sek"`
	DropPct        float64 `json:"drop_pct"`
	DropSEK        float64 `json:"drop_sek"`
	StartedAt      string  `json:"started_at"`
	LatestAt       string  `json:"latest_at"`
}

type dropsResult struct {
	Since       string    `json:"since"`
	MinPct      float64   `json:"min_pct"`
	WatchedOnly bool      `json:"watched_only"`
	Rows        []dropRow `json:"rows"`
	Reason      string    `json:"reason,omitempty"`
}

func newDropsCmd(flags *rootFlags) *cobra.Command {
	var (
		flagSince       string
		flagMinPct      float64
		flagWatchedOnly bool
		flagLimit       int
	)
	cmd := &cobra.Command{
		Use:   "drops",
		Short: "Watched products whose latest snapshot is at least your percentage threshold below the snapshot at the start of the window.",
		Example: "  se-prices-pp-cli drops --since 7d --min-pct 10\n" +
			"  se-prices-pp-cli drops --since 30d --watched-only --json --select rows.name,rows.drop_pct",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			st, err := openSEPStore(cmd.Context())
			if err != nil {
				return err
			}
			defer st.Close()
			windowStart, err := parseSinceWindow(flagSince)
			if err != nil {
				return err
			}
			result := &dropsResult{Since: flagSince, MinPct: flagMinPct, WatchedOnly: flagWatchedOnly}
			result.Rows, err = computeDrops(cmd.Context(), st, windowStart, flagMinPct, flagWatchedOnly)
			if err != nil {
				return err
			}
			sort.Slice(result.Rows, func(i, j int) bool { return result.Rows[i].DropPct > result.Rows[j].DropPct })
			if flagLimit > 0 && len(result.Rows) > flagLimit {
				result.Rows = result.Rows[:flagLimit]
			}
			if len(result.Rows) == 0 {
				result.Reason = "no price drops in window; need at least two snapshots per product (run `sync --watched-only` periodically)"
			}
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringVar(&flagSince, "since", "7d", "Window length: <N>d, <N>h, or <N>m")
	cmd.Flags().Float64Var(&flagMinPct, "min-pct", 5, "Minimum percentage drop to surface")
	cmd.Flags().BoolVar(&flagWatchedOnly, "watched-only", false, "Restrict to products on the watchlist")
	cmd.Flags().IntVar(&flagLimit, "limit", 0, "Maximum rows to return")
	return cmd
}

func parseSinceWindow(s string) (time.Time, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return time.Now().Add(-7 * 24 * time.Hour), nil
	}
	var n int
	var unit byte
	if _, err := fmt.Sscanf(s, "%d%c", &n, &unit); err != nil {
		return time.Time{}, fmt.Errorf("invalid --since %q (expect form like 7d, 12h, 30m)", s)
	}
	now := time.Now()
	switch unit {
	case 'd':
		return now.Add(-time.Duration(n) * 24 * time.Hour), nil
	case 'h':
		return now.Add(-time.Duration(n) * time.Hour), nil
	case 'm':
		return now.Add(-time.Duration(n) * time.Minute), nil
	default:
		return time.Time{}, fmt.Errorf("invalid --since unit %q (use d, h, or m)", string(unit))
	}
}

func computeDrops(ctx context.Context, st *store.Store, windowStart time.Time, minPct float64, watchedOnly bool) ([]dropRow, error) {
	keys, err := allProductKeys(ctx, st, watchedOnly)
	if err != nil {
		return nil, err
	}
	var rows []dropRow
	for _, k := range keys {
		snaps, err := st.SnapshotsForProduct(ctx, k.Source, k.SourceID, windowStart)
		if err != nil {
			return nil, err
		}
		if len(snaps) < 2 {
			continue
		}
		start := snaps[0]
		latest := snaps[len(snaps)-1]
		if start.LowestSEK <= 0 || latest.LowestSEK <= 0 {
			continue
		}
		if latest.LowestSEK >= start.LowestSEK {
			continue
		}
		drop := start.LowestSEK - latest.LowestSEK
		pct := drop / start.LowestSEK * 100
		if pct < minPct {
			continue
		}
		row := dropRow{
			Source: k.Source, SourceID: k.SourceID, Name: k.Name, URL: k.URL,
			StartPriceSEK: start.LowestSEK, LatestPriceSEK: latest.LowestSEK,
			DropPct: pct, DropSEK: drop, StartedAt: start.TakenAt, LatestAt: latest.TakenAt,
		}
		rows = append(rows, row)
	}
	return rows, nil
}

type productKey struct {
	Source, SourceID, Name, URL string
}

func allProductKeys(ctx context.Context, st *store.Store, watchedOnly bool) ([]productKey, error) {
	if watchedOnly {
		watched, err := st.ListWatched(ctx)
		if err != nil {
			return nil, err
		}
		var out []productKey
		for _, w := range watched {
			if w.Source == "" || w.SourceID == "" {
				if w.EAN == "" {
					continue
				}
				prods, err := st.ProductsByEAN(ctx, w.EAN)
				if err != nil {
					return nil, err
				}
				for _, p := range prods {
					out = append(out, productKey{p.Source, p.SourceID, p.Name, p.URL})
				}
				continue
			}
			out = append(out, productKey{w.Source, w.SourceID, w.Label, ""})
		}
		return out, nil
	}
	rows, err := st.DB().QueryContext(ctx, `SELECT source, source_id, name, COALESCE(url, '') FROM sep_products`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []productKey
	for rows.Next() {
		var k productKey
		if err := rows.Scan(&k.Source, &k.SourceID, &k.Name, &k.URL); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}
