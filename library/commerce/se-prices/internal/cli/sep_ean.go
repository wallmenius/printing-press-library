// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored novel command: resolve EAN to both sites' product IDs.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type eanResult struct {
	EAN      string `json:"ean"`
	Products any    `json:"products"`
	Reason   string `json:"reason,omitempty"`
}

func newEANCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ean [ean]",
		Short: "Resolve an EAN/GTIN to both sites' product IDs from the local store.",
		Long: "Looks up a barcode in the local store and returns matching products from each site\n" +
			"(both Prisjakt and PriceRunner) plus the union of their current offers. Requires a\n" +
			"prior `sync` so EANs are populated.",
		Example: "  se-prices-pp-cli ean 0194253433927\n" +
			"  se-prices-pp-cli ean 0194253433927 --json --select products.source,products.lowest_sek,products.url",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ean := ""
			if len(args) > 0 {
				ean = args[0]
			}
			if ean == "" && !flags.dryRun {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if !looksLikeEAN(ean) {
				return fmt.Errorf("invalid EAN/GTIN %q: expected 8-14 digits", ean)
			}
			st, err := openSEPStore(cmd.Context())
			if err != nil {
				return err
			}
			defer st.Close()
			products, err := st.ProductsByEAN(cmd.Context(), ean)
			if err != nil {
				return err
			}
			res := &eanResult{EAN: ean, Products: products}
			if len(products) == 0 {
				res.Reason = "no products with this EAN in the local store; run `se-prices-pp-cli sync --source both` after browsing categories that include this product"
			}
			return printJSONFiltered(cmd.OutOrStdout(), res, flags)
		},
	}
	return cmd
}
