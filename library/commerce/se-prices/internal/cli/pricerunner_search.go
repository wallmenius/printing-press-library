// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: parses PriceRunner's __DEHYDRATED_QUERY_STATE__ search payload.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/source/pricerunner"
)

func newPricerunnerSearchCmd(flags *rootFlags) *cobra.Command {
	var (
		flagQuery string
		flagLimit int
	)
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search PriceRunner products by name. Returns the matching product list.",
		Example: "  se-prices-pp-cli pricerunner search --query iphone\n" +
			"  se-prices-pp-cli pricerunner search --query iphone --json --select products.name,products.lowest_price_sek",
		Annotations: map[string]string{
			"pp:endpoint":   "pricerunner.search",
			"pp:method":     "GET",
			"pp:path":       "https://www.pricerunner.se/results",
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagQuery == "" && !flags.dryRun {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			path := "https://www.pricerunner.se/results"
			params := map[string]string{"q": flagQuery}
			html, err := c.Get(path, params)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			result, perr := pricerunner.ParseSearch(html, flagQuery)
			if perr != nil {
				return fmt.Errorf("parsing PriceRunner search: %w", perr)
			}
			if flagLimit > 0 && len(result.Products) > flagLimit {
				result.Products = result.Products[:flagLimit]
			}
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringVarP(&flagQuery, "query", "q", "", "Search text")
	cmd.Flags().IntVar(&flagLimit, "limit", 0, "Maximum number of results to return")
	return cmd
}
