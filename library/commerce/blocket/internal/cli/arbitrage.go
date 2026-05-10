package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/blocket/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/commerce/blocket/internal/transcendence"
)

func newArbitrageCmd(flags *rootFlags) *cobra.Command {
	var vertical string
	var make_ string
	var model string
	var threshold float64
	var minSamples int

	cmd := &cobra.Command{
		Use:   "arbitrage",
		Short: "Find listings priced below threshold × median for their cohort.",
		Long: `Locate underpriced listings in the local store.

For every (make, model, year-band, mileage-band) cohort, computes the
median price across the synced corpus and lists current ads whose
price is at or below threshold × median. Cohorts smaller than
--min-samples (default 4) are skipped — comparing to a one-listing
median makes no sense.

The vehicle-specific filters (--make, --model) are optional. When
omitted, every make/model is considered separately. When set, the
comparison set is restricted to that filter — e.g. "find underpriced
Volvo XC70s" vs "find underpriced cars overall".`,
		Example:     "  blocket-pp-cli arbitrage --vertical car --make Volvo --threshold 0.8 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cliutil.IsVerifyEnv() && dryRunOK(flags) {
				return nil
			}
			if dryRunOK(flags) {
				return nil
			}
			if strings.TrimSpace(vertical) == "" {
				return fmt.Errorf("--vertical is required (e.g. car, ads, boats)")
			}
			if threshold <= 0 || threshold > 1 {
				return fmt.Errorf("--threshold must be between 0 and 1 (got %v)", threshold)
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

			// Filter by user-supplied make/model.
			if make_ != "" {
				makeLower := strings.ToLower(make_)
				var f []transcendence.AdRow
				for _, r := range rows {
					if strings.ToLower(r.Make) == makeLower {
						f = append(f, r)
					}
				}
				rows = f
			}
			if model != "" {
				modelLower := strings.ToLower(model)
				var f []transcendence.AdRow
				for _, r := range rows {
					if strings.ToLower(r.Model) == modelLower {
						f = append(f, r)
					}
				}
				rows = f
			}

			// Bucket by cohort.
			type cohort struct {
				key    string
				prices []int
				rows   []transcendence.AdRow
			}
			buckets := map[string]*cohort{}
			for _, r := range rows {
				if r.PriceAmount <= 0 {
					continue
				}
				key := strings.Join([]string{
					strings.ToLower(strings.TrimSpace(r.Make)),
					strings.ToLower(strings.TrimSpace(r.Model)),
					transcendence.YearBand(r.Year),
					transcendence.MileageBand(r.Mileage),
				}, "|")
				b := buckets[key]
				if b == nil {
					b = &cohort{key: key}
					buckets[key] = b
				}
				b.prices = append(b.prices, r.PriceAmount)
				b.rows = append(b.rows, r)
			}

			type underpriced struct {
				Ad        transcendence.AdRow `json:"ad"`
				Median    int                 `json:"cohort_median"`
				Threshold int                 `json:"cohort_threshold_price"`
				Discount  float64             `json:"discount_vs_median"`
				Cohort    string              `json:"cohort_key"`
				CohortN   int                 `json:"cohort_size"`
			}
			var hits []underpriced
			for _, b := range buckets {
				if len(b.prices) < minSamples {
					continue
				}
				median := transcendence.MedianInt(b.prices)
				if median <= 0 {
					continue
				}
				cutoff := int(float64(median) * threshold)
				for _, r := range b.rows {
					if r.PriceAmount > 0 && r.PriceAmount <= cutoff {
						hits = append(hits, underpriced{
							Ad:        r,
							Median:    median,
							Threshold: cutoff,
							Discount:  1 - float64(r.PriceAmount)/float64(median),
							Cohort:    b.key,
							CohortN:   len(b.prices),
						})
					}
				}
			}

			out := map[string]any{
				"vertical":    vertical,
				"threshold":   threshold,
				"corpus_size": len(rows),
				"cohorts":     len(buckets),
				"hits":        hits,
				"hit_count":   len(hits),
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}

	cmd.Flags().StringVar(&vertical, "vertical", "car", "Vertical to scan (car, ads, boats, motorcycles).")
	cmd.Flags().StringVar(&make_, "make", "", "Restrict to a single vehicle make.")
	cmd.Flags().StringVar(&model, "model", "", "Restrict to a single vehicle model.")
	cmd.Flags().Float64Var(&threshold, "threshold", 0.8, "Price as fraction of cohort median (0.8 = 20% under).")
	cmd.Flags().IntVar(&minSamples, "min-samples", 4, "Skip cohorts with fewer than this many comparable listings.")
	return cmd
}
