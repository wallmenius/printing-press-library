// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored novel command: cross-site unified price history.

package cli

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/store"
)

func isNumericID(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

type historyEntry struct {
	Source     string  `json:"source"`
	SourceID   string  `json:"source_id"`
	TakenAt    string  `json:"taken_at"`
	LowestSEK  float64 `json:"lowest_price_sek"`
	OfferCount int     `json:"offer_count,omitempty"`
}

type historyResult struct {
	EAN     string         `json:"ean,omitempty"`
	ID      string         `json:"id,omitempty"`
	Source  string         `json:"source,omitempty"`
	WindowD int            `json:"window_days"`
	Series  []historyEntry `json:"series"`
	Reason  string         `json:"reason,omitempty"`
}

func newHistoryComboCmd(flags *rootFlags) *cobra.Command {
	var (
		flagEAN    string
		flagID     string
		flagSource string
		flagWindow int
	)
	cmd := &cobra.Command{
		Use:   "history-combo [ean-or-id]",
		Short: "Merged time series of price snapshots from both sources for one product, resolved via EAN.",
		Example: "  se-prices-pp-cli history-combo --ean 0194253433927\n" +
			"  se-prices-pp-cli history-combo --id 14969878 --source prisjakt --window 60\n" +
			"  # window is in days as an integer",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if looksLikeEAN(args[0]) {
					flagEAN = args[0]
				} else {
					flagID = args[0]
				}
			}
			if flagEAN == "" && flagID == "" && !flags.dryRun {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if flagEAN == "" && !isNumericID(flagID) {
				return fmt.Errorf("invalid id %q: expected numeric product ID, or pass --ean", flagID)
			}
			st, err := openSEPStore(cmd.Context())
			if err != nil {
				return err
			}
			defer st.Close()
			windowStart, _ := parseSinceWindow(toDayString(flagWindow))
			result := &historyResult{EAN: flagEAN, ID: flagID, Source: flagSource, WindowD: flagWindow}

			var keys []productKey
			if flagEAN != "" {
				prods, err := st.ProductsByEAN(cmd.Context(), flagEAN)
				if err != nil {
					return err
				}
				for _, p := range prods {
					keys = append(keys, productKey{p.Source, p.SourceID, p.Name, p.URL})
				}
			} else if flagID != "" {
				if flagSource == "" {
					flagSource = guessSourceForID(flagID)
				}
				if flagSource != "" {
					keys = append(keys, productKey{flagSource, flagID, "", ""})
				}
			}
			result.Series = mergeSnapshots(cmd.Context(), st, keys, windowStart)
			if len(result.Series) == 0 {
				result.Reason = "no snapshots in window for the given product; run `sync` periodically to build history"
			}
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringVar(&flagEAN, "ean", "", "Product EAN/GTIN (preferred)")
	cmd.Flags().StringVar(&flagID, "id", "", "Single-site product ID (use --source to disambiguate)")
	cmd.Flags().StringVar(&flagSource, "source", "", "Site if --id is given: prisjakt or pricerunner")
	cmd.Flags().IntVar(&flagWindow, "window", 90, "Window length in days")
	return cmd
}

func toDayString(days int) string {
	if days <= 0 {
		return "90d"
	}
	return itoa(days) + "d"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func mergeSnapshots(ctx context.Context, st *store.Store, keys []productKey, since time.Time) []historyEntry {
	var out []historyEntry
	for _, k := range keys {
		snaps, err := st.SnapshotsForProduct(ctx, k.Source, k.SourceID, since)
		if err != nil {
			continue
		}
		for _, s := range snaps {
			out = append(out, historyEntry{
				Source: s.Source, SourceID: s.SourceID,
				TakenAt: s.TakenAt, LowestSEK: s.LowestSEK, OfferCount: s.OfferCount,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TakenAt < out[j].TakenAt })
	return out
}
