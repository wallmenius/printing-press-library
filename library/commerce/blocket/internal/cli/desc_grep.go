package cli

import (
	"context"
	"fmt"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/blocket/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/commerce/blocket/internal/transcendence"
)

func newDescGrepCmd(flags *rootFlags) *cobra.Command {
	var vertical string
	var pattern string
	var caseSensitive bool

	cmd := &cobra.Command{
		Use:   "desc-grep",
		Short: "Match a Go regular expression against ad descriptions in the local store.",
		Long: `Match a Go regular expression against the ad description text
backfilled by 'sync' and 'ad get' calls.

The Blocket search API matches the 'q' parameter against headings, not
descriptions. Hot deal-defining keywords ("nyservad", "rökfri",
"original") often appear only in the body. desc-grep runs your regex
locally against the synced description corpus.

If no descriptions exist yet, run 'sync' or 'ad get <id>' to backfill.`,
		Example:     "  blocket-pp-cli desc-grep --vertical car --pattern \"\\bnyservad\\b\" --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cliutil.IsVerifyEnv() && dryRunOK(flags) {
				return nil
			}
			if dryRunOK(flags) {
				return nil
			}
			if pattern == "" {
				return fmt.Errorf("--pattern is required")
			}
			pat := pattern
			if !caseSensitive {
				pat = "(?i)" + pat
			}
			re, err := regexp.Compile(pat)
			if err != nil {
				return fmt.Errorf("invalid regex: %w", err)
			}

			ctx := context.Background()
			s, err := transcendence.OpenStore(ctx, defaultDBPath("blocket-pp-cli"))
			if err != nil {
				return err
			}
			defer s.Close()

			query := `SELECT ad_id, vertical, description FROM ad_descriptions`
			var queryArgs []any
			if vertical != "" {
				query += ` WHERE vertical = ?`
				queryArgs = append(queryArgs, vertical)
			}

			rows, err := s.DB().QueryContext(ctx, query, queryArgs...)
			if err != nil {
				return err
			}
			defer rows.Close()

			type hit struct {
				AdID     string `json:"ad_id"`
				Vertical string `json:"vertical"`
				Snippet  string `json:"snippet"`
			}
			var hits []hit
			for rows.Next() {
				var adID, vert, desc string
				if err := rows.Scan(&adID, &vert, &desc); err != nil {
					return err
				}
				if loc := re.FindStringIndex(desc); loc != nil {
					start := loc[0] - 50
					if start < 0 {
						start = 0
					}
					end := loc[1] + 50
					if end > len(desc) {
						end = len(desc)
					}
					hits = append(hits, hit{
						AdID:     adID,
						Vertical: vert,
						Snippet:  desc[start:end],
					})
				}
			}

			out := map[string]any{
				"pattern":   pattern,
				"vertical":  vertical,
				"hit_count": len(hits),
				"hits":      hits,
			}
			if len(hits) == 0 {
				out["hint"] = "No matches. Run 'sync' or 'ads get <id>' first to backfill description text into the local store."
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}

	cmd.Flags().StringVar(&vertical, "vertical", "", "Restrict to one vertical (car, ads, …); empty = all.")
	cmd.Flags().StringVar(&pattern, "pattern", "", "Go regexp to match (e.g. \"\\bnyservad\\b\").")
	cmd.Flags().BoolVar(&caseSensitive, "case-sensitive", false, "Disable the default case-insensitive match.")
	return cmd
}
