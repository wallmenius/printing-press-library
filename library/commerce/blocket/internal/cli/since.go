package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/blocket/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/commerce/blocket/internal/transcendence"
)

func newSinceCmd(flags *rootFlags) *cobra.Command {
	var search string
	var since string
	var verticalFlag string

	cmd := &cobra.Command{
		Use:   "since",
		Short: "List ads added since a timestamp for a stored named-search.",
		Long: `List ads added since a given timestamp for a stored named-search.

Use this in cron jobs or agent scheduled tasks: exit code 2 means new
ads were found, exit code 0 means nothing new. The timestamp accepts
RFC3339, RFC3339Nano, or a relative form like '24h' / '7d'.

The named-search is a stored watch — create one first with:

  blocket-pp-cli watch add <name> --vertical car --query "Volvo XC70"

then re-run since with --search <name>.`,
		Example: "  blocket-pp-cli since --search xc70 --since 24h --json",
		Annotations: map[string]string{
			"mcp:read-only":       "true",
			"pp:typed-exit-codes": "0,2",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cliutil.IsVerifyEnv() {
				if dryRunOK(flags) {
					return nil
				}
			}
			if dryRunOK(flags) {
				return nil
			}
			if strings.TrimSpace(search) == "" && strings.TrimSpace(verticalFlag) == "" {
				return fmt.Errorf("--search <name> or --vertical <vertical> is required")
			}

			cutoff, err := parseSinceTimestamp(since)
			if err != nil {
				return fmt.Errorf("--since: %w", err)
			}

			ctx := context.Background()
			s, err := transcendence.OpenStore(ctx, defaultDBPath("blocket-pp-cli"))
			if err != nil {
				return err
			}
			defer s.Close()

			var watchVertical string
			var watchParams map[string]string
			if strings.TrimSpace(search) != "" {
				row := s.DB().QueryRowContext(ctx,
					`SELECT vertical, params_json FROM watches WHERE name = ?`, search)
				var paramsJSON string
				if err := row.Scan(&watchVertical, &paramsJSON); err != nil {
					// No matching watch is not an error — print empty result with a hint
					// so cron + agent flows treat "watch not yet seeded" the same as "no
					// new ads since cutoff".
					out := map[string]any{
						"watch":    search,
						"vertical": "",
						"since":    cutoff.UTC().Format(time.RFC3339),
						"count":    0,
						"ads":      []transcendence.AdRow{},
						"hint":     fmt.Sprintf("No watch named %q yet — create one with 'watch add %s --vertical car --query \"…\"'.", search, search),
					}
					return printJSONFiltered(cmd.OutOrStdout(), out, flags)
				}
				_ = json.Unmarshal([]byte(paramsJSON), &watchParams)
			} else {
				watchVertical = verticalFlag
			}

			rows, err := transcendence.LoadVertical(ctx, s, watchVertical)
			if err != nil {
				return err
			}

			var fresh []transcendence.AdRow
			for _, r := range rows {
				if r.Timestamp == 0 {
					continue
				}
				ts := time.UnixMilli(r.Timestamp)
				if ts.After(cutoff) && rowMatchesParams(r, watchParams) {
					fresh = append(fresh, r)
				}
			}

			out := map[string]any{
				"watch":    search,
				"vertical": watchVertical,
				"since":    cutoff.UTC().Format(time.RFC3339),
				"count":    len(fresh),
				"ads":      fresh,
			}
			if err := printJSONFiltered(cmd.OutOrStdout(), out, flags); err != nil {
				return err
			}
			if len(fresh) > 0 {
				return &cliError{code: 2, err: fmt.Errorf("%d new ad(s) since %s", len(fresh), cutoff.UTC().Format(time.RFC3339))}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&search, "search", "", "Stored watch name (see 'watch list').")
	cmd.Flags().StringVar(&since, "since", "24h", "Cutoff timestamp (RFC3339 or relative like 24h, 7d, 1h).")
	cmd.Flags().StringVar(&verticalFlag, "vertical", "", "Vertical to scan when no --search is given (ads, cars, boats, motorcycles, trucks, etc.).")
	return cmd
}

// parseSinceTimestamp accepts RFC3339, RFC3339Nano, or a duration suffix
// (e.g. 24h, 7d, 1h). Relative forms are subtracted from time.Now().
func parseSinceTimestamp(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t, nil
	}
	// Relative: 24h, 7d, 1h, 30m
	if d, err := parseDurationFlexible(s); err == nil {
		return time.Now().Add(-d), nil
	}
	return time.Time{}, fmt.Errorf("could not parse %q (use RFC3339, YYYY-MM-DD, or relative like 24h, 7d)", s)
}

// parseDurationFlexible extends time.ParseDuration with a 'd' (days) unit.
func parseDurationFlexible(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "d") {
		var n int
		if _, err := fmt.Sscanf(strings.TrimSuffix(s, "d"), "%d", &n); err == nil && n >= 0 {
			return time.Duration(n) * 24 * time.Hour, nil
		}
		return 0, fmt.Errorf("bad day duration %q", s)
	}
	return time.ParseDuration(s)
}

// rowMatchesParams checks if a stored ad row matches the watch's
// params (loose substring match against heading + make + model). The
// match is tolerant: missing or empty params count as wildcards.
func rowMatchesParams(r transcendence.AdRow, params map[string]string) bool {
	if len(params) == 0 {
		return true
	}
	q := strings.ToLower(strings.TrimSpace(params["query"]))
	if q != "" {
		hay := strings.ToLower(r.Heading + " " + r.Make + " " + r.Model)
		if !strings.Contains(hay, q) {
			return false
		}
	}
	if priceTo := params["price_to"]; priceTo != "" {
		var max int
		_, _ = fmt.Sscanf(priceTo, "%d", &max)
		if max > 0 && r.PriceAmount > max {
			return false
		}
	}
	if priceFrom := params["price_from"]; priceFrom != "" {
		var min int
		_, _ = fmt.Sscanf(priceFrom, "%d", &min)
		if min > 0 && r.PriceAmount < min {
			return false
		}
	}
	return true
}
