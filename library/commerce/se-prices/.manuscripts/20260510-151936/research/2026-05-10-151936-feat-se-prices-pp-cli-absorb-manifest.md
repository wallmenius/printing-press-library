# se-prices Absorb Manifest

Combo CLI for **Prisjakt.nu** and **PriceRunner.se**. Every feature either site
exposes is absorbed; cross-site combo features no single-site tool can offer
form the transcendence table.

## Source tools surveyed

| Source | Repo / URL | Status | Contribution |
|--------|-----------|--------|--------------|
| freedick/prisjakt | github.com/freedick/prisjakt | Dead (PHP `ajax/server.php` retired 404) | Historical endpoint reference only; no live features |
| serpis/pynik prisjakt plugin | github.com/serpis/pynik | Dead (same retired PHP) | None |
| Apify studio-amba/prisjakt-scraper | apify.com | Low-signal, paid platform | None |
| Prisjakt Partner Search API | api.pj.nu/partner-search/ | Real, gated behind OAuth/partnership | Out of scope (no anonymous access); shape inspiration |
| Prisjakt Insights GraphQL | api.pj.nu/insights/ | Same | Same |
| rstaniek/pricerunner-product-notifications | github.com/rstaniek/... | Low-star Python HTML scraper for pricerunner.dk | None (wrong country, fragile) |
| Apify m3web/pricerunner-product-offers-scraper | apify.com | Paid platform | None |
| PriceRunner GitHub org | github.com/PriceRunner | Empty | None |
| Klarna API docs | docs.klarna.com | Payments only, no PriceRunner data API | None |

**Net signal:** Zero maintained community wrappers, zero MCPs, zero CLIs. Every
absorbed feature comes from direct HTTP discovery of each site's SSR JSON state.

## Absorbed (match every feature both sites expose)

### From Prisjakt (REACT_QUERY_STATE in HTML)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|------------|--------------------|-------------|
| 1 | Search products by name | `/search?search=` (Surf) → REACT_QUERY_STATE | `prisjakt search "<query>"` | --json, --select, --limit, --csv; offline FTS5 after sync |
| 2 | Get product detail | `/produkt.php?p=<id>` → product object | `prisjakt product <id>` | Bundles brand+category+badges+ratings in one call |
| 3 | List offers per product | `product.prices.nodes` | `prisjakt offers <id>` | --in-stock, --max-shipping filters |
| 4 | Price summary (lowest/highest) | `product.priceSummary` | embedded in `prisjakt product` | --json |
| 5 | Price history sparkline | `product.sparkline` | `prisjakt history <id>` | --window N --csv for spreadsheet |
| 6 | Mobile contracts comparison | `product.prices.mobileContractsV2` | `prisjakt contracts <id>` | --json |
| 7 | Browse category | `/c/<slug>` productCollection | `prisjakt category <slug>` | --limit, --offset, --select |
| 8 | Filter category by brand | `/c/<slug>?brand=<id>` | `prisjakt category <slug> --brand <id>` | Multi-brand: `--brand 142 --brand 8` |
| 9 | Brand detail | `product.brand` | `prisjakt brand <id>` | --json, agent-friendly |
| 10 | Popular products in category | `category.products` | `prisjakt popular --category <slug>` | --limit |
| 11 | Trending products | `product.trendingProducts` | `prisjakt trending --category <slug>` | --limit |
| 12 | User-review summary + ratings | `product.userReviewSummary` + `aggregatedRatingSummary` | embedded in product output | --json |
| 13 | Expert content | `product.expertContent` | `prisjakt expert <id>` | --json |
| 14 | Product variants | `product.variants` | `prisjakt variants <id>` | --json |
| 15 | Related products | `product.relations` | `prisjakt related <id>` | --limit, --select |
| 16 | Product specs / properties | `product.coreProperties` | `prisjakt specs <id>` | --json, --select |
| 17 | Verified product badges | `product.verifiedProductBadge` | embedded in product | --json |
| 18 | Deal info on a product | `product.dealInfo` | embedded in product | --json |
| 19 | Campaigns on a product | `product.campaigns` | embedded in product | --json |
| 20 | FAQ for a product | `product.sanityFaq` | `prisjakt faq <id>` | --json |

### From PriceRunner (`<script id="initial_payload">` in HTML)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|------------|--------------------|-------------|
| 21 | Search products | `/results?q=` initial_payload.Results | `pricerunner search "<query>"` | --json, --select, --limit, --csv |
| 22 | Get product detail | `/pl/<group>-<id>/.../-priser` initial_payload.__INITIAL_PROPS__.ProductList | `pricerunner product <id>` | Bundles offers+reviews+price-history+specs+tests in one call |
| 23 | List offers per product | `ProductList.offers` | `pricerunner offers <id>` | --in-stock, --max-shipping filters |
| 24 | Product reviews | `reviewsPL` | `pricerunner reviews <id>` | --limit, --json |
| 25 | Product price history | `priceHistoryPL` | `pricerunner history <id>` | --window N --csv |
| 26 | Product info / specs | `productInformationPL` | `pricerunner specs <id>` | --json, --select |
| 27 | External / expert tests | `externalTestsPL` | `pricerunner tests <id>` | --json |
| 28 | Browse category | `/cl/<id>/<slug>` | `pricerunner category <id>` | --limit, --offset |
| 29 | Filter category by attribute | `?attr_<id>=<value>` | `pricerunner category <id> --attr key=value` | Multi-attr support |
| 30 | Curated deals | `/deals` | `pricerunner deals` | --json, --category filter |

### Foundation (every printed CLI gets these)
- `sync [--source prisjakt|pricerunner|both] [--category ...] [--full]` — populate local SQLite store
- `search "<query>"` — FTS5 across local store, both sources
- `sql "<select>"` — read-only SELECT against local store
- `doctor` — health check (reachability per source, store size, last sync)
- `agent-context` — agent-friendly enumerated command tree
- `auth status` (no-op since no auth required, but doctor reports "auth: none")

## Transcendence (cross-site combo, only possible with our approach)

| # | Feature | Command | Score | How It Works | Evidence |
|---|---------|---------|-------|--------------|----------|
| 1 | Cross-site arbitrage scan | `arbitrage --category <slug> --min-gap 10 [--min-sek 200]` | 9/10 | Joins synced `offer` rows from both sources on EAN (fallback: normalized title), groups by product, computes lowest-per-source, filters where `min_gap_pct >= flag` | Brief Cross-Site Combo §2; Maja persona's bottleneck is cross-site product matching |
| 2 | Stock-aware best-of-both lowest | `lowest "<query>" [--in-stock] [--max-shipping <sek>] [--ean <ean>]` | 9/10 | FTS5 lookup over local `product` table, then SQL aggregate `MIN(price + shipping) WHERE stock_status = 'in_stock'` over `offer` rows from both sources joined by product | Brief Top Workflows §1, Cross-Site Combo §1; Anders' nightly tab-comparison ritual; Karl's agent use case |
| 3 | EAN cross-link | `ean <ean>` | 8/10 | Local lookup in `product` table by EAN column, returns both Prisjakt `product_id` and PriceRunner `<group>-<id>`, plus the union of current offers | Brief Data Layer (EAN/GTIN cross-site join called out); Anders persona's "manually pasting EANs" frustration |
| 4 | Catalogue diff per category | `catalogue-diff --category <slug> [--only prisjakt\|pricerunner]` | 8/10 | Set-difference SQL: products tagged with one source's category but absent from the other site's index after a fresh `sync` of both | Brief Cross-Site Combo §3; Maja persona, niche Swedish merchants |
| 5 | Cross-site price-drop digest | `drops --since 7d [--min-pct 10] [--watched-only]` | 8/10 | Time-window diff over `price_snapshot` joined to `tracked_product`, grouped per product, returns rows where `latest.price <= earliest.price * (1 - min-pct/100)` | Brief Top Workflows §3; Elin and Anders personas' weekly ritual |
| 6 | Cross-site watchlist with thresholds | `watchlist add <id-or-ean> --max <sek>`, `watchlist list`, `watchlist check` | 7/10 | `tracked_product` table with `(source, product_key, ean, max_price_sek)`; `check` runs the cross-source lowest query and reports items at/below threshold | Brief Table Stakes ("Bookmark/track favorites locally") + Cross-Site Combo §1; Elin persona |
| 7 | Unified cross-site price history | `history-combo <ean-or-id> [--window 90d]` | 7/10 | Time-series merge of `price_snapshot` rows tagged by source for the same product (resolved via EAN), output as a single sorted series with per-row source label | Brief Cross-Site Combo §6 (explicitly: "neither site exposes the OTHER's history") |
| 8 | Sale-season anomaly check | `is-sale <id-or-ean> [--window 90d]` | 6/10 | Pure local stat over `price_snapshot`: returns current price, 90-day median, 90-day min, and a boolean `actually_a_sale = current < median * 0.9` | Brief mentions Black Week / July sales seasonality; Anders persona; mechanical, no LLM |

(Killed candidates with reasons preserved in
`research/2026-05-10-151936-novel-features-brainstorm.md`.)

## Stubs

None planned. All transcendence features are pure-local SQL/aggregation over
synced data — buildable from the data layer that ships in P0 + P1.
