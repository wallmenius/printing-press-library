package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/blocket/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/commerce/blocket/internal/transcendence"
)

func newAppraiseCmd(flags *rootFlags) *cobra.Command {
	var vertical string
	var make_, model string
	var year, mileage int
	var yearTolerance, mileageTolerance int

	cmd := &cobra.Command{
		Use:   "appraise",
		Short: "Compute p10/p50/p90 of asking prices for a comparable vehicle from the synced corpus.",
		Long: `Price a hypothetical or specific vehicle against the local corpus.

Selects comparable listings (same make, same model, year within
+/- year-tolerance, mileage within +/- mileage-tolerance) and reports
p10/p50/p90 of their asking prices. Use this before bidding to know
where the listing sits in the distribution.`,
		Example:     "  blocket-pp-cli appraise --vertical car --make Volvo --model XC70 --year 2014 --mileage 18000 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cliutil.IsVerifyEnv() && dryRunOK(flags) {
				return nil
			}
			if dryRunOK(flags) {
				return nil
			}
			if strings.TrimSpace(vertical) == "" {
				return fmt.Errorf("--vertical is required")
			}
			if make_ == "" {
				return fmt.Errorf("--make is required")
			}

			ctx := context.Background()
			s, err := transcendence.OpenStore(ctx, defaultDBPath("blocket-pp-cli"))
			if err != nil {
				return err
			}
			defer s.Close()

			rows, err := transcendence.LoadVertical(ctx, s, vertical)
			if err != nil {
				return err
			}

			makeLower := strings.ToLower(make_)
			modelLower := strings.ToLower(model)
			var prices []int
			var matched []transcendence.AdRow
			for _, r := range rows {
				if r.PriceAmount <= 0 {
					continue
				}
				if strings.ToLower(r.Make) != makeLower {
					continue
				}
				if model != "" && strings.ToLower(r.Model) != modelLower {
					continue
				}
				if year > 0 && (r.Year < year-yearTolerance || r.Year > year+yearTolerance) {
					continue
				}
				if mileage > 0 && (r.Mileage < mileage-mileageTolerance || r.Mileage > mileage+mileageTolerance) {
					continue
				}
				prices = append(prices, r.PriceAmount)
				matched = append(matched, r)
			}

			p10, p50, p90 := transcendence.PercentileInt(prices)
			out := map[string]any{
				"vertical":         vertical,
				"make":             make_,
				"model":            model,
				"year":             year,
				"mileage":          mileage,
				"comparable_count": len(prices),
				"price_p10":        p10,
				"price_p50":        p50,
				"price_p90":        p90,
				"currency":         "SEK",
			}
			if len(prices) < 4 {
				out["hint"] = fmt.Sprintf(
					"Only %d comparable listings in the corpus — widen tolerances or run sync to broaden the comparable set.",
					len(prices),
				)
			}
			// Optionally include a small sample of the comparable rows.
			if len(matched) > 0 {
				sample := matched
				if len(sample) > 5 {
					sample = sample[:5]
				}
				out["sample"] = sample
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().StringVar(&vertical, "vertical", "car", "Vertical to scan.")
	cmd.Flags().StringVar(&make_, "make", "", "Vehicle make (required).")
	cmd.Flags().StringVar(&model, "model", "", "Vehicle model.")
	cmd.Flags().IntVar(&year, "year", 0, "Model year.")
	cmd.Flags().IntVar(&mileage, "mileage", 0, "Mileage in mil (Swedish miles, 1 mil = 10 km).")
	cmd.Flags().IntVar(&yearTolerance, "year-tolerance", 1, "Year tolerance for the comparable set.")
	cmd.Flags().IntVar(&mileageTolerance, "mileage-tolerance", 3000, "Mileage tolerance in mil.")
	return cmd
}
