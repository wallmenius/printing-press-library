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

func newWatchCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Stored named-searches with run, list, and price-drop notification.",
	}
	cmd.AddCommand(newWatchAddCmd(flags))
	cmd.AddCommand(newWatchListCmd(flags))
	cmd.AddCommand(newWatchRunCmd(flags))
	cmd.AddCommand(newWatchRemoveCmd(flags))
	return cmd
}

func newWatchAddCmd(flags *rootFlags) *cobra.Command {
	var vertical string
	var query string
	var priceFrom, priceTo int

	cmd := &cobra.Command{
		Use:     "add [name]",
		Short:   "Save a named search for later runs.",
		Example: "  blocket-pp-cli watch add xc70 --vertical car --query \"Volvo XC70\" --price-to 80000",
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
			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("a name is required")
			}
			if strings.TrimSpace(vertical) == "" {
				return fmt.Errorf("--vertical is required")
			}
			params := map[string]string{"query": query}
			if priceFrom > 0 {
				params["price_from"] = fmt.Sprintf("%d", priceFrom)
			}
			if priceTo > 0 {
				params["price_to"] = fmt.Sprintf("%d", priceTo)
			}
			paramsJSON, _ := json.Marshal(params)

			ctx := context.Background()
			s, err := transcendence.OpenStore(ctx, defaultDBPath("blocket-pp-cli"))
			if err != nil {
				return err
			}
			defer s.Close()

			now := time.Now().Unix()
			_, err = s.DB().ExecContext(ctx,
				`INSERT INTO watches (name, vertical, params_json, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?)
				 ON CONFLICT(name) DO UPDATE SET vertical=excluded.vertical, params_json=excluded.params_json, updated_at=excluded.updated_at`,
				name, vertical, string(paramsJSON), now, now,
			)
			if err != nil {
				return err
			}
			out := map[string]any{
				"saved":    name,
				"vertical": vertical,
				"params":   params,
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().StringVar(&vertical, "vertical", "", "Vertical to scan (car, ads, boats, motorcycles, trucks…).")
	cmd.Flags().StringVar(&query, "query", "", "Free-text query to match against heading/make/model.")
	cmd.Flags().IntVar(&priceFrom, "price-from", 0, "Minimum price in SEK.")
	cmd.Flags().IntVar(&priceTo, "price-to", 0, "Maximum price in SEK.")
	return cmd
}

func newWatchListCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "list",
		Short:       "List every stored watch.",
		Example:     "  blocket-pp-cli watch list --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cliutil.IsVerifyEnv() && dryRunOK(flags) {
				return nil
			}
			if dryRunOK(flags) {
				return nil
			}
			ctx := context.Background()
			s, err := transcendence.OpenStore(ctx, defaultDBPath("blocket-pp-cli"))
			if err != nil {
				return err
			}
			defer s.Close()

			rows, err := s.DB().QueryContext(ctx,
				`SELECT name, vertical, params_json, created_at, updated_at FROM watches ORDER BY name`)
			if err != nil {
				return err
			}
			defer rows.Close()
			type w struct {
				Name      string            `json:"name"`
				Vertical  string            `json:"vertical"`
				Params    map[string]string `json:"params"`
				CreatedAt string            `json:"created_at"`
				UpdatedAt string            `json:"updated_at"`
			}
			var watches []w
			for rows.Next() {
				var name, vertical, paramsJSON string
				var createdAt, updatedAt int64
				if err := rows.Scan(&name, &vertical, &paramsJSON, &createdAt, &updatedAt); err != nil {
					return err
				}
				var params map[string]string
				_ = json.Unmarshal([]byte(paramsJSON), &params)
				watches = append(watches, w{
					Name:      name,
					Vertical:  vertical,
					Params:    params,
					CreatedAt: time.Unix(createdAt, 0).UTC().Format(time.RFC3339),
					UpdatedAt: time.Unix(updatedAt, 0).UTC().Format(time.RFC3339),
				})
			}
			out := map[string]any{"count": len(watches), "watches": watches}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	return cmd
}

func newWatchRemoveCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove [name]",
		Aliases: []string{"rm"},
		Short:   "Delete a stored watch (and its run history).",
		Example: "  blocket-pp-cli watch remove xc70",
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
			name := args[0]
			ctx := context.Background()
			s, err := transcendence.OpenStore(ctx, defaultDBPath("blocket-pp-cli"))
			if err != nil {
				return err
			}
			defer s.Close()

			res, err := s.DB().ExecContext(ctx, `DELETE FROM watches WHERE name = ?`, name)
			if err != nil {
				return err
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				return notFoundErr(fmt.Errorf("watch %q not found", name))
			}
			_, _ = s.DB().ExecContext(ctx, `DELETE FROM watch_runs WHERE watch_name = ?`, name)
			out := map[string]any{
				"removed":    name,
				"watch_rows": affected,
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	return cmd
}

func newWatchRunCmd(flags *rootFlags) *cobra.Command {
	var notifyOn string

	cmd := &cobra.Command{
		Use:   "run [name]",
		Short: "Run a stored watch and report new ads or price drops.",
		Long: `Replay a stored watch against the local store. With
--notify-on=price-drop, exit non-zero with structured JSON when ads in
the result set got cheaper since the previous run.

The previous run is whatever 'watch run' wrote last. The first run
records a baseline and reports nothing.`,
		Example: "  blocket-pp-cli watch run xc70 --notify-on price-drop --json",
		Annotations: map[string]string{
			"mcp:read-only":       "true",
			"pp:typed-exit-codes": "0,2",
		},
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
			name := args[0]

			ctx := context.Background()
			s, err := transcendence.OpenStore(ctx, defaultDBPath("blocket-pp-cli"))
			if err != nil {
				return err
			}
			defer s.Close()

			row := s.DB().QueryRowContext(ctx,
				`SELECT vertical, params_json FROM watches WHERE name = ?`, name)
			var vertical, paramsJSON string
			if err := row.Scan(&vertical, &paramsJSON); err != nil {
				return fmt.Errorf("watch %q not found — create it with 'watch add'", name)
			}
			var params map[string]string
			_ = json.Unmarshal([]byte(paramsJSON), &params)

			rows, err := transcendence.LoadVertical(ctx, s, vertical)
			if err != nil {
				return err
			}
			var matched []transcendence.AdRow
			for _, r := range rows {
				if rowMatchesParams(r, params) {
					matched = append(matched, r)
				}
			}

			currentPrices := map[string]int{}
			currentIDs := make([]string, 0, len(matched))
			for _, r := range matched {
				currentIDs = append(currentIDs, r.AdID)
				if r.PriceAmount > 0 {
					currentPrices[r.AdID] = r.PriceAmount
				}
			}

			// Load the previous run's state.
			prev := s.DB().QueryRowContext(ctx,
				`SELECT ad_ids, ad_prices FROM watch_runs WHERE watch_name = ? ORDER BY ran_at DESC LIMIT 1`, name)
			var prevIDsJSON, prevPricesJSON string
			havePrev := prev.Scan(&prevIDsJSON, &prevPricesJSON) == nil

			var prevIDs []string
			var prevPrices map[string]int
			if havePrev {
				_ = json.Unmarshal([]byte(prevIDsJSON), &prevIDs)
				_ = json.Unmarshal([]byte(prevPricesJSON), &prevPrices)
			}

			// Compute new ads.
			prevSet := make(map[string]struct{}, len(prevIDs))
			for _, id := range prevIDs {
				prevSet[id] = struct{}{}
			}
			var newAds []transcendence.AdRow
			for _, r := range matched {
				if _, ok := prevSet[r.AdID]; !ok {
					newAds = append(newAds, r)
				}
			}

			// Compute price drops.
			type drop struct {
				AdID         string  `json:"ad_id"`
				Heading      string  `json:"heading"`
				PrevAmount   int     `json:"prev_amount"`
				CurrAmount   int     `json:"current_amount"`
				DeltaAbs     int     `json:"delta_abs"`
				DeltaPct     float64 `json:"delta_pct"`
				CanonicalURL string  `json:"canonical_url,omitempty"`
			}
			var drops []drop
			for _, r := range matched {
				prev, ok := prevPrices[r.AdID]
				if !ok || r.PriceAmount <= 0 || prev <= 0 {
					continue
				}
				if r.PriceAmount < prev {
					drops = append(drops, drop{
						AdID:         r.AdID,
						Heading:      r.Heading,
						PrevAmount:   prev,
						CurrAmount:   r.PriceAmount,
						DeltaAbs:     prev - r.PriceAmount,
						DeltaPct:     float64(prev-r.PriceAmount) / float64(prev),
						CanonicalURL: r.CanonicalURL,
					})
				}
				_ = transcendence.SnapshotPrice(ctx, s.DB(), r)
			}

			// Persist this run.
			currentIDsJSON, _ := json.Marshal(currentIDs)
			currentPricesJSON, _ := json.Marshal(currentPrices)
			_, _ = s.DB().ExecContext(ctx,
				`INSERT INTO watch_runs (watch_name, ran_at, ad_ids, ad_prices) VALUES (?, ?, ?, ?)`,
				name, time.Now().Unix(), string(currentIDsJSON), string(currentPricesJSON),
			)

			out := map[string]any{
				"watch":        name,
				"vertical":     vertical,
				"matched":      len(matched),
				"new_count":    len(newAds),
				"price_drops":  drops,
				"baseline_run": !havePrev,
			}
			if !havePrev {
				out["hint"] = "First run — recording baseline; no diffs to report."
			}

			if err := printJSONFiltered(cmd.OutOrStdout(), out, flags); err != nil {
				return err
			}

			notify := strings.ToLower(strings.TrimSpace(notifyOn))
			switch notify {
			case "":
				return nil
			case "new":
				if havePrev && len(newAds) > 0 {
					return &cliError{code: 2, err: fmt.Errorf("%d new ad(s) since last run", len(newAds))}
				}
			case "price-drop":
				if havePrev && len(drops) > 0 {
					return &cliError{code: 2, err: fmt.Errorf("%d price drop(s) since last run", len(drops))}
				}
			default:
				return fmt.Errorf("--notify-on must be 'new' or 'price-drop'")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&notifyOn, "notify-on", "", "Exit non-zero on: 'new' (any new ad) or 'price-drop' (any price drop).")
	return cmd
}
