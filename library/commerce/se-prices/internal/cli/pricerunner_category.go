// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: parses PriceRunner category page payload.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/source/pricerunner"
)

func newPricerunnerCategoryCmd(flags *rootFlags) *cobra.Command {
	var (
		flagID    int
		flagSlug  string
		flagLimit int
	)
	cmd := &cobra.Command{
		Use:   "category [id] [slug]",
		Short: "Browse a PriceRunner category. Returns the listed products.",
		Example: "  se-prices-pp-cli pricerunner category 1 Mobiltelefoner\n" +
			"  se-prices-pp-cli pricerunner category 1 Mobiltelefoner --limit 10 --json --select products.name,products.lowest_price_sek",
		Annotations: map[string]string{
			"pp:endpoint":   "pricerunner.category",
			"pp:method":     "GET",
			"pp:path":       "/cl/{id}/{slug}",
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if _, err := fmt.Sscanf(args[0], "%d", &flagID); err != nil {
					return fmt.Errorf("invalid category id %q: expected a positive integer", args[0])
				}
			}
			if len(args) > 1 {
				flagSlug = args[1]
			}
			if (flagID == 0 || flagSlug == "") && !flags.dryRun {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			path := fmt.Sprintf("https://www.pricerunner.se/cl/%d/%s", flagID, flagSlug)
			html, err := c.Get(path, nil)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			cat, perr := pricerunner.ParseCategory(html)
			if perr != nil {
				return fmt.Errorf("parsing PriceRunner category: %w", perr)
			}
			cat.ID = flagID
			cat.Slug = flagSlug
			if flagLimit > 0 && len(cat.Products) > flagLimit {
				cat.Products = cat.Products[:flagLimit]
			}
			return printJSONFiltered(cmd.OutOrStdout(), cat, flags)
		},
	}
	cmd.Flags().IntVar(&flagID, "id", 0, "Category numeric ID (e.g., 1 for mobiltelefoner)")
	cmd.Flags().StringVar(&flagSlug, "slug", "", "Category slug (e.g., Mobiltelefoner)")
	cmd.Flags().IntVar(&flagLimit, "limit", 0, "Maximum products to return")
	return cmd
}
