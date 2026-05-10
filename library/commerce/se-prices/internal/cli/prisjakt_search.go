// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: parses Prisjakt's React Query state into typed search results.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/source/prisjakt"
)

func newPrisjaktSearchCmd(flags *rootFlags) *cobra.Command {
	var (
		flagQuery string
		flagLimit int
	)
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search Prisjakt products by name. Returns the matching product list.",
		Example: "  se-prices-pp-cli prisjakt search --query iphone\n" +
			"  se-prices-pp-cli prisjakt search --query iphone --json --select products.name,products.lowest_price_sek",
		Annotations: map[string]string{
			"pp:endpoint":   "prisjakt.search",
			"pp:method":     "GET",
			"pp:path":       "/search",
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
			path := "/search"
			params := map[string]string{"search": flagQuery}
			html, err := c.Get(path, params)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			result, perr := prisjakt.ParseSearch(html, flagQuery)
			if perr != nil {
				return fmt.Errorf("parsing Prisjakt search: %w", perr)
			}
			if flagLimit > 0 && len(result.Products) > flagLimit {
				result.Products = result.Products[:flagLimit]
			}
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringVarP(&flagQuery, "query", "q", "", "Search text (Swedish or English)")
	cmd.Flags().IntVar(&flagLimit, "limit", 0, "Maximum number of results to return")
	cmd.Flags().StringVar(&flagQuery, "search", "", "")
	_ = cmd.Flags().MarkHidden("search")
	return cmd
}
