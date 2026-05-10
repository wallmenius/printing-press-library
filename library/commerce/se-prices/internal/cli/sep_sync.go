// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: cross-site population command that feeds the novel commands.
//
// The generated `sync` walks spec-declared endpoints and writes raw rows into
// the per-resource tables. The cross-site analytical commands (arbitrage,
// catalogue-diff, drops, etc.) need a normalized view in the sep_* tables, so
// this command does an explicit category sweep on each source, parses the SSR
// JSON state, upserts typed rows, and records a price snapshot per product.

package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/client"
	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/source/pricerunner"
	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/source/prisjakt"
	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/store"
)

type sepSyncReport struct {
	Source         string `json:"source"`
	CategoryInput  string `json:"category"`
	ProductsSaved  int    `json:"products_saved"`
	SnapshotsTaken int    `json:"snapshots_taken"`
	Reason         string `json:"reason,omitempty"`
}

type sepSyncResult struct {
	StartedAt  string          `json:"started_at"`
	FinishedAt string          `json:"finished_at"`
	Reports    []sepSyncReport `json:"reports"`
}

func newSepSyncCmd(flags *rootFlags) *cobra.Command {
	var (
		flagSource     string
		flagCategories []string
		flagPRCats     []string
	)
	cmd := &cobra.Command{
		Use:   "sep-sync",
		Short: "Populate the local cross-site store from category sweeps on Prisjakt and PriceRunner.",
		Long: "Fetches category pages from each source, parses their SSR state, upserts typed\n" +
			"product rows into sep_products, and records a price snapshot per product so the\n" +
			"analytical commands (arbitrage, drops, history-combo, watchlist check, is-sale)\n" +
			"have data to work with. Run this before invoking those commands; rerun on a cron\n" +
			"to build snapshot history.",
		Example: "  se-prices-pp-cli sep-sync\n" +
			"  se-prices-pp-cli sep-sync --source prisjakt --categories mobiltelefoner,datorer\n" +
			"  se-prices-pp-cli sep-sync --source pricerunner --pr-categories 1/Mobiltelefoner,29/Digitalkameror",
		Annotations: map[string]string{},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			st, err := openSEPStore(cmd.Context())
			if err != nil {
				return err
			}
			defer st.Close()
			started := time.Now().UTC()
			result := &sepSyncResult{StartedAt: started.Format(time.RFC3339)}
			source := strings.ToLower(flagSource)
			if source == "" {
				source = "both"
			}
			if source == "both" || source == "prisjakt" {
				cats := flagCategories
				if len(cats) == 0 {
					cats = []string{"mobiltelefoner", "datorer", "horlurar"}
				}
				for _, slug := range cats {
					rep := syncPrisjaktCategory(cmd.Context(), c, st, slug)
					result.Reports = append(result.Reports, rep)
				}
			}
			if source == "both" || source == "pricerunner" {
				cats := flagPRCats
				if len(cats) == 0 {
					cats = []string{"1/Mobiltelefoner", "29/Digitalkameror"}
				}
				for _, idAndSlug := range cats {
					rep := syncPriceRunnerCategory(cmd.Context(), c, st, idAndSlug)
					result.Reports = append(result.Reports, rep)
				}
			}
			result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringVar(&flagSource, "source", "both", "Source to sync: prisjakt, pricerunner, or both")
	cmd.Flags().StringSliceVar(&flagCategories, "categories", nil, "Prisjakt category slugs (default: mobiltelefoner,datorer,horlurar)")
	cmd.Flags().StringSliceVar(&flagPRCats, "pr-categories", nil, "PriceRunner '<id>/<slug>' pairs (default: 1/Mobiltelefoner,29/Digitalkameror)")
	return cmd
}

func syncPrisjaktCategory(ctx context.Context, c *client.Client, st *store.Store, slug string) sepSyncReport {
	rep := sepSyncReport{Source: "prisjakt", CategoryInput: slug}
	html, err := c.Get("/c/"+slug, nil)
	if err != nil {
		rep.Reason = fmt.Sprintf("fetch error: %v", err)
		return rep
	}
	cat, err := prisjakt.ParseCategory(html)
	if err != nil {
		rep.Reason = fmt.Sprintf("parse error: %v", err)
		return rep
	}
	products := make([]store.SEProduct, 0, len(cat.Products))
	for _, p := range cat.Products {
		products = append(products, store.SEProduct{
			Source:    "prisjakt",
			SourceID:  fmt.Sprintf("%d", p.ID),
			Name:      p.Name,
			Brand:     p.Brand,
			Category:  cat.Name,
			URL:       p.URL,
			ImageURL:  p.ImageURL,
			LowestSEK: p.LowestPriceSEK,
		})
	}
	if err := st.UpsertSEProductBatch(ctx, products); err != nil {
		rep.Reason = fmt.Sprintf("upsert error: %v", err)
		return rep
	}
	rep.ProductsSaved = len(products)
	for _, p := range products {
		if p.LowestSEK <= 0 {
			continue
		}
		_ = st.AppendSESnapshot(ctx, store.SESnapshot{Source: p.Source, SourceID: p.SourceID, LowestSEK: p.LowestSEK})
		rep.SnapshotsTaken++
	}
	return rep
}

func syncPriceRunnerCategory(ctx context.Context, c *client.Client, st *store.Store, idSlug string) sepSyncReport {
	rep := sepSyncReport{Source: "pricerunner", CategoryInput: idSlug}
	parts := strings.SplitN(idSlug, "/", 2)
	if len(parts) != 2 {
		rep.Reason = "expected <id>/<slug> (e.g., 1/Mobiltelefoner)"
		return rep
	}
	html, err := c.Get("https://www.pricerunner.se/cl/"+parts[0]+"/"+parts[1], nil)
	if err != nil {
		rep.Reason = fmt.Sprintf("fetch error: %v", err)
		return rep
	}
	cat, err := pricerunner.ParseCategory(html)
	if err != nil {
		rep.Reason = fmt.Sprintf("parse error: %v", err)
		return rep
	}
	products := make([]store.SEProduct, 0, len(cat.Products))
	for _, p := range cat.Products {
		products = append(products, store.SEProduct{
			Source:    "pricerunner",
			SourceID:  p.ID,
			Name:      p.Name,
			Brand:     p.Brand,
			Category:  parts[1],
			URL:       p.URL,
			ImageURL:  p.ImageURL,
			LowestSEK: p.LowestPriceSEK,
		})
	}
	if err := st.UpsertSEProductBatch(ctx, products); err != nil {
		rep.Reason = fmt.Sprintf("upsert error: %v", err)
		return rep
	}
	rep.ProductsSaved = len(products)
	for _, p := range products {
		if p.LowestSEK <= 0 {
			continue
		}
		_ = st.AppendSESnapshot(ctx, store.SESnapshot{Source: p.Source, SourceID: p.SourceID, LowestSEK: p.LowestSEK})
		rep.SnapshotsTaken++
	}
	return rep
}
