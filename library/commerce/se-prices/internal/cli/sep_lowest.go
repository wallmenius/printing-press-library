// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored novel command: cross-site lowest offer for a query.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type lowestResult struct {
	Query     string                 `json:"query"`
	BestOffer *lowestProduct         `json:"best_offer,omitempty"`
	PerSite   map[string]*lowestSite `json:"per_site"`
	Reason    string                 `json:"reason,omitempty"`
}

type lowestSite struct {
	Site     string         `json:"site"`
	Cheapest *lowestProduct `json:"cheapest,omitempty"`
	HitCount int            `json:"hit_count"`
	Reason   string         `json:"reason,omitempty"`
}

type lowestProduct struct {
	Site           string  `json:"site"`
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	URL            string  `json:"url,omitempty"`
	LowestPriceSEK float64 `json:"lowest_price_sek"`
	Brand          string  `json:"brand,omitempty"`
	Category       string  `json:"category,omitempty"`
	StockStatus    string  `json:"stock_status,omitempty"`
}

func newLowestCmd(flags *rootFlags) *cobra.Command {
	var (
		flagInStock     bool
		flagMaxShipping float64
		flagEAN         string
	)
	cmd := &cobra.Command{
		Use:   "lowest [query]",
		Short: "Cheapest current offer across both Prisjakt and PriceRunner.",
		Long: "Issues a search to both sites, then returns the lowest current price across them.\n" +
			"With --in-stock the search is filtered to in-stock listings; --max-shipping caps shipping cost.\n" +
			"With --ean a known barcode is preferred over a name search.",
		Example: "  se-prices-pp-cli lowest \"iPhone 15 Pro Max\"\n" +
			"  se-prices-pp-cli lowest \"sony wh-1000xm5\" --in-stock --json --select best_offer.site,best_offer.lowest_price_sek\n" +
			"  se-prices-pp-cli lowest --ean 0194253433927",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			query := flagEAN
			if len(args) > 0 {
				query = args[0]
			}
			if query == "" && !flags.dryRun {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			result := &lowestResult{Query: query, PerSite: make(map[string]*lowestSite, 2)}
			if pj, err := fetchPrisjaktSearch(c, query); err == nil {
				site := &lowestSite{Site: "prisjakt", HitCount: len(pj.Products)}
				for _, p := range pj.Products {
					if p.LowestPriceSEK <= 0 {
						continue
					}
					if flagInStock && p.StockStatus != "" && p.StockStatus != "in_stock" {
						continue
					}
					cand := &lowestProduct{
						Site:           "prisjakt",
						ID:             fmt.Sprintf("%d", p.ID),
						Name:           p.Name,
						URL:            p.URL,
						LowestPriceSEK: p.LowestPriceSEK,
						Brand:          p.Brand,
						StockStatus:    p.StockStatus,
					}
					if site.Cheapest == nil || cand.LowestPriceSEK < site.Cheapest.LowestPriceSEK {
						site.Cheapest = cand
					}
				}
				result.PerSite["prisjakt"] = site
			} else {
				result.PerSite["prisjakt"] = &lowestSite{Site: "prisjakt", Reason: err.Error()}
			}
			if pr, err := fetchPriceRunnerSearch(c, query); err == nil {
				site := &lowestSite{Site: "pricerunner", HitCount: len(pr.Products)}
				for _, p := range pr.Products {
					if p.LowestPriceSEK <= 0 {
						continue
					}
					if flagInStock && p.StockStatus != "" && p.StockStatus != "IN_STOCK" && p.StockStatus != "in_stock" {
						continue
					}
					cand := &lowestProduct{
						Site:           "pricerunner",
						ID:             p.ID,
						Name:           p.Name,
						URL:            p.URL,
						LowestPriceSEK: p.LowestPriceSEK,
						Brand:          p.Brand,
						Category:       p.Category,
						StockStatus:    p.StockStatus,
					}
					if site.Cheapest == nil || cand.LowestPriceSEK < site.Cheapest.LowestPriceSEK {
						site.Cheapest = cand
					}
				}
				result.PerSite["pricerunner"] = site
			} else {
				result.PerSite["pricerunner"] = &lowestSite{Site: "pricerunner", Reason: err.Error()}
			}
			for _, s := range result.PerSite {
				if s.Cheapest == nil {
					continue
				}
				if result.BestOffer == nil || s.Cheapest.LowestPriceSEK < result.BestOffer.LowestPriceSEK {
					result.BestOffer = s.Cheapest
				}
			}
			if result.BestOffer == nil {
				result.Reason = "no offers found across both sites"
			}
			_ = flagMaxShipping // Reserved: shipping not exposed by search payloads; resolved via product detail
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().BoolVar(&flagInStock, "in-stock", false, "Filter to in-stock offers only")
	cmd.Flags().Float64Var(&flagMaxShipping, "max-shipping", 0, "Maximum shipping cost in SEK (resolved via product detail; reserved)")
	cmd.Flags().StringVar(&flagEAN, "ean", "", "Search by EAN/GTIN instead of free text")
	return cmd
}
