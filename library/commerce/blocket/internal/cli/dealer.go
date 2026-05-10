package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/blocket/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/commerce/blocket/internal/transcendence"
)

func newDealerCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dealer",
		Short: "Dealer portfolio commands — list ads, aggregate stats by org_id.",
	}
	cmd.AddCommand(newDealerAdsCmd(flags))
	return cmd
}

func newDealerAdsCmd(flags *rootFlags) *cobra.Command {
	var stats bool
	var vertical string

	cmd := &cobra.Command{
		Use:   "ads [org_id]",
		Short: "List or aggregate ads for a dealer org_id.",
		Long: `List every ad in the local store for a given dealer org_id, optionally
aggregated.

With --stats, returns a one-row aggregate summary: inventory count,
median price, oldest-listing-days, and the percentage of listings
whose snapshotted price dropped this week.`,
		Example:     "  blocket-pp-cli dealer ads 1234567 --stats --json",
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
			orgIDArg := strings.TrimSpace(args[0])
			if orgIDArg == "" {
				return fmt.Errorf("org_id is required")
			}
			// Reject obviously invalid org_id values (must be numeric).
			for _, r := range orgIDArg {
				if r < '0' || r > '9' {
					return usageErr(fmt.Errorf("org_id must be numeric (got %q)", orgIDArg))
				}
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

			var hits []transcendence.AdRow
			for _, r := range rows {
				if fmt.Sprintf("%d", r.OrgID) == orgIDArg {
					hits = append(hits, r)
				}
			}

			if !stats {
				out := map[string]any{
					"org_id":   orgIDArg,
					"vertical": vertical,
					"count":    len(hits),
					"ads":      hits,
				}
				return printJSONFiltered(cmd.OutOrStdout(), out, flags)
			}

			// Compute aggregate stats.
			now := time.Now()
			oneWeekAgo := now.Add(-7 * 24 * time.Hour).Unix()
			var prices []int
			var oldest int64
			var droppedThisWeek int

			adIDs := make([]string, 0, len(hits))
			for _, r := range hits {
				if r.PriceAmount > 0 {
					prices = append(prices, r.PriceAmount)
				}
				if r.Timestamp > 0 {
					ts := r.Timestamp / 1000
					if oldest == 0 || ts < oldest {
						oldest = ts
					}
				}
				adIDs = append(adIDs, r.AdID)
			}

			// Count snapshot drops in the last 7 days for these ads.
			if len(adIDs) > 0 {
				placeholders := strings.Repeat("?,", len(adIDs))
				placeholders = strings.TrimSuffix(placeholders, ",")
				query := `
					WITH snapped AS (
						SELECT ad_id, taken_at, amount,
						       LAG(amount) OVER (PARTITION BY ad_id ORDER BY taken_at) AS prev_amount
						FROM ad_price_snapshots
						WHERE ad_id IN (` + placeholders + `) AND taken_at >= ?
					)
					SELECT COUNT(DISTINCT ad_id) FROM snapped WHERE prev_amount IS NOT NULL AND amount < prev_amount`

				queryArgs := make([]any, 0, len(adIDs)+1)
				for _, id := range adIDs {
					queryArgs = append(queryArgs, id)
				}
				queryArgs = append(queryArgs, oneWeekAgo)

				if row := s.DB().QueryRowContext(ctx, query, queryArgs...); row != nil {
					_ = row.Scan(&droppedThisWeek)
				}
			}

			summary := map[string]any{
				"org_id":           orgIDArg,
				"vertical":         vertical,
				"inventory_count":  len(hits),
				"median_price_sek": transcendence.MedianInt(prices),
				"oldest_listing_days": func() int {
					if oldest == 0 {
						return 0
					}
					return int(now.Sub(time.Unix(oldest, 0)) / (24 * time.Hour))
				}(),
				"price_drops_last_7d": droppedThisWeek,
				"price_drop_rate_pct": func() float64 {
					if len(hits) == 0 {
						return 0
					}
					return float64(droppedThisWeek) * 100 / float64(len(hits))
				}(),
			}
			if hits != nil {
				if name := firstNonEmpty(hitsOrgNames(hits)); name != "" {
					summary["organisation_name"] = name
				}
			}
			return printJSONFiltered(cmd.OutOrStdout(), summary, flags)
		},
	}
	cmd.Flags().BoolVar(&stats, "stats", false, "Return aggregate stats instead of the full ad list.")
	cmd.Flags().StringVar(&vertical, "vertical", "car", "Vertical to scan.")
	return cmd
}

func hitsOrgNames(hits []transcendence.AdRow) []string {
	out := make([]string, 0, len(hits))
	for _, r := range hits {
		out = append(out, r.OrgName)
	}
	return out
}

func firstNonEmpty(values []string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
