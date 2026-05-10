// Hand-authored se-prices schema + helpers for cross-site product/offer/snapshot
// state and the watchlist. Lives alongside the generator-emitted store.go;
// EnsureSEPricesSchema is called lazily by commands that depend on these tables
// so a fresh DB is migrated on first novel-command use.

package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SEProduct is one cross-site product row.
type SEProduct struct {
	Source     string  `json:"source"`
	SourceID   string  `json:"source_id"`
	Name       string  `json:"name"`
	Brand      string  `json:"brand,omitempty"`
	Category   string  `json:"category,omitempty"`
	EAN        string  `json:"ean,omitempty"`
	URL        string  `json:"url,omitempty"`
	ImageURL   string  `json:"image_url,omitempty"`
	LastSeenAt string  `json:"last_seen_at,omitempty"`
	LowestSEK  float64 `json:"lowest_sek,omitempty"`
}

// SEOffer is one current offer row.
type SEOffer struct {
	Source     string  `json:"source"`
	SourceID   string  `json:"source_id"`
	MerchantID string  `json:"merchant_id,omitempty"`
	Merchant   string  `json:"merchant,omitempty"`
	PriceSEK   float64 `json:"price_sek"`
	Shipping   float64 `json:"shipping_sek,omitempty"`
	TotalSEK   float64 `json:"total_sek,omitempty"`
	Stock      string  `json:"stock,omitempty"`
	URL        string  `json:"url,omitempty"`
	FetchedAt  string  `json:"fetched_at,omitempty"`
}

// SESnapshot is one historical price observation per product per timestamp.
type SESnapshot struct {
	Source     string  `json:"source"`
	SourceID   string  `json:"source_id"`
	TakenAt    string  `json:"taken_at"`
	LowestSEK  float64 `json:"lowest_sek,omitempty"`
	OfferCount int     `json:"offer_count,omitempty"`
}

// SEWatchedItem is one tracked-product wishlist row.
type SEWatchedItem struct {
	ID          int64   `json:"id"`
	Source      string  `json:"source,omitempty"`
	SourceID    string  `json:"source_id,omitempty"`
	EAN         string  `json:"ean,omitempty"`
	Label       string  `json:"label,omitempty"`
	MaxPriceSEK float64 `json:"max_price_sek,omitempty"`
	AddedAt     string  `json:"added_at,omitempty"`
}

// EnsureSEPricesSchema creates the se-prices tables on first use. All
// statements are CREATE ... IF NOT EXISTS, so this is idempotent and safe
// to call repeatedly. The previous implementation cached the result in a
// package-level sync.Once, but that permanently poisoned every caller in
// the process if a transient error fired during early init (e.g., during
// a parallel test run before OpenWithContext finished migrating). Re-running
// the CREATE statements on every call costs microseconds against an open
// SQLite handle and lets transient failures self-heal on the next call.
func (s *Store) EnsureSEPricesSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS sep_products (
				source TEXT NOT NULL,
				source_id TEXT NOT NULL,
				name TEXT NOT NULL,
				brand TEXT,
				category TEXT,
				ean TEXT,
				url TEXT,
				image_url TEXT,
				lowest_price_sek REAL,
				last_seen_at DATETIME,
				PRIMARY KEY (source, source_id)
			)`,
		`CREATE INDEX IF NOT EXISTS idx_sep_products_ean ON sep_products(ean)`,
		`CREATE INDEX IF NOT EXISTS idx_sep_products_brand ON sep_products(brand)`,
		`CREATE INDEX IF NOT EXISTS idx_sep_products_category ON sep_products(category)`,
		`CREATE INDEX IF NOT EXISTS idx_sep_products_name ON sep_products(name)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS sep_products_fts USING fts5(
				source UNINDEXED, source_id UNINDEXED, name, brand, category, tokenize='porter unicode61'
			)`,
		`CREATE TABLE IF NOT EXISTS sep_offers (
				source TEXT NOT NULL,
				source_id TEXT NOT NULL,
				merchant_id TEXT,
				merchant TEXT,
				price_sek REAL,
				shipping_sek REAL,
				total_sek REAL,
				stock TEXT,
				url TEXT,
				fetched_at DATETIME,
				PRIMARY KEY (source, source_id, merchant_id)
			)`,
		`CREATE INDEX IF NOT EXISTS idx_sep_offers_product ON sep_offers(source, source_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sep_offers_price ON sep_offers(total_sek)`,
		`CREATE TABLE IF NOT EXISTS sep_price_snapshots (
				source TEXT NOT NULL,
				source_id TEXT NOT NULL,
				taken_at DATETIME NOT NULL,
				lowest_price_sek REAL,
				offer_count INTEGER,
				PRIMARY KEY (source, source_id, taken_at)
			)`,
		`CREATE INDEX IF NOT EXISTS idx_sep_snapshots_taken ON sep_price_snapshots(taken_at)`,
		`CREATE INDEX IF NOT EXISTS idx_sep_snapshots_product ON sep_price_snapshots(source, source_id)`,
		`CREATE TABLE IF NOT EXISTS sep_watchlist (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				source TEXT,
				source_id TEXT,
				ean TEXT,
				label TEXT,
				max_price_sek REAL,
				added_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	for _, q := range stmts {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("se_prices migration %q: %w", firstLine(q), err)
		}
	}
	return nil
}

func firstLine(s string) string {
	for i, c := range s {
		if c == '\n' {
			return s[:i]
		}
	}
	return s
}

// UpsertSEProduct inserts or updates a product row.
func (s *Store) UpsertSEProduct(ctx context.Context, p SEProduct) error {
	if err := s.EnsureSEPricesSchema(ctx); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if p.LastSeenAt == "" {
		p.LastSeenAt = now
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO sep_products (source, source_id, name, brand, category, ean, url, image_url, lowest_price_sek, last_seen_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(source, source_id) DO UPDATE SET
			name=excluded.name,
			brand=COALESCE(NULLIF(excluded.brand, ''), brand),
			category=COALESCE(NULLIF(excluded.category, ''), category),
			ean=COALESCE(NULLIF(excluded.ean, ''), ean),
			url=COALESCE(NULLIF(excluded.url, ''), url),
			image_url=COALESCE(NULLIF(excluded.image_url, ''), image_url),
			lowest_price_sek=excluded.lowest_price_sek,
			last_seen_at=excluded.last_seen_at`,
		p.Source, p.SourceID, p.Name, p.Brand, p.Category, p.EAN, p.URL, p.ImageURL, p.LowestSEK, p.LastSeenAt,
	); err != nil {
		return fmt.Errorf("upserting product (%s, %s): %w", p.Source, p.SourceID, err)
	}
	// DELETE before INSERT keeps the FTS index aligned 1:1 with sep_products.
	// Without this, every upsert appended a new FTS row, so after N syncs each
	// product appeared N times in MATCH results — search hit counts inflated
	// linearly with sync history. The FTS table is contentless-style here
	// (no triggers wired against sep_products), so we own the cleanup.
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sep_products_fts WHERE source=? AND source_id=?`,
		p.Source, p.SourceID,
	); err != nil {
		return fmt.Errorf("clearing fts row for (%s, %s): %w", p.Source, p.SourceID, err)
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO sep_products_fts (source, source_id, name, brand, category) VALUES (?,?,?,?,?)`,
		p.Source, p.SourceID, p.Name, p.Brand, p.Category,
	); err != nil {
		return fmt.Errorf("indexing fts row for (%s, %s): %w", p.Source, p.SourceID, err)
	}
	return nil
}

// UpsertSEProductBatch upserts many products in a single transaction.
func (s *Store) UpsertSEProductBatch(ctx context.Context, products []SEProduct) error {
	if len(products) == 0 {
		return nil
	}
	if err := s.EnsureSEPricesSchema(ctx); err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO sep_products (source, source_id, name, brand, category, ean, url, image_url, lowest_price_sek, last_seen_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(source, source_id) DO UPDATE SET
			name=excluded.name,
			brand=COALESCE(NULLIF(excluded.brand, ''), brand),
			category=COALESCE(NULLIF(excluded.category, ''), category),
			ean=COALESCE(NULLIF(excluded.ean, ''), ean),
			url=COALESCE(NULLIF(excluded.url, ''), url),
			image_url=COALESCE(NULLIF(excluded.image_url, ''), image_url),
			lowest_price_sek=excluded.lowest_price_sek,
			last_seen_at=excluded.last_seen_at`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	// FTS DELETE+INSERT per row, scoped by (source, source_id). Without this
	// the batch path accumulated duplicate FTS rows on every sync, mirroring
	// the bug fixed in UpsertSEProduct.
	ftsDel, err := tx.PrepareContext(ctx, `DELETE FROM sep_products_fts WHERE source=? AND source_id=?`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer ftsDel.Close()
	ftsIns, err := tx.PrepareContext(ctx, `INSERT INTO sep_products_fts (source, source_id, name, brand, category) VALUES (?,?,?,?,?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer ftsIns.Close()
	now := time.Now().UTC().Format(time.RFC3339)
	for _, p := range products {
		if p.LastSeenAt == "" {
			p.LastSeenAt = now
		}
		if _, err := stmt.ExecContext(ctx, p.Source, p.SourceID, p.Name, p.Brand, p.Category, p.EAN, p.URL, p.ImageURL, p.LowestSEK, p.LastSeenAt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("upserting product batch (%s, %s): %w", p.Source, p.SourceID, err)
		}
		if _, err := ftsDel.ExecContext(ctx, p.Source, p.SourceID); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("clearing fts row (%s, %s): %w", p.Source, p.SourceID, err)
		}
		if _, err := ftsIns.ExecContext(ctx, p.Source, p.SourceID, p.Name, p.Brand, p.Category); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("indexing fts row (%s, %s): %w", p.Source, p.SourceID, err)
		}
	}
	return tx.Commit()
}

// ReplaceSEOffers swaps the offer rows for one product to the supplied list.
// Used after re-fetching a product page.
func (s *Store) ReplaceSEOffers(ctx context.Context, source, sourceID string, offers []SEOffer) error {
	if err := s.EnsureSEPricesSchema(ctx); err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sep_offers WHERE source=? AND source_id=?`, source, sourceID); err != nil {
		_ = tx.Rollback()
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT OR REPLACE INTO sep_offers (source, source_id, merchant_id, merchant, price_sek, shipping_sek, total_sek, stock, url, fetched_at) VALUES (?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	now := time.Now().UTC().Format(time.RFC3339)
	for _, o := range offers {
		if o.FetchedAt == "" {
			o.FetchedAt = now
		}
		if o.MerchantID == "" {
			o.MerchantID = o.Merchant
		}
		if _, err := stmt.ExecContext(ctx, source, sourceID, o.MerchantID, o.Merchant, o.PriceSEK, o.Shipping, o.TotalSEK, o.Stock, o.URL, o.FetchedAt); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// AppendSESnapshot records a price observation in the historical series.
func (s *Store) AppendSESnapshot(ctx context.Context, snap SESnapshot) error {
	if err := s.EnsureSEPricesSchema(ctx); err != nil {
		return err
	}
	if snap.TakenAt == "" {
		snap.TakenAt = time.Now().UTC().Format(time.RFC3339)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO sep_price_snapshots (source, source_id, taken_at, lowest_price_sek, offer_count) VALUES (?,?,?,?,?)`,
		snap.Source, snap.SourceID, snap.TakenAt, snap.LowestSEK, snap.OfferCount,
	)
	return err
}

// SearchSEProducts performs an FTS5 lookup. Returns products ordered by name.
func (s *Store) SearchSEProducts(ctx context.Context, query string, limit int) ([]SEProduct, error) {
	if err := s.EnsureSEPricesSchema(ctx); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT source, source_id, name, brand, category, ean, url, image_url, COALESCE(lowest_price_sek, 0), COALESCE(last_seen_at, '')
		FROM sep_products
		WHERE rowid IN (SELECT rowid FROM sep_products_fts WHERE sep_products_fts MATCH ? LIMIT ?)
		ORDER BY name LIMIT ?`,
		query, limit*2, limit,
	)
	if err != nil {
		// FTS5 syntax errors fall back to LIKE
		return s.searchProductsLike(ctx, query, limit)
	}
	defer rows.Close()
	return scanSEProducts(rows)
}

func (s *Store) searchProductsLike(ctx context.Context, query string, limit int) ([]SEProduct, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT source, source_id, name, brand, category, ean, url, image_url, COALESCE(lowest_price_sek, 0), COALESCE(last_seen_at, '')
		FROM sep_products
		WHERE name LIKE ? OR brand LIKE ?
		ORDER BY name LIMIT ?`,
		"%"+query+"%", "%"+query+"%", limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSEProducts(rows)
}

func scanSEProducts(rows *sql.Rows) ([]SEProduct, error) {
	var out []SEProduct
	for rows.Next() {
		var p SEProduct
		if err := rows.Scan(&p.Source, &p.SourceID, &p.Name, &p.Brand, &p.Category, &p.EAN, &p.URL, &p.ImageURL, &p.LowestSEK, &p.LastSeenAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ProductsByEAN returns all rows matching the EAN (typically one per source).
func (s *Store) ProductsByEAN(ctx context.Context, ean string) ([]SEProduct, error) {
	if err := s.EnsureSEPricesSchema(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT source, source_id, name, brand, category, ean, url, image_url, COALESCE(lowest_price_sek, 0), COALESCE(last_seen_at, '')
		FROM sep_products WHERE ean = ?`, ean)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSEProducts(rows)
}

// OffersForProduct returns all current offers for a product.
func (s *Store) OffersForProduct(ctx context.Context, source, sourceID string) ([]SEOffer, error) {
	if err := s.EnsureSEPricesSchema(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT source, source_id, COALESCE(merchant_id, ''), COALESCE(merchant, ''), COALESCE(price_sek, 0), COALESCE(shipping_sek, 0), COALESCE(total_sek, 0), COALESCE(stock, ''), COALESCE(url, ''), COALESCE(fetched_at, '')
		FROM sep_offers WHERE source=? AND source_id=? ORDER BY total_sek ASC`, source, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SEOffer
	for rows.Next() {
		var o SEOffer
		if err := rows.Scan(&o.Source, &o.SourceID, &o.MerchantID, &o.Merchant, &o.PriceSEK, &o.Shipping, &o.TotalSEK, &o.Stock, &o.URL, &o.FetchedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// SnapshotsForProduct returns the historical snapshot series.
func (s *Store) SnapshotsForProduct(ctx context.Context, source, sourceID string, since time.Time) ([]SESnapshot, error) {
	if err := s.EnsureSEPricesSchema(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT source, source_id, taken_at, COALESCE(lowest_price_sek, 0), COALESCE(offer_count, 0)
		FROM sep_price_snapshots WHERE source=? AND source_id=? AND taken_at >= ? ORDER BY taken_at ASC`,
		source, sourceID, since.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SESnapshot
	for rows.Next() {
		var ss SESnapshot
		if err := rows.Scan(&ss.Source, &ss.SourceID, &ss.TakenAt, &ss.LowestSEK, &ss.OfferCount); err != nil {
			return nil, err
		}
		out = append(out, ss)
	}
	return out, rows.Err()
}

// AddWatched inserts a wishlist row.
func (s *Store) AddWatched(ctx context.Context, item SEWatchedItem) (int64, error) {
	if err := s.EnsureSEPricesSchema(ctx); err != nil {
		return 0, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	res, err := s.db.ExecContext(ctx, `INSERT INTO sep_watchlist (source, source_id, ean, label, max_price_sek) VALUES (?,?,?,?,?)`,
		item.Source, item.SourceID, item.EAN, item.Label, item.MaxPriceSEK)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListWatched returns all wishlist rows.
func (s *Store) ListWatched(ctx context.Context) ([]SEWatchedItem, error) {
	if err := s.EnsureSEPricesSchema(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(source, ''), COALESCE(source_id, ''), COALESCE(ean, ''), COALESCE(label, ''), COALESCE(max_price_sek, 0), COALESCE(added_at, '')
		FROM sep_watchlist ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SEWatchedItem
	for rows.Next() {
		var w SEWatchedItem
		if err := rows.Scan(&w.ID, &w.Source, &w.SourceID, &w.EAN, &w.Label, &w.MaxPriceSEK, &w.AddedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// RemoveWatched deletes a wishlist row by id.
func (s *Store) RemoveWatched(ctx context.Context, id int64) (int64, error) {
	if err := s.EnsureSEPricesSchema(ctx); err != nil {
		return 0, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	res, err := s.db.ExecContext(ctx, `DELETE FROM sep_watchlist WHERE id=?`, id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// CategoryProducts returns all products in a category.
func (s *Store) CategoryProducts(ctx context.Context, source, category string) ([]SEProduct, error) {
	if err := s.EnsureSEPricesSchema(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT source, source_id, name, brand, category, ean, url, image_url, COALESCE(lowest_price_sek, 0), COALESCE(last_seen_at, '')
		FROM sep_products WHERE source=? AND category=?`, source, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSEProducts(rows)
}

// AllProductsByCategoryAcrossSources returns products grouped by normalized
// title to find cross-site overlaps for a given category. Used by
// `catalogue-diff` and `arbitrage`.
func (s *Store) AllProductsByCategoryAcrossSources(ctx context.Context, category string) ([]SEProduct, error) {
	if err := s.EnsureSEPricesSchema(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT source, source_id, name, brand, category, ean, url, image_url, COALESCE(lowest_price_sek, 0), COALESCE(last_seen_at, '')
		FROM sep_products WHERE category LIKE ?`, "%"+category+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSEProducts(rows)
}

// AllSEProducts returns every product in the store, capped at limit. Used by
// `arbitrage` when the caller does not scope the scan to a single category.
// Previously the no-category branch routed through SearchSEProducts(ctx, "*", N),
// which is invalid FTS5 syntax (silent fall-through to a LIKE '%*%' query that
// matches no rows). The result was that `arbitrage` without --category always
// returned zero rows regardless of how much had been synced.
func (s *Store) AllSEProducts(ctx context.Context, limit int) ([]SEProduct, error) {
	if err := s.EnsureSEPricesSchema(ctx); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 5000
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT source, source_id, name, brand, category, ean, url, image_url, COALESCE(lowest_price_sek, 0), COALESCE(last_seen_at, '')
		FROM sep_products
		ORDER BY name LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSEProducts(rows)
}
