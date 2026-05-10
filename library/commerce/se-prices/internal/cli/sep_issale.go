// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored novel command: Black-Week sale-anomaly check.

package cli

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

type isSaleResult struct {
	EAN             string  `json:"ean,omitempty"`
	Source          string  `json:"source,omitempty"`
	SourceID        string  `json:"source_id,omitempty"`
	WindowDays      int     `json:"window_days"`
	CurrentPriceSEK float64 `json:"current_price_sek"`
	WindowMedianSEK float64 `json:"window_median_sek"`
	WindowMinSEK    float64 `json:"window_min_sek"`
	WindowMaxSEK    float64 `json:"window_max_sek"`
	ActuallyASale   bool    `json:"actually_a_sale"`
	SamplesInWindow int     `json:"samples_in_window"`
	Reason          string  `json:"reason,omitempty"`
}

func newIsSaleCmd(flags *rootFlags) *cobra.Command {
	var (
		flagEAN    string
		flagID     string
		flagSource string
		flagWindow int
	)
	cmd := &cobra.Command{
		Use:   "is-sale [ean-or-id]",
		Short: "Compare current price against the local 90-day median to flag sticker-stuffed sales.",
		Long: "Pure local stat over the price-snapshot table: returns the current price, the\n" +
			"window's median, min, and max, and a boolean `actually_a_sale = current < median * 0.9`.\n" +
			"Run `sync` periodically to build enough history for this to be meaningful.",
		Example: "  se-prices-pp-cli is-sale 14969878 --source prisjakt --window 90\n" +
			"  se-prices-pp-cli is-sale --ean 0194253433927 --window 60",
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
			windowStart := time.Now().AddDate(0, 0, -flagWindow)
			result := &isSaleResult{EAN: flagEAN, Source: flagSource, SourceID: flagID, WindowDays: flagWindow}

			var keys []productKey
			if flagEAN != "" {
				prods, _ := st.ProductsByEAN(cmd.Context(), flagEAN)
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
			var prices []float64
			var latest float64
			var latestAt string
			for _, k := range keys {
				snaps, _ := st.SnapshotsForProduct(cmd.Context(), k.Source, k.SourceID, windowStart)
				for _, s := range snaps {
					if s.LowestSEK > 0 {
						prices = append(prices, s.LowestSEK)
					}
					if s.TakenAt > latestAt {
						latestAt = s.TakenAt
						latest = s.LowestSEK
					}
				}
			}
			result.SamplesInWindow = len(prices)
			result.CurrentPriceSEK = latest
			if len(prices) == 0 {
				result.Reason = "no snapshots in window for this product; run `sync` periodically to build history"
				return printJSONFiltered(cmd.OutOrStdout(), result, flags)
			}
			sort.Float64s(prices)
			result.WindowMinSEK = prices[0]
			result.WindowMaxSEK = prices[len(prices)-1]
			if len(prices)%2 == 1 {
				result.WindowMedianSEK = prices[len(prices)/2]
			} else {
				result.WindowMedianSEK = (prices[len(prices)/2-1] + prices[len(prices)/2]) / 2
			}
			result.ActuallyASale = result.WindowMedianSEK > 0 && result.CurrentPriceSEK < result.WindowMedianSEK*0.9
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringVar(&flagEAN, "ean", "", "Product EAN/GTIN")
	cmd.Flags().StringVar(&flagID, "id", "", "Single-site product ID")
	cmd.Flags().StringVar(&flagSource, "source", "", "Site if --id is given")
	cmd.Flags().IntVar(&flagWindow, "window", 90, "Window length in days")
	return cmd
}
