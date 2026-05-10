// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored novel command: cross-site watchlist with thresholds.

package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/store"
)

type watchlistCheckRow struct {
	ID             int64   `json:"id"`
	Source         string  `json:"source,omitempty"`
	SourceID       string  `json:"source_id,omitempty"`
	EAN            string  `json:"ean,omitempty"`
	Label          string  `json:"label,omitempty"`
	ProductName    string  `json:"product_name,omitempty"`
	TargetMaxSEK   float64 `json:"target_max_sek"`
	BestSite       string  `json:"best_site,omitempty"`
	BestPriceSEK   float64 `json:"best_price_sek,omitempty"`
	BestURL        string  `json:"best_url,omitempty"`
	BelowThreshold bool    `json:"below_threshold"`
	Reason         string  `json:"reason,omitempty"`
}

func newWatchlistCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watchlist",
		Short: "Track products by ID or EAN with a maximum price; check whether any tracked item is at or below threshold across either site.",
	}
	cmd.AddCommand(newWatchlistAddCmd(flags))
	cmd.AddCommand(newWatchlistListCmd(flags))
	cmd.AddCommand(newWatchlistRemoveCmd(flags))
	cmd.AddCommand(newWatchlistCheckCmd(flags))
	return cmd
}

func newWatchlistAddCmd(flags *rootFlags) *cobra.Command {
	var (
		flagSource string
		flagID     string
		flagEAN    string
		flagLabel  string
		flagMax    float64
	)
	cmd := &cobra.Command{
		Use:   "add [identifier]",
		Short: "Add a product to the watchlist. Identifier is a Prisjakt int ID, a PriceRunner string ID, or an EAN.",
		Example: "  se-prices-pp-cli watchlist add 14969878 --source prisjakt --max 9990\n" +
			"  se-prices-pp-cli watchlist add 0194253433927 --ean --max 7500 --label \"iphone 15 budget\"",
		Annotations: map[string]string{},
		RunE: func(cmd *cobra.Command, args []string) error {
			ident := ""
			if len(args) > 0 {
				ident = args[0]
			}
			if ident == "" && flagID == "" && flagEAN == "" && !flags.dryRun {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if flagEAN != "" || (ident != "" && looksLikeEAN(ident)) {
				ean := flagEAN
				if ean == "" {
					ean = ident
				}
				st, err := openSEPStore(cmd.Context())
				if err != nil {
					return err
				}
				defer st.Close()
				id, err := st.AddWatched(cmd.Context(), store.SEWatchedItem{EAN: ean, Label: flagLabel, MaxPriceSEK: flagMax})
				if err != nil {
					return err
				}
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"id": id, "ean": ean, "max_price_sek": flagMax}, flags)
			}
			source := strings.ToLower(flagSource)
			if source == "" {
				source = guessSourceForID(ident)
			}
			if source != "prisjakt" && source != "pricerunner" {
				return fmt.Errorf("--source required (prisjakt or pricerunner) when identifier is not an EAN")
			}
			id := flagID
			if id == "" {
				id = ident
			}
			if id == "" {
				return fmt.Errorf("identifier missing")
			}
			st, err := openSEPStore(cmd.Context())
			if err != nil {
				return err
			}
			defer st.Close()
			watchedID, err := st.AddWatched(cmd.Context(), store.SEWatchedItem{Source: source, SourceID: id, Label: flagLabel, MaxPriceSEK: flagMax})
			if err != nil {
				return err
			}
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"id": watchedID, "source": source, "source_id": id, "max_price_sek": flagMax}, flags)
		},
	}
	cmd.Flags().StringVar(&flagSource, "source", "", "Site: prisjakt or pricerunner")
	cmd.Flags().StringVar(&flagID, "id", "", "Product ID on the chosen site")
	cmd.Flags().StringVar(&flagEAN, "ean", "", "Track by EAN/GTIN instead")
	cmd.Flags().StringVar(&flagLabel, "label", "", "Optional human label")
	cmd.Flags().Float64Var(&flagMax, "max", 0, "Maximum price in SEK that triggers `check`")
	return cmd
}

func newWatchlistListCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:         "list",
		Short:       "List all tracked products.",
		Example:     "  se-prices-pp-cli watchlist list\n  se-prices-pp-cli watchlist list --json --select label,max_price_sek",
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
			items, err := st.ListWatched(cmd.Context())
			if err != nil {
				return err
			}
			return printJSONFiltered(cmd.OutOrStdout(), items, flags)
		},
	}
}

func newWatchlistRemoveCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove [id]",
		Short:   "Remove a tracked item by row ID (from `watchlist list`).",
		Example: "  se-prices-pp-cli watchlist remove 3\n  se-prices-pp-cli watchlist list --json --select id  # find IDs first",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !flags.dryRun {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id %q", args[0])
			}
			st, err := openSEPStore(cmd.Context())
			if err != nil {
				return err
			}
			defer st.Close()
			n, err := st.RemoveWatched(cmd.Context(), id)
			if err != nil {
				return err
			}
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"id": id, "removed": n}, flags)
		},
	}
	return cmd
}

func newWatchlistCheckCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run the cross-source lowest query for every watched item and report threshold-met items.",
		Example: "  se-prices-pp-cli watchlist check\n" +
			"  se-prices-pp-cli watchlist check --json --select rows.product_name,rows.best_site,rows.best_price_sek,rows.below_threshold",
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
			items, err := st.ListWatched(cmd.Context())
			if err != nil {
				return err
			}
			rows := make([]watchlistCheckRow, 0, len(items))
			for _, w := range items {
				row := watchlistCheckRow{ID: w.ID, Source: w.Source, SourceID: w.SourceID, EAN: w.EAN, Label: w.Label, TargetMaxSEK: w.MaxPriceSEK}
				var products []store.SEProduct
				if w.EAN != "" {
					products, _ = st.ProductsByEAN(cmd.Context(), w.EAN)
				} else if w.Source != "" && w.SourceID != "" {
					rowsCur, qerr := st.DB().QueryContext(cmd.Context(), `
						SELECT source, source_id, name, brand, category, ean, COALESCE(url, ''), COALESCE(image_url, ''), COALESCE(lowest_price_sek, 0), COALESCE(last_seen_at, '')
						FROM sep_products WHERE source=? AND source_id=?`, w.Source, w.SourceID)
					if qerr == nil {
						for rowsCur.Next() {
							var p store.SEProduct
							if err := rowsCur.Scan(&p.Source, &p.SourceID, &p.Name, &p.Brand, &p.Category, &p.EAN, &p.URL, &p.ImageURL, &p.LowestSEK, &p.LastSeenAt); err == nil {
								products = append(products, p)
							}
						}
						rowsCur.Close()
					}
				}
				if len(products) == 0 {
					row.Reason = "no synced product matched this watched item; run `sync` for the relevant category"
					rows = append(rows, row)
					continue
				}
				for _, p := range products {
					if p.LowestSEK <= 0 {
						continue
					}
					if row.BestPriceSEK == 0 || p.LowestSEK < row.BestPriceSEK {
						row.BestPriceSEK = p.LowestSEK
						row.BestSite = p.Source
						row.BestURL = p.URL
						row.ProductName = p.Name
					}
				}
				if w.MaxPriceSEK > 0 && row.BestPriceSEK > 0 && row.BestPriceSEK <= w.MaxPriceSEK {
					row.BelowThreshold = true
				}
				rows = append(rows, row)
			}
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"rows": rows}, flags)
		},
	}
	return cmd
}

func looksLikeEAN(s string) bool {
	if len(s) < 8 || len(s) > 14 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func guessSourceForID(s string) string {
	// Prisjakt IDs are short numeric (5-9 digits). PriceRunner are long (10+ digits, often).
	// Use a digit-only check; default to empty when ambiguous.
	allDigits := true
	for _, r := range s {
		if r < '0' || r > '9' {
			allDigits = false
			break
		}
	}
	if !allDigits {
		return ""
	}
	if len(s) <= 9 {
		return "prisjakt"
	}
	return "pricerunner"
}
