// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: parses PriceRunner's product-detail-initial / -offers / -price-level
// queries from the __DEHYDRATED_QUERY_STATE__ payload.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/source/pricerunner"
)

func newPricerunnerProductCmd(flags *rootFlags) *cobra.Command {
	var flagIDPath string
	cmd := &cobra.Command{
		Use:   "product [id-path]",
		Short: "Get PriceRunner product detail by URL path. Returns offers, brand, category, price level.",
		Long: "id-path is the segment after /pl/ in the product URL — e.g.,\n" +
			"  '1-3208336567/Mobiltelefoner/Apple-iPhone-15-Pro-Max-256GB-Natural-Titanium-priser'.\n\n" +
			"Get an id-path by running `pricerunner search '<query>' --json --select products.url` first.",
		Example: "  se-prices-pp-cli pricerunner product 1-3208336567/Mobiltelefoner/Apple-iPhone-15-Pro-Max-256GB-Natural-Titanium-priser\n" +
			"  se-prices-pp-cli pricerunner product '<id-path>' --json --select offers.merchant,offers.price_sek",
		Annotations: map[string]string{
			"pp:endpoint":   "pricerunner.product",
			"pp:method":     "GET",
			"pp:path":       "/pl/{id_path}",
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				flagIDPath = args[0]
			}
			if flagIDPath == "" && !flags.dryRun {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			path := "https://www.pricerunner.se/pl/" + flagIDPath
			html, err := c.Get(path, nil)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			p, perr := pricerunner.ParseProduct(html)
			if perr != nil {
				return fmt.Errorf("parsing PriceRunner product: %w", perr)
			}
			return printJSONFiltered(cmd.OutOrStdout(), p, flags)
		},
	}
	cmd.Flags().StringVar(&flagIDPath, "id", "", "PriceRunner product path (group-id/category/slug-priser)")
	return cmd
}
