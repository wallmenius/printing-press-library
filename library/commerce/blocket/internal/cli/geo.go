package cli

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/blocket/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/commerce/blocket/internal/transcendence"
)

func newGeoCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "geo",
		Short: "Geo radius queries over synced ads.",
	}
	cmd.AddCommand(newGeoNearCmd(flags))
	return cmd
}

func newGeoNearCmd(flags *rootFlags) *cobra.Command {
	var lat, lon float64
	var radiusKm float64
	var vertical string

	cmd := &cobra.Command{
		Use:   "near",
		Short: "Filter synced ads by point + radius using haversine distance.",
		Long: `Filter ads in the local store by an arbitrary lat/lon and radius
in kilometres.

The Blocket public search API filters by Swedish region IDs (län), not
arbitrary lat/lon. The per-ad coordinates only become useful in the
local store. Sync ads first, then narrow with this command.`,
		Example:     "  blocket-pp-cli geo near --lat 59.33 --lon 18.06 --radius 30 --vertical car --json",
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
			if lat == 0 && lon == 0 {
				return fmt.Errorf("--lat and --lon are required")
			}
			if radiusKm <= 0 {
				return fmt.Errorf("--radius must be > 0 km")
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

			type hit struct {
				transcendence.AdRow
				DistanceKm float64 `json:"distance_km"`
			}
			var hits []hit
			for _, r := range rows {
				if r.Lat == 0 && r.Lon == 0 {
					continue
				}
				d := haversineKm(lat, lon, r.Lat, r.Lon)
				if d <= radiusKm {
					hits = append(hits, hit{AdRow: r, DistanceKm: d})
				}
			}

			out := map[string]any{
				"center":    map[string]float64{"lat": lat, "lon": lon},
				"radius_km": radiusKm,
				"vertical":  vertical,
				"hit_count": len(hits),
				"hits":      hits,
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().Float64Var(&lat, "lat", 0, "Center latitude.")
	cmd.Flags().Float64Var(&lon, "lon", 0, "Center longitude.")
	cmd.Flags().Float64Var(&radiusKm, "radius", 30, "Radius in kilometres.")
	cmd.Flags().StringVar(&vertical, "vertical", "car", "Vertical to scan.")
	return cmd
}

// haversineKm computes great-circle distance between two lat/lon points
// in kilometres. Earth radius 6371 km.
func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371.0
	rad := func(deg float64) float64 { return deg * math.Pi / 180 }
	dlat := rad(lat2 - lat1)
	dlon := rad(lon2 - lon1)
	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(rad(lat1))*math.Cos(rad(lat2))*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return r * c
}
