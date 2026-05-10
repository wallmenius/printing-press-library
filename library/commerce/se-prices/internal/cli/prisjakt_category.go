// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: returns trending products from a Prisjakt category page.
//
// Note: Prisjakt's full paginated category list is loaded via subsequent
// client-side React Query fetches we cannot reach (the GraphQL host is
// internal-only). We return trendingProducts as the publicly-visible
// surface; documented in README's Known Gaps.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/source/prisjakt"
)

func newPrisjaktCategoryCmd(flags *rootFlags) *cobra.Command {
	var (
		flagSlug  string
		flagBrand int
		flagLimit int
	)
	cmd := &cobra.Command{
		Use:   "category [slug]",
		Short: "Browse a Prisjakt category page. Returns trending products + brand/category metadata.",
		Example: "  se-prices-pp-cli prisjakt category mobiltelefoner\n" +
			"  se-prices-pp-cli prisjakt category mobiltelefoner --brand 142\n" +
			"  se-prices-pp-cli prisjakt category mobiltelefoner --json --select products.name,products.lowest_price_sek",
		Annotations: map[string]string{
			"pp:endpoint":   "prisjakt.category",
			"pp:method":     "GET",
			"pp:path":       "/c/{slug}",
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				flagSlug = args[0]
			}
			if flagSlug == "" && !flags.dryRun {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			path := "/c/" + flagSlug
			params := map[string]string{}
			if flagBrand != 0 {
				params["brand"] = fmt.Sprintf("%d", flagBrand)
			}
			html, err := c.Get(path, params)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			cat, perr := prisjakt.ParseCategory(html)
			if perr != nil {
				return fmt.Errorf("parsing Prisjakt category %s: %w", flagSlug, perr)
			}
			if flagLimit > 0 && len(cat.Products) > flagLimit {
				cat.Products = cat.Products[:flagLimit]
				cat.Total = len(cat.Products)
			}
			return printJSONFiltered(cmd.OutOrStdout(), cat, flags)
		},
	}
	cmd.Flags().StringVar(&flagSlug, "slug", "", "Category slug (e.g., mobiltelefoner)")
	cmd.Flags().IntVar(&flagBrand, "brand", 0, "Filter by brand ID")
	cmd.Flags().IntVar(&flagLimit, "limit", 0, "Maximum products to return")
	return cmd
}
