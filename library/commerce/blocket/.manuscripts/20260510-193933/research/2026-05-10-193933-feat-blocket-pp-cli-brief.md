# Blocket CLI Brief

## API Identity
- **Domain:** Swedish second-hand classifieds — Sweden's largest marketplace (BAP "Buying And Selling", cars, motorcycles, boats, B2B vehicles, real estate, jobs).
- **Owner:** Schibsted (also runs Bytbil, Finn.no shares stack — Finn fonts/icons leak in HTML).
- **Users:** Mostly Swedish consumers and dealerships. Power users maintain saved searches (Bevakningar) for high-turnover items (cars, apartments, deals on hardware/electronics). Dealers and arbitrage hunters need fast filtering across millions of listings.
- **Data profile:** ~17,500 Volvo car listings alone; multi-million ad inventory. Per-ad data is rich (regno, mileage, fuel, transmission, dealer info, geocoordinates, multiple images, equipment lists, price). Hierarchical category tree (3 levels deep, Swedish display names: "Affärsverksamhet" → "Butik och detaljhandel" → "Butiksbelysning").

## Reachability Risk
- **None / Low.** `printing-press probe-reachability` returned `mode: standard_http` with confidence 0.95. Plain stdlib HTTP returns `200 application/json` directly from the public search endpoints. No bot-detection, no Cloudflare challenge, no clearance cookies needed for the primary read surface.
- **Watch-out:** `bostad.blocket.se` (real estate) returned `429` to anonymous curl — that subdomain has stricter rate limiting and is *out of scope* for the v1 CLI. The popular `dunderrrrrr/blocket_api` wrapper also doesn't cover bostad, confirming the boundary.
- **Sustainability check:** `dunderrrrrr/blocket_api` has been functional throughout 2025; only one community issue ("does it still work on new blocket version?") was filed in Nov 2025 and closed without follow-up. Endpoints are stable.

## Top Workflows
1. **Hunt a deal** — search Blocket for `<query>` filtered by location/price/category, sorted by recency or price-ascending; refine on filter chips; open the listing in browser. The signature use case.
2. **Track a saved-search inventory over time** — Bevakningar power users want to know what's *new* since yesterday and what *prices dropped*. Today this requires opening each saved search by hand.
3. **Comparison-shop a vehicle (used cars / boats / motorcycles)** — narrow by year, mileage, fuel, transmission, body type, geographic radius; compare across pages. The mobility verticals (`SEARCH_ID_CAR_USED`, etc.) have rich filter sets.
4. **Find arbitrage / undervalued listings** — locate items priced below the median for their category+brand+condition. No existing tool does this; requires local aggregation across pages.
5. **Watch a specific seller / dealer (org_id)** — see everything a particular dealer is selling, monitor their inventory turnover. Possible via `org_id` filter today, but tedious manually.

## Table Stakes
Every Blocket tool that exists today (3 wrappers + 1 MCP) covers some subset:
- Free-text search across ads (`q`)
- Category-filtered search (BAP categories: 12 top-level, hundreds of leaves)
- Location filter (Swedish counties / cities, radius, polygon)
- Price range
- Sort order (newest, price asc/desc, distance)
- Pagination (`page`, default 50 per page)
- Per-vertical search: cars, boats, motorcycles
- Vehicle-specific filters: year, mileage, fuel, transmission, model, color, horsepower, wheel-drive, dealer ID
- Get full ad details by ID

The MCP server `bjesus/begagnad-mcp` adds: cross-marketplace search (Blocket + Tradera).

The Python wrapper `dunderrrrrr/blocket_api` (32 stars) is the reference implementation of the public surface.

The Python wrapper `siavashg/blocket-api` (19 stars) covers the *official* dealer-only API (requires `app_id`+`api_key` from Blocket support).

## Data Layer
- **Primary entities:**
  - `ads` (across BAP, Cars, Boats, MCs — typed by `vertical` + `main_search_key`); shared fields: `ad_id`, `heading`, `price.amount`, `location`, `coordinates`, `image_urls`, `timestamp`, `canonical_url`, `category`, `trade_type`, `flags` (private vs dealer); per-vertical fields: `make/model/year/mileage/fuel/transmission/regno/dealer_segment` for cars, etc.
  - `categories` (hierarchical 3-level tree: category 0.91 → sub_category 1.91.3108 → product_category 2.91.3108.362; Swedish + English-numeric IDs)
  - `dealers` / `organisations` (`org_id`, `organisation_name`, `dealer_segment` — implicit from ads)
  - `saved_searches` (Bevakningar — only with auth)
- **Sync cursor:** `timestamp` (Unix milliseconds, monotonic per ad publish) — perfect for `--since` and incremental sync. `metadata.timestamp` on the response, plus per-doc `timestamp`.
- **FTS/search:** Free-text against `heading`, `location`, `make`, `model`, `organisation_name`, `category` display name. Power-user workflows want offline regex against descriptions, which the API doesn't expose by default — fetching ad detail backfills this.
- **Local-store wins (transcendence territory):**
  - Cross-vertical join (find Volvos under 50k SEK across cars, MCs, parts in BAP)
  - Price-history per ad (fetch detail repeatedly, snapshot `price.amount`)
  - Median-price-per-category-and-make computed locally
  - Stale listing detection (`timestamp` older than N days, still active)
  - "What's new since yesterday" queries that no live API exposes

## Codebase Intelligence
- **Source:** `dunderrrrrr/blocket_api` (Python wrapper) source code analysis
- **Endpoints (verified working):**
  - `GET https://www.blocket.se/recommerce/forsale/search/api/search/SEARCH_ID_BAP_COMMON` (200 JSON, no auth)
  - `GET https://www.blocket.se/mobility/search/api/search/SEARCH_ID_CAR_USED` (200 JSON, no auth)
  - `GET https://www.blocket.se/mobility/search/api/search/SEARCH_ID_BOAT_USED` (200 JSON, no auth)
  - `GET https://www.blocket.se/mobility/search/api/search/SEARCH_ID_MC_USED` (200 JSON, no auth)
  - `GET https://www.blocket.se/recommerce/forsale/item/{ad_id}` (200 JSON with `Accept: application/json`)
  - `GET https://api.blocket.se/searches` (401 without bearer; needs token from logged-in browser)
  - `GET https://api.blocket.se/search_bff/v2/content/{ad_id}` (401 without bearer; needs token)
- **Auth (optional, for power-user features only):**
  - Token type: opaque bearer (long-lived)
  - Header: `Authorization: Bearer <token>`
  - Capture: log in to blocket.se → "Bevakningar" → DevTools → Network tab → `api.blocket.se/searches` → copy `Authorization` header value
  - Env var pattern: `BLOCKET_TOKEN`
- **Architecture:**
  - Two CDN host families: `www.blocket.se` (public, search/list/detail) and `api.blocket.se` (authed, user-data — searches, content)
  - Response envelope: `{ docs[], filters[], metadata{paging, search_key, vertical, sort, descriptions, tracking, ...} }` for searches
  - Pagination: `page` query param, `metadata.paging.last` for total pages, `is_end_of_paging`
  - Filter discovery: every search response returns the filters tree with current-state values (selected/hits) — agents can introspect the filter set per query
- **Rate limiting:** No documented limits on `www.blocket.se`; subdomain `bostad.blocket.se` clearly throttles anon traffic (429 on first request from clean IP). v1 CLI scopes to www.blocket.se only.

## User Vision
- (User selected "Let's go" — no additional vision context provided.)

## Product Thesis
- **Name:** `blocket-pp-cli` (binary `blocket-pp-cli`).
- **Display name:** Blocket
- **Why it should exist:** Blocket has 4 partial wrappers and 1 MCP server — none is a real CLI, none has offline search or local state, none surfaces price changes or arbitrage. Power users (apartment hunters, used-car shoppers, deal-flippers) repeatedly poll multiple saved searches by hand. A CLI with a local SQLite store of synced inventory unlocks: cross-vertical aggregations, price-drop detection, stale-listing pruning, dealer-portfolio views, and offline regex over descriptions — none of which the live API or any wrapper offers. This is also the first Blocket tool agents can use natively (the existing MCP is a thin search shim over a wrapper service).

## Build Priorities
1. **Foundation (Priority 0):** SQLite store with `ads` (typed unions for BAP/Car/Boat/MC), `categories` tree, `dealers`, `saved_searches` (auth-gated), FTS5 across `heading`/`organisation_name`/free-text, sync cursor on `timestamp`. Endpoint-mirror commands for the 5 verified GETs.
2. **Absorb (Priority 1):** Match every wrapper's surface — `search` (general), `search-car`, `search-boat`, `search-mc`, `get-ad`, plus all filter parameters (year, mileage, fuel, transmission, location, price-range, dealer/org_id, color, horsepower, wheel-drive, engine-volume). Match the begagnad MCP's combined-search shape *only by exposing each via the standard `--json/--select/--csv/--limit` output flags*.
3. **Transcend (Priority 2):** Local-only analytics — `arbitrage` (under-priced detection per make+year), `price-track` (snapshot prices over time), `since` (what's new since timestamp X across saved searches), `stale` (listings older than N days still active), `dealer` (full dealer-inventory query with stats), `categories` (offline browsable category tree), `bevakningar` (saved-search browse + delta + alerts — auth-gated).

## Source Priority
- Single-source CLI (Blocket only). No combo-priority gate triggered.
