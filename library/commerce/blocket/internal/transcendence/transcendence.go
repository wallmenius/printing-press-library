// Package transcendence holds shared helpers for the novel-feature
// commands (since, arbitrage, price-history, dealer, stale, watch, etc.).
// Generator-emitted code lives in internal/store and internal/client; this
// package wraps them with the snapshot, watch, and aggregation surfaces
// that only the local SQLite layer can answer.
package transcendence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/commerce/blocket/internal/store"
)

// Schema additions on top of the generator's `ads` and `cars` tables.
const novelSchema = `
CREATE TABLE IF NOT EXISTS ad_price_snapshots (
	ad_id     TEXT NOT NULL,
	vertical  TEXT NOT NULL,
	taken_at  INTEGER NOT NULL,
	amount    INTEGER NOT NULL,
	currency  TEXT NOT NULL DEFAULT 'SEK',
	PRIMARY KEY (ad_id, taken_at)
);
CREATE INDEX IF NOT EXISTS idx_ad_price_snapshots_ad ON ad_price_snapshots(ad_id);
CREATE INDEX IF NOT EXISTS idx_ad_price_snapshots_taken ON ad_price_snapshots(taken_at);

CREATE TABLE IF NOT EXISTS watches (
	name        TEXT PRIMARY KEY,
	vertical    TEXT NOT NULL,
	params_json TEXT NOT NULL,
	created_at  INTEGER NOT NULL,
	updated_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS watch_runs (
	watch_name TEXT NOT NULL,
	ran_at     INTEGER NOT NULL,
	ad_ids     TEXT NOT NULL,
	ad_prices  TEXT NOT NULL,
	PRIMARY KEY (watch_name, ran_at)
);
CREATE INDEX IF NOT EXISTS idx_watch_runs_name ON watch_runs(watch_name, ran_at DESC);

CREATE TABLE IF NOT EXISTS ad_descriptions (
	ad_id      TEXT PRIMARY KEY,
	vertical   TEXT NOT NULL,
	description TEXT NOT NULL,
	fetched_at INTEGER NOT NULL
);
`

func EnsureSchema(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, novelSchema); err != nil {
		return fmt.Errorf("ensure transcendence schema: %w", err)
	}
	return nil
}

func OpenStore(ctx context.Context, dbPath string) (*store.Store, error) {
	s, err := store.OpenWithContext(ctx, dbPath)
	if err != nil {
		return nil, err
	}
	if err := EnsureSchema(ctx, s.DB()); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

type AdRow struct {
	AdID         string  `json:"ad_id"`
	Vertical     string  `json:"vertical"`
	Heading      string  `json:"heading"`
	Make         string  `json:"make,omitempty"`
	Model        string  `json:"model,omitempty"`
	Year         int     `json:"year,omitempty"`
	Mileage      int     `json:"mileage,omitempty"`
	PriceAmount  int     `json:"price_amount"`
	PriceCurr    string  `json:"price_currency,omitempty"`
	Location     string  `json:"location,omitempty"`
	Lat          float64 `json:"lat,omitempty"`
	Lon          float64 `json:"lon,omitempty"`
	Timestamp    int64   `json:"timestamp,omitempty"`
	OrgID        int     `json:"org_id,omitempty"`
	OrgName      string  `json:"organisation_name,omitempty"`
	CanonicalURL string  `json:"canonical_url,omitempty"`
}

func VerticalTable(vertical string) string {
	switch strings.ToLower(strings.TrimSpace(vertical)) {
	case "car", "cars":
		return "cars"
	case "ad", "ads", "bap":
		return "ads"
	case "":
		return ""
	default:
		return "resources"
	}
}

func ScanAdRows(rows *sql.Rows, vertical string) ([]AdRow, error) {
	defer rows.Close()
	var out []AdRow
	for rows.Next() {
		var id string
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			return nil, err
		}
		row, err := UnmarshalAdRow(id, data, vertical)
		if err != nil {
			continue
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func UnmarshalAdRow(id string, data []byte, vertical string) (AdRow, error) {
	var raw struct {
		AdID    json.Number `json:"ad_id"`
		Heading string      `json:"heading"`
		Make    string      `json:"make"`
		Model   string      `json:"model"`
		Year    int         `json:"year"`
		Mileage int         `json:"mileage"`
		Price   struct {
			Amount       int    `json:"amount"`
			CurrencyCode string `json:"currency_code"`
		} `json:"price"`
		Location    string `json:"location"`
		Coordinates struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"coordinates"`
		Timestamp        int64  `json:"timestamp"`
		OrgID            int    `json:"org_id"`
		OrganisationName string `json:"organisation_name"`
		CanonicalURL     string `json:"canonical_url"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return AdRow{}, err
	}
	out := AdRow{
		AdID:         id,
		Vertical:     vertical,
		Heading:      raw.Heading,
		Make:         raw.Make,
		Model:        raw.Model,
		Year:         raw.Year,
		Mileage:      raw.Mileage,
		PriceAmount:  raw.Price.Amount,
		PriceCurr:    raw.Price.CurrencyCode,
		Location:     raw.Location,
		Lat:          raw.Coordinates.Lat,
		Lon:          raw.Coordinates.Lon,
		Timestamp:    raw.Timestamp,
		OrgID:        raw.OrgID,
		OrgName:      raw.OrganisationName,
		CanonicalURL: raw.CanonicalURL,
	}
	if out.AdID == "" {
		if s := strings.TrimSpace(string(raw.AdID)); s != "" {
			out.AdID = s
		}
	}
	return out, nil
}

func LoadVertical(ctx context.Context, s *store.Store, vertical string) ([]AdRow, error) {
	table := VerticalTable(vertical)
	if table == "" {
		return nil, fmt.Errorf("transcendence: unknown vertical %q", vertical)
	}
	switch table {
	case "ads", "cars":
		rows, err := s.DB().QueryContext(ctx, "SELECT id, data FROM "+table)
		if err != nil {
			return nil, err
		}
		return ScanAdRows(rows, vertical)
	default:
		rows, err := s.DB().QueryContext(ctx,
			`SELECT id, data FROM resources WHERE resource_type = ?`,
			vertical,
		)
		if err != nil {
			return nil, err
		}
		return ScanAdRows(rows, vertical)
	}
}

func SnapshotPrice(ctx context.Context, db *sql.DB, ad AdRow) error {
	if ad.PriceAmount <= 0 || ad.AdID == "" {
		return nil
	}
	curr := ad.PriceCurr
	if curr == "" {
		curr = "SEK"
	}
	_, err := db.ExecContext(ctx,
		`INSERT OR IGNORE INTO ad_price_snapshots (ad_id, vertical, taken_at, amount, currency)
		 VALUES (?, ?, ?, ?, ?)`,
		ad.AdID, ad.Vertical, time.Now().Unix(), ad.PriceAmount, curr,
	)
	return err
}

func MedianInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int(nil), values...)
	for i := 1; i < len(sorted); i++ {
		v := sorted[i]
		j := i
		for j > 0 && sorted[j-1] > v {
			sorted[j] = sorted[j-1]
			j--
		}
		sorted[j] = v
	}
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func PercentileInt(values []int) (p10, p50, p90 int) {
	if len(values) == 0 {
		return 0, 0, 0
	}
	sorted := append([]int(nil), values...)
	for i := 1; i < len(sorted); i++ {
		v := sorted[i]
		j := i
		for j > 0 && sorted[j-1] > v {
			sorted[j] = sorted[j-1]
			j--
		}
		sorted[j] = v
	}
	pick := func(p float64) int {
		idx := int(p * float64(len(sorted)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sorted) {
			idx = len(sorted) - 1
		}
		return sorted[idx]
	}
	return pick(0.1), pick(0.5), pick(0.9)
}

func MileageBand(mileage int) string {
	switch {
	case mileage <= 0:
		return "unknown"
	case mileage < 5000:
		return "0-5k"
	case mileage < 10000:
		return "5k-10k"
	case mileage < 15000:
		return "10k-15k"
	case mileage < 20000:
		return "15k-20k"
	case mileage < 30000:
		return "20k-30k"
	default:
		return "30k+"
	}
}

func YearBand(year int) string {
	if year <= 0 {
		return "unknown"
	}
	bucket := (year / 3) * 3
	return fmt.Sprintf("%d-%d", bucket, bucket+2)
}
