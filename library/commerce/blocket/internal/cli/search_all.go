package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/blocket/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/commerce/blocket/internal/transcendence"
)

func newSearchAllCmd(flags *rootFlags) *cobra.Command {
	var maxPrice int
	var minPrice int
	var limit int

	cmd := &cobra.Command{
		Use:   "search-all [query]",
		Short: "Cross-vertical free-text search across the local store.",
		Long: `Search across every synced vertical at once: BAP, cars, boats,
motorcycles, trucks, buses, agriculture machinery, caravans, mobile
homes, A-tractors, ATVs, and scooters.

Match is case-insensitive substring against heading, make, model,
and organisation name. Every wrapper is per-vertical; this command
is the local-only union.`,
		Example:     "  blocket-pp-cli search-all \"Volvo\" --max-price 50000 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cliutil.IsVerifyEnv() && dryRunOK(flags) {
				return nil
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) == 0 {
				return cmd.Help()
			}
			query := strings.ToLower(strings.TrimSpace(strings.Join(args, " ")))
			if query == "" {
				return fmt.Errorf("a query argument is required")
			}

			ctx := context.Background()
			s, err := transcendence.OpenStore(ctx, defaultDBPath("blocket-pp-cli"))
			if err != nil {
				return err
			}
			defer s.Close()

			verticals := []string{
				"ads", "cars",
				"boats", "motorcycles",
				"trucks", "buses", "construction",
				"tractors", "tools", "combines",
				"caravans", "mobilehomes", "atractors",
				"atvs", "scooters",
			}

			var hits []transcendence.AdRow
			for _, v := range verticals {
				rows, err := transcendence.LoadVertical(ctx, s, v)
				if err != nil {
					continue
				}
				for _, r := range rows {
					if maxPrice > 0 && r.PriceAmount > maxPrice {
						continue
					}
					if minPrice > 0 && r.PriceAmount < minPrice {
						continue
					}
					hay := strings.ToLower(r.Heading + " " + r.Make + " " + r.Model + " " + r.OrgName)
					if strings.Contains(hay, query) {
						r.Vertical = v
						hits = append(hits, r)
					}
				}
			}
			if limit > 0 && len(hits) > limit {
				hits = hits[:limit]
			}

			out := map[string]any{
				"query": query,
				"count": len(hits),
				"hits":  hits,
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().IntVar(&maxPrice, "max-price", 0, "Maximum price in SEK.")
	cmd.Flags().IntVar(&minPrice, "min-price", 0, "Minimum price in SEK.")
	cmd.Flags().IntVar(&limit, "limit", 0, "Cap the number of returned hits (0 = no cap).")
	return cmd
}
