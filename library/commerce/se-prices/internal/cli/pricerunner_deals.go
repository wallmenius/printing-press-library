// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: lists curated deals from PriceRunner /deals.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/source/pricerunner"
)

func newPricerunnerDealsCmd(flags *rootFlags) *cobra.Command {
	var flagLimit int
	cmd := &cobra.Command{
		Use:     "deals",
		Short:   "List PriceRunner curated deals.",
		Example: "  se-prices-pp-cli pricerunner deals --limit 20 --json",
		Annotations: map[string]string{
			"pp:endpoint":   "pricerunner.deals",
			"pp:method":     "GET",
			"pp:path":       "/deals",
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			html, err := c.Get("https://www.pricerunner.se/deals", nil)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			deals, perr := pricerunner.ParseDeals(html)
			if perr != nil {
				return fmt.Errorf("parsing PriceRunner deals: %w", perr)
			}
			if flagLimit > 0 && len(deals) > flagLimit {
				deals = deals[:flagLimit]
			}
			return printJSONFiltered(cmd.OutOrStdout(), deals, flags)
		},
	}
	cmd.Flags().IntVar(&flagLimit, "limit", 0, "Maximum deals to return")
	return cmd
}
