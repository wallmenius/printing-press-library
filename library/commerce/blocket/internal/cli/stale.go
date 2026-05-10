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

func newStaleCmd(flags *rootFlags) *cobra.Command {
	var vertical string
	var olderThan string
	var orgID int

	cmd := &cobra.Command{
		Use:   "stale",
		Short: "Listings older than N days, still active in the latest sync.",
		Long: `List ads whose published timestamp is older than the threshold but
that were still present in the most recent sync.

This is the negotiation candidate set: a listing that has been active
for 60+ days is by definition not selling at its current price. Use
--older-than to set the threshold (default 30d).`,
		Example:     "  blocket-pp-cli stale --vertical car --older-than 60d --json",
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

			d, err := parseDurationFlexible(olderThan)
			if err != nil {
				return fmt.Errorf("--older-than: %w", err)
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

			cutoff := time.Now().Add(-d).UnixMilli()
			var hits []map[string]any
			for _, r := range rows {
				if orgID != 0 && r.OrgID != orgID {
					continue
				}
				if r.Timestamp == 0 || r.Timestamp >= cutoff {
					continue
				}
				ageDays := int(time.Now().Sub(time.UnixMilli(r.Timestamp)) / (24 * time.Hour))
				hits = append(hits, map[string]any{
					"ad_id":             r.AdID,
					"heading":           r.Heading,
					"price_amount":      r.PriceAmount,
					"price_currency":    r.PriceCurr,
					"location":          r.Location,
					"org_id":            r.OrgID,
					"organisation_name": r.OrgName,
					"canonical_url":     r.CanonicalURL,
					"age_days":          ageDays,
					"posted_at":         time.UnixMilli(r.Timestamp).UTC().Format(time.RFC3339),
				})
			}

			out := map[string]any{
				"vertical":   vertical,
				"older_than": olderThan,
				"count":      len(hits),
				"ads":        hits,
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().StringVar(&vertical, "vertical", "car", "Vertical to scan.")
	cmd.Flags().StringVar(&olderThan, "older-than", "30d", "Age threshold (e.g. 30d, 60d, 168h).")
	cmd.Flags().IntVar(&orgID, "org-id", 0, "Restrict to a specific dealer org_id.")
	return cmd
}
