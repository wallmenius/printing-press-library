package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/blocket/internal/cliutil"
)

func newFiltersCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "filters",
		Short: "Inspect the live filter tree for a vertical so agents know valid filter values.",
	}
	cmd.AddCommand(newFiltersListCmd(flags))
	return cmd
}

// filterPaths maps each verticalFlag value to the SEARCH_ID API path.
// The same set used by every absorbed and mobility list command.
var filterPaths = map[string]string{
	"ads":          "/recommerce/forsale/search/api/search/SEARCH_ID_BAP_COMMON",
	"car":          "/mobility/search/api/search/SEARCH_ID_CAR_USED",
	"cars":         "/mobility/search/api/search/SEARCH_ID_CAR_USED",
	"boat":         "/mobility/search/api/search/SEARCH_ID_BOAT_USED",
	"boats":        "/mobility/search/api/search/SEARCH_ID_BOAT_USED",
	"mc":           "/mobility/search/api/search/SEARCH_ID_MC_USED",
	"motorcycles":  "/mobility/search/api/search/SEARCH_ID_MC_USED",
	"truck":        "/mobility/search/api/search/SEARCH_ID_CAR_TRUCK",
	"trucks":       "/mobility/search/api/search/SEARCH_ID_CAR_TRUCK",
	"bus":          "/mobility/search/api/search/SEARCH_ID_CAR_BUS",
	"buses":        "/mobility/search/api/search/SEARCH_ID_CAR_BUS",
	"construction": "/mobility/search/api/search/SEARCH_ID_CAR_AGRI",
	"caravans":     "/mobility/search/api/search/SEARCH_ID_CAR_CARAVAN",
	"mobilehomes":  "/mobility/search/api/search/SEARCH_ID_CAR_MOBILE_HOME",
	"atractors":    "/mobility/search/api/search/SEARCH_ID_CAR_A_TRACTOR",
	"atvs":         "/mobility/search/api/search/SEARCH_ID_MC_ATV",
	"scooters":     "/mobility/search/api/search/SEARCH_ID_MC_SCOOTER",
	"tractors":     "/mobility/search/api/search/SEARCH_ID_AGRICULTURE_TRACTOR",
	"tools":        "/mobility/search/api/search/SEARCH_ID_AGRICULTURE_TOOL",
	"combines":     "/mobility/search/api/search/SEARCH_ID_AGRICULTURE_THRESHING",
}

func newFiltersListCmd(flags *rootFlags) *cobra.Command {
	var vertical string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Probe a search response and print the filter tree for a vertical.",
		Long: `Issue one search request for the given vertical and print the
filters[] tree the API returns. Use this when you need to know which
make, model, fuel, body_type, etc. values are accepted by other
commands.

The API returns the filter tree on every search response — this
command is the standalone one-shot version for agents introspecting
the surface.`,
		Example:     "  blocket-pp-cli filters list --vertical car --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cliutil.IsVerifyEnv() && dryRunOK(flags) {
				return nil
			}
			if dryRunOK(flags) {
				return nil
			}
			path, ok := filterPaths[strings.ToLower(strings.TrimSpace(vertical))]
			if !ok {
				return fmt.Errorf("--vertical must be one of: ads, car, boat, mc, truck, bus, construction, caravans, mobilehomes, atractors, atvs, scooters, tractors, tools, combines")
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, err := c.Get(path, map[string]string{"page": "1"})
			if err != nil {
				return fmt.Errorf("probe failed: %w", err)
			}
			var resp struct {
				Filters  json.RawMessage `json:"filters"`
				Metadata struct {
					SearchKey string `json:"search_key"`
					Vertical  string `json:"vertical"`
				} `json:"metadata"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return err
			}

			out := map[string]any{
				"vertical":     vertical,
				"search_key":   resp.Metadata.SearchKey,
				"api_vertical": resp.Metadata.Vertical,
				"filters":      resp.Filters,
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().StringVar(&vertical, "vertical", "car", "Vertical to probe.")
	_ = cmd.RegisterFlagCompletionFunc("vertical", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		out := make([]string, 0, len(filterPaths))
		for k := range filterPaths {
			out = append(out, k)
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}
