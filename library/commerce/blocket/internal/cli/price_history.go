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

func newPriceHistoryCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "price-history [ad_id]",
		Short: "Show every snapshotted price for an ad with deltas.",
		Long: `Show the price history for an ad over time.

Snapshots are populated by 'blocket-pp-cli sync' (which writes one
row per ad per call) plus 'blocket-pp-cli watch run' (which records
the prices observed each run). The Blocket API exposes only the
current price; this command answers "did the price drop, when, and
by how much".

If no rows exist yet for the ad, the command prints an empty array.
Run sync first or use the ad's heading directly via 'ads get'.`,
		Example:     "  blocket-pp-cli price-history 22587669 --json",
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
			adID := strings.TrimSpace(args[0])
			// Reject obviously invalid ad_id values (must be numeric).
			for _, r := range adID {
				if r < '0' || r > '9' {
					return usageErr(fmt.Errorf("ad_id must be numeric (got %q)", adID))
				}
			}

			ctx := context.Background()
			s, err := transcendence.OpenStore(ctx, defaultDBPath("blocket-pp-cli"))
			if err != nil {
				return err
			}
			defer s.Close()

			rows, err := s.DB().QueryContext(ctx,
				`SELECT taken_at, amount, currency, vertical
				 FROM ad_price_snapshots
				 WHERE ad_id = ?
				 ORDER BY taken_at ASC`,
				adID,
			)
			if err != nil {
				return err
			}
			defer rows.Close()

			type snap struct {
				At       string  `json:"at"`
				Amount   int     `json:"amount"`
				Currency string  `json:"currency"`
				Vertical string  `json:"vertical,omitempty"`
				DeltaAbs int     `json:"delta_abs,omitempty"`
				DeltaPct float64 `json:"delta_pct,omitempty"`
			}
			var snaps []snap
			var prevAmount int
			for rows.Next() {
				var ts int64
				var amt int
				var curr, vert string
				if err := rows.Scan(&ts, &amt, &curr, &vert); err != nil {
					return err
				}
				s := snap{
					At:       time.Unix(ts, 0).UTC().Format(time.RFC3339),
					Amount:   amt,
					Currency: curr,
					Vertical: vert,
				}
				if prevAmount > 0 {
					s.DeltaAbs = amt - prevAmount
					s.DeltaPct = float64(s.DeltaAbs) / float64(prevAmount)
				}
				snaps = append(snaps, s)
				prevAmount = amt
			}
			if err := rows.Err(); err != nil {
				return err
			}

			out := map[string]any{
				"ad_id":     adID,
				"snapshots": snaps,
				"count":     len(snaps),
			}
			if len(snaps) >= 2 {
				first := snaps[0]
				last := snaps[len(snaps)-1]
				out["overall_delta_abs"] = last.Amount - first.Amount
				if first.Amount > 0 {
					out["overall_delta_pct"] = float64(last.Amount-first.Amount) / float64(first.Amount)
				}
			}
			if len(snaps) == 0 {
				out["hint"] = fmt.Sprintf("No snapshots yet for ad %s — run 'sync' or 'watch run' first.", adID)
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	return cmd
}
