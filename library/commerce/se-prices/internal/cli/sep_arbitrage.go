// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored novel command: cross-site arbitrage scanner.

package cli

import (
	"sort"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/store"
)

type arbitrageRow struct {
	NormName         string  `json:"-"`
	ProductName      string  `json:"product_name"`
	PrisjaktID       string  `json:"prisjakt_id,omitempty"`
	PrisjaktURL      string  `json:"prisjakt_url,omitempty"`
	PrisjaktPrice    float64 `json:"prisjakt_price,omitempty"`
	PriceRunnerID    string  `json:"pricerunner_id,omitempty"`
	PriceRunnerURL   string  `json:"pricerunner_url,omitempty"`
	PriceRunnerPrice float64 `json:"pricerunner_price,omitempty"`
	GapSEK           float64 `json:"gap_sek"`
	GapPct           float64 `json:"gap_pct"`
	CheaperSite      string  `json:"cheaper_site"`
}

type arbitrageResult struct {
	Category string         `json:"category,omitempty"`
	MinGap   float64        `json:"min_gap_pct"`
	MinSEK   float64        `json:"min_gap_sek"`
	Rows     []arbitrageRow `json:"rows"`
	Reason   string         `json:"reason,omitempty"`
}

func newArbitrageCmd(flags *rootFlags) *cobra.Command {
	var (
		flagCategory string
		flagMinGap   float64
		flagMinSEK   float64
		flagLimit    int
	)
	cmd := &cobra.Command{
		Use:   "arbitrage",
		Short: "Find products where one comparator's lowest beats the other by your gap threshold.",
		Long: "Joins the local store's products from both sites by EAN (preferred) or normalized\n" +
			"product name and computes the percentage gap between Prisjakt's and PriceRunner's\n" +
			"lowest offers. Requires `sync --source both` first so both stores are populated.",
		Example: "  se-prices-pp-cli arbitrage --category mobiltelefoner --min-gap 10\n" +
			"  se-prices-pp-cli arbitrage --category mobiltelefoner --min-gap 10 --json --select rows.product_name,rows.gap_pct,rows.cheaper_site",
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
			var products []store.SEProduct
			if flagCategory != "" {
				products, err = st.AllProductsByCategoryAcrossSources(cmd.Context(), flagCategory)
			} else {
				products, err = st.AllSEProducts(cmd.Context(), 5000)
			}
			if err != nil {
				return err
			}
			result := &arbitrageResult{Category: flagCategory, MinGap: flagMinGap, MinSEK: flagMinSEK}
			result.Rows = computeArbitrage(products, flagMinGap, flagMinSEK)
			sort.Slice(result.Rows, func(i, j int) bool { return result.Rows[i].GapSEK > result.Rows[j].GapSEK })
			if flagLimit > 0 && len(result.Rows) > flagLimit {
				result.Rows = result.Rows[:flagLimit]
			}
			if len(result.Rows) == 0 {
				result.Reason = "no products met the gap threshold; sync more categories or lower --min-gap / --min-sek"
			}
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringVar(&flagCategory, "category", "", "Category slug to scope the scan (e.g., mobiltelefoner)")
	cmd.Flags().Float64Var(&flagMinGap, "min-gap", 10, "Minimum percentage gap to surface")
	cmd.Flags().Float64Var(&flagMinSEK, "min-sek", 0, "Minimum absolute SEK savings to surface")
	cmd.Flags().IntVar(&flagLimit, "limit", 0, "Maximum rows to return")
	return cmd
}

func computeArbitrage(products []store.SEProduct, minGapPct, minSEK float64) []arbitrageRow {
	type bucket struct {
		Name      string
		Prisjakt  *store.SEProduct
		Pricerunn *store.SEProduct
	}
	// Single bucket pool with two index maps that may alias the same bucket.
	// Previously byEAN and byName were independent: a product carrying an EAN
	// landed in byEAN, while another product with no EAN but the same name
	// landed in byName, so both maps could produce a pair for what is
	// conceptually the same product — duplicates in the output. Now whenever
	// a product carries both an EAN and a name, both index entries point to
	// the same bucket, and `seen` deduplicates at collection time so each
	// physical bucket is processed exactly once.
	byEAN := map[string]*bucket{}
	byName := map[string]*bucket{}

	resolveBucket := func(eanKey, nameKey string) *bucket {
		var b *bucket
		if eanKey != "" {
			b = byEAN[eanKey]
		}
		if b == nil && nameKey != "" {
			b = byName[nameKey]
		}
		if b == nil {
			b = &bucket{}
		}
		// Wire both keys to the resolved bucket so the next product with
		// either key joins the same bucket.
		if eanKey != "" {
			byEAN[eanKey] = b
		}
		if nameKey != "" {
			byName[nameKey] = b
		}
		return b
	}

	for i := range products {
		p := &products[i]
		eanKey := ""
		if p.EAN != "" {
			eanKey = "ean:" + p.EAN
		}
		nameKey := ""
		if nk := normalizeName(p.Name); nk != "" {
			nameKey = "name:" + nk
		}
		if eanKey == "" && nameKey == "" {
			continue
		}
		b := resolveBucket(eanKey, nameKey)
		if b.Name == "" {
			b.Name = p.Name
		}
		switch p.Source {
		case "prisjakt":
			if b.Prisjakt == nil || p.LowestSEK > 0 && (b.Prisjakt.LowestSEK == 0 || p.LowestSEK < b.Prisjakt.LowestSEK) {
				b.Prisjakt = p
			}
		case "pricerunner":
			if b.Pricerunn == nil || p.LowestSEK > 0 && (b.Pricerunn.LowestSEK == 0 || p.LowestSEK < b.Pricerunn.LowestSEK) {
				b.Pricerunn = p
			}
		}
	}

	var out []arbitrageRow
	seen := map[*bucket]struct{}{}
	collect := func(buckets map[string]*bucket) {
		for _, b := range buckets {
			if _, dup := seen[b]; dup {
				continue
			}
			seen[b] = struct{}{}
			if b.Prisjakt == nil || b.Pricerunn == nil {
				continue
			}
			pj, pr := b.Prisjakt.LowestSEK, b.Pricerunn.LowestSEK
			if pj <= 0 || pr <= 0 {
				continue
			}
			row := arbitrageRow{
				ProductName:      b.Prisjakt.Name,
				PrisjaktID:       b.Prisjakt.SourceID,
				PrisjaktURL:      b.Prisjakt.URL,
				PrisjaktPrice:    pj,
				PriceRunnerID:    b.Pricerunn.SourceID,
				PriceRunnerURL:   b.Pricerunn.URL,
				PriceRunnerPrice: pr,
			}
			if pj < pr {
				row.GapSEK = pr - pj
				row.GapPct = (pr - pj) / pr * 100
				row.CheaperSite = "prisjakt"
			} else if pr < pj {
				row.GapSEK = pj - pr
				row.GapPct = (pj - pr) / pj * 100
				row.CheaperSite = "pricerunner"
			} else {
				continue
			}
			if row.GapPct < minGapPct {
				continue
			}
			if minSEK > 0 && row.GapSEK < minSEK {
				continue
			}
			out = append(out, row)
		}
	}
	collect(byEAN)
	collect(byName)
	return out
}
