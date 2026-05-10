// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored novel command: catalogue diff per category.

package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/store"
)

type diffResult struct {
	Category       string            `json:"category,omitempty"`
	OnlySource     string            `json:"only_source,omitempty"`
	UniqueProducts []store.SEProduct `json:"unique_products"`
	Reason         string            `json:"reason,omitempty"`
}

func newCatalogueDiffCmd(flags *rootFlags) *cobra.Command {
	var (
		flagCategory string
		flagOnly     string
		flagLimit    int
	)
	cmd := &cobra.Command{
		Use:   "catalogue-diff",
		Short: "Products that appear in one site's category index but are missing from the other.",
		Long: "Set-difference SQL: returns products tagged with the requested category in one\n" +
			"source's index but absent from the other site's index after a fresh `sync` of both.\n" +
			"Matching uses EAN when present and falls back to normalized title.",
		Example: "  se-prices-pp-cli catalogue-diff --category mobiltelefoner --only prisjakt\n" +
			"  se-prices-pp-cli catalogue-diff --category mobiltelefoner --only pricerunner --json",
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
			products, err := st.AllProductsByCategoryAcrossSources(cmd.Context(), flagCategory)
			if err != nil {
				return err
			}
			only := strings.ToLower(flagOnly)
			result := &diffResult{Category: flagCategory, OnlySource: only}
			result.UniqueProducts = computeCatalogueDiff(products, only)
			if flagLimit > 0 && len(result.UniqueProducts) > flagLimit {
				result.UniqueProducts = result.UniqueProducts[:flagLimit]
			}
			if len(result.UniqueProducts) == 0 {
				result.Reason = "no products unique to the requested side; sync more categories or check that both sites have populated rows"
			}
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringVar(&flagCategory, "category", "", "Category slug to scope (e.g., mobiltelefoner)")
	cmd.Flags().StringVar(&flagOnly, "only", "prisjakt", "Which side to report unique products for: prisjakt or pricerunner")
	cmd.Flags().IntVar(&flagLimit, "limit", 0, "Maximum products to return")
	return cmd
}

func computeCatalogueDiff(products []store.SEProduct, only string) []store.SEProduct {
	if only != "prisjakt" && only != "pricerunner" {
		only = "prisjakt"
	}
	other := "prisjakt"
	if only == "prisjakt" {
		other = "pricerunner"
	}
	otherIndex := map[string]struct{}{}
	for _, p := range products {
		if p.Source != other {
			continue
		}
		if p.EAN != "" {
			otherIndex["ean:"+p.EAN] = struct{}{}
		}
		if nk := normalizeName(p.Name); nk != "" {
			otherIndex["name:"+nk] = struct{}{}
		}
	}
	var out []store.SEProduct
	for _, p := range products {
		if p.Source != only {
			continue
		}
		var keys []string
		if p.EAN != "" {
			keys = append(keys, "ean:"+p.EAN)
		}
		if nk := normalizeName(p.Name); nk != "" {
			keys = append(keys, "name:"+nk)
		}
		matched := false
		for _, k := range keys {
			if _, ok := otherIndex[k]; ok {
				matched = true
				break
			}
		}
		if !matched {
			out = append(out, p)
		}
	}
	return out
}
