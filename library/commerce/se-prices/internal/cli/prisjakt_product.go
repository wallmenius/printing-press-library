// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: parses Prisjakt's React Query state into a typed product object.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/source/prisjakt"
)

func newPrisjaktProductCmd(flags *rootFlags) *cobra.Command {
	var flagID int
	cmd := &cobra.Command{
		Use:   "product",
		Short: "Get Prisjakt product detail by ID. Returns offers, ratings, brand, category, badges.",
		Example: "  se-prices-pp-cli prisjakt product --id 14969878\n" +
			"  se-prices-pp-cli prisjakt product --id 14969878 --json --select offers.merchant,offers.price_sek,offers.url",
		Annotations: map[string]string{
			"pp:endpoint":   "prisjakt.product",
			"pp:method":     "GET",
			"pp:path":       "/produkt.php",
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagID == 0 && !flags.dryRun {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			path := "/produkt.php"
			params := map[string]string{"p": fmt.Sprintf("%d", flagID)}
			html, err := c.Get(path, params)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			p, perr := prisjakt.ParseProduct(html)
			if perr != nil {
				return fmt.Errorf("parsing Prisjakt product %d: %w", flagID, perr)
			}
			return printJSONFiltered(cmd.OutOrStdout(), p, flags)
		},
	}
	cmd.Flags().IntVar(&flagID, "id", 0, "Prisjakt product ID (e.g., 14969878)")
	cmd.Flags().IntVar(&flagID, "p", 0, "")
	_ = cmd.Flags().MarkHidden("p")
	return cmd
}
