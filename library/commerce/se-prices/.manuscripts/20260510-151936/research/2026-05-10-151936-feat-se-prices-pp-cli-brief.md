# se-prices CLI Brief

A unified command-line interface for **Prisjakt.nu** and **PriceRunner.se** —
the two largest Swedish price-comparison aggregators. The CLI absorbs each
site's discoverable surface and adds combo-only commands neither tool can offer
on its own.

## API Identity
- **Domain:** Consumer price-comparison for Swedish retail. Mainly electronics,
  appliances, fashion. Both sites also list service contracts (mobile, broadband).
- **Users:** Swedish shoppers comparing offers across retailers. Power users:
  arbitrage hunters, price-drop watchers, EAN/SKU lookup workflows.
- **Data profile:** Products (millions), offers (per product, per merchant),
  merchants (~1,000-5,000), categories, prices over time, reviews, ratings,
  curated deals.

## Reachability Risk
- **Prisjakt:** Mostly low. Category pages and product detail (legacy
  `/produkt.php?p=<id>`) return 200 with stdlib HTTP and a real browser UA.
  Search (`/search?search=<q>`) returns 403 stdlib (Cloudflare challenge), but
  **Surf with Chrome TLS fingerprint clears it (200)**. Internal GraphQL host
  `graphql.expert.prisjakt.nu` does not resolve externally.
- **PriceRunner:** Low for the surfaces we need. Search (`/results?q=<q>`),
  product detail (`/pl/<group>-<id>/<cat>/<slug>-priser`), and category
  (`/cl/<id>/<slug>`) all return 200 with stdlib + browser UA. Internal
  `/api/*.json` returns 403 even with Surf — but that surface is unnecessary
  because the SSR HTML embeds the full data state.
- **No Cloudflare wall on the data layer for either site.** This run does not
  share the Booli-class problem (no Chrome required at runtime).

## Top Workflows
1. **Search by name** — "iphone 15", "bosch dishwasher", "sony tv 55" →
   matched products with current lowest price across both comparators.
2. **Get a product's offers** — for a specific product, list every merchant's
   current price, shipping, stock status, and link.
3. **Watch a product** — sync to local store, run periodically, surface price
   drops or new merchants.
4. **Cross-site arbitrage** — find products where one comparator has a lower
   offer than the other (because they index different merchants).
5. **Category browse with filters** — drill into "mobiltelefoner" by brand,
   price range, attribute (storage, RAM, color).

## Table Stakes
Every feature these competing tools have:
- Product search by name → list of matching products
- Product detail by ID → name, brand, category, image, current best price
- Offers per product → merchant, price, shipping, link (deep-link to merchant)
- Category browse with filters
- Price history (sparkline / time series)
- Merchant directory + per-merchant rating
- Curated deals page (PriceRunner has `/deals`)
- JSON output mode for agents
- Bookmark/track favorites locally

## Data Layer
- **Primary entities:** `product`, `offer`, `merchant`, `category`,
  `price_snapshot` (per product, per timestamp), `tracked_product` (user's
  watch list), `arbitrage_signal` (computed from price snapshots).
- **Sync cursor:** Last-seen update timestamp per product (PriceRunner exposes
  `__SERVER_TIMESTAMP__`; Prisjakt via React Query state freshness).
- **FTS/search:** SQLite FTS5 on product `name`, `brand`, `category` for
  offline search and combo lookups.
- **Cross-site join:** Match by name + EAN/GTIN where Prisjakt exposes it; fall
  back to fuzzy normalized-title match for cross-site arbitrage detection.
- **Product ID stability:** Both sites use stable numeric IDs. Prisjakt:
  `product_id` (e.g., `14969878`). PriceRunner: `<group>-<numeric>` (e.g.,
  `1-3208336567`). IDs persist across price changes.

## Codebase Intelligence
**Discovery via direct HTTP** (no MCP/SDK source to read):

**Prisjakt** — Modern React app (post-Next.js).
- HTML carries `window.__REACT_QUERY_STATE__` wrapped in `JSON.parse('...')`.
- Product page query data shape (verified, real product `14969878`):
  `product { id, name, description, pathName, stockStatus, releaseDate,
  priceSummary, prices { meta, nodes, mobileContractsV2 }, category, brand,
  userReviewSummary, aggregatedRatingSummary, media, sparkline, popularity,
  dealInfo, expertContent, relations, variants, popularProducts,
  trendingProducts, metadata, verifiedProductBadge, sanityBadges, campaigns }`.
- Category page query data: `productCollection { ... }` with paginated product list.
- Brand filter: `?brand=<numeric_id>` (e.g., `/c/mobiltelefoner?brand=142`).
- Sitemap: 404 (no public sitemap).
- Documented APIs (require partnership credentials; out of scope without OAuth):
  - Insights API (GraphQL, `https://api.pj.nu/insights/`)
  - Partner Search API (`https://api.pj.nu/partner-search/`)
  - Ingest API + Click & Conversion API (merchant-side)
  - Developer portal: https://developer-docs.cloud.pji.nu/

**PriceRunner** — Server-rendered React (Klarna-owned since 2022).
- HTML carries `<script id="initial_payload">` (raw JSON, ~875KB-957KB) with:
  - `__INITIAL_PROPS__` containing per-page data: `Results` (search),
    `ProductList` + `reviewsPL` + `priceHistoryPL` + `productInformationPL` +
    `externalTestsPL` (product), category data on `/cl/`.
  - `__DEHYDRATED_QUERY_STATE__` (React Query cache).
  - `__SETTINGS__`, `__CONTEXT__`, `__SITE__`, `__SERVER_TIMESTAMP__`.
- Product URL pattern: `/pl/<group>-<numeric_id>/<category>/<slug>-priser`.
- Category URL: `/cl/<id>/<slug>`. Filter via `?attr_<numericAttrId>=<val>`.
- Search canonical URL: `/results?q=<query>` (`/search?q=` 301-redirects here).
- Hard-walled: `/api/*.json` (returns 403 + "Åtkomst nekad" page).
- Unusually hostile robots.txt (blanket `Disallow: /` for many crawlers).
- No public API (Klarna offers only payment/checkout APIs).

## Source Priority
- Equal peers (user confirmed). README leads with Prisjakt by user mention order.
- **Prisjakt:** stdlib HTTP for category/product, **Surf** for search → no auth.
- **PriceRunner:** stdlib HTTP for all reachable surfaces → no auth.
- **Economics:** Both free. No paid-tier split.
- **Inversion risk:** Low. Both have similar discovery quality (rich SSR JSON).
  Neither has a clean OpenAPI spec; both rely on direct HTTP HTML parsing.

## Product Thesis
- **Name:** `se-prices` (binary `se-prices-pp-cli`).
- **Why it should exist:** No CLI exists for either site, let alone a unified
  one. Both sites have rich data layers that are completely accessible via
  direct HTTP (with the right transport per site). A combo CLI unlocks
  arbitrage and catalogue-diff features that no single-site tool can produce.
  Power users get scriptable, agent-native price comparison without scraping
  HTML by hand.

## Cross-Site Combo Value
Features that require both sources, joined locally:
1. **Best-of-both lookup** — `lowest "iphone 15"` returns the cheapest offer
   across both comparators in one call.
2. **Arbitrage watch** — find products where the cheaper site undercuts the
   other by ≥X%.
3. **Catalogue diff** — products only one site indexes (especially niche Swedish
   merchants).
4. **Merchant overlap audit** — which retailers list a SKU on only one site.
5. **Trust triangulation** — flag review-score divergence per merchant
   between the two sites.
6. **Local cross-site price history** — neither site exposes the OTHER's
   history; the local store fills the gap as we sync.

## Build Priorities
1. **P0 — Foundation:** SQLite store with schemas for `product`, `offer`,
   `merchant`, `category`, `price_snapshot`, `tracked_product`, FTS5 indexes,
   `sync` command per source, `search`/`sql` offline commands.
2. **P1 — Per-source absorbed features:**
   - `prisjakt search`, `prisjakt product <id>`, `prisjakt category <slug>`,
     `prisjakt brand list/get`, `prisjakt deals`
   - `pricerunner search`, `pricerunner product <id>`, `pricerunner category <id>`,
     `pricerunner deals`
   - Each with `--json`, `--select`, `--csv`, `--limit`, structured output.
   - HTTP transport per site (Surf for Prisjakt search; stdlib elsewhere).
3. **P2 — Cross-site transcendence:** combo commands listed in the absorb manifest.
4. **P3 — Polish:** README, SKILL.md narrative, doctor diagnostics,
   transparent rate limiting, JSON-state extraction tested across site changes.
