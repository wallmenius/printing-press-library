# Blocket Discovery Report

## Outcome

**Browser-sniff approved by user, but tooling failed.** Falling back to passive HTML inspection + manual endpoint probing, which produced strong results: every additional SEARCH_ID variant in Blocket's listing tree was discovered, all five primary endpoints verified live, and response schemas captured.

## Tooling Status

- `browser-use` (uvx-installed): **broken on Python 3.14** — raises `RuntimeError: There is no current event loop in thread 'MainThread'` from `asyncio.get_event_loop()` in `browser_use/skill_cli/main.py:907`. Runtime has no Python ≤ 3.13 wired into uvx by default.
- `agent-browser 0.10.0`: installed but **bundled Chromium version mismatch** — looks for `chromium_headless_shell-1208` while Playwright provides `chromium_headless_shell-1217`.
- `chrome-MCP` (`mcp__claude-in-chrome__*`): **not exposed** in this runtime's deferred tool catalog.
- Manual HAR fallback was available but not requested — passive probing already covered the user's discovery goal (additional endpoints) without round-tripping through devtools export.

Per the SKILL's cardinal rule against tool-integration debugging, attempts were time-boxed and pivoted to manual probing.

## Endpoints Verified (all 200 OK, no auth required)

| Method | URL pattern | Response | Notes |
|---|---|---|---|
| GET | `https://www.blocket.se/recommerce/forsale/search/api/search/SEARCH_ID_BAP_COMMON` | JSON `{docs[], filters[], metadata}` | General items (BAP). 50 results/page. Filters: q, category, shipping_types, polylocation, radius, bbox, location, trade_type, price, dealer_segment, condition, published. |
| GET | `https://www.blocket.se/mobility/search/api/search/SEARCH_ID_CAR_USED` | JSON same shape | Used cars. 17,531 "volvo" matches at probe time. Per-doc fields include `make/model/year/mileage/fuel/transmission/regno/dealer_group_id/dealer_segment/horsepower/wheel_drive`. Filters: q, location, polylocation, radius, bbox, variant, fuel, body_type, sales_form, price, year, mileage, dealer_segment, transmission, wheel_drive, engine_effect, car_equipment, exterior_colour, vat_deductible, published, wheel_sets. |
| GET | `https://www.blocket.se/mobility/search/api/search/SEARCH_ID_BOAT_USED` | JSON same shape | Used boats. |
| GET | `https://www.blocket.se/mobility/search/api/search/SEARCH_ID_MC_USED` | JSON same shape | Used motorcycles. |
| GET | `https://www.blocket.se/recommerce/forsale/item/{ad_id}` | JSON `{isPreview,itemData,…}` | Full ad detail. Requires `Accept: application/json` header. Returns title, location, position+map links, full description, seller info, image grid. |

## Additional SEARCH_IDs Discovered via HTML Probing

The same `/mobility/search/api/search/{SEARCH_ID}` pattern works for ten additional verticals — all returned 200 OK with the same JSON envelope:

- `SEARCH_ID_CAR_TRUCK` — Trucks (vertical: B2B, 4,664 listings at probe)
- `SEARCH_ID_CAR_BUS` — Buses
- `SEARCH_ID_CAR_AGRI` — Construction equipment
- `SEARCH_ID_AGRICULTURE_TRACTOR` — Agriculture tractors
- `SEARCH_ID_AGRICULTURE_TOOL` — Agriculture tools
- `SEARCH_ID_AGRICULTURE_THRESHING` — Agriculture combines
- `SEARCH_ID_CAR_CARAVAN` — Caravans
- `SEARCH_ID_CAR_MOBILE_HOME` — Mobile homes
- `SEARCH_ID_CAR_A_TRACTOR` — A-tractors
- `SEARCH_ID_MC_ATV` — ATVs
- `SEARCH_ID_MC_SCOOTER` — Scooters

The CLI will expose all 14 verticals via a single typed `search` resource with a `--vertical` flag (or per-vertical sub-commands for mobility specialists).

## Endpoints Confirmed Auth-Only (out of scope per user choice)

- `https://api.blocket.se/searches` (Bevakningar / saved searches) — 401 without bearer token
- `https://api.blocket.se/search_bff/v2/content/{ad_id}` — 401 without bearer token

User selected "Public endpoints only" in Phase 1.6, so these are excluded from v1 surface.

## Endpoints That Don't Exist or Aren't Exposed Anonymously

- `bostad.blocket.se/api/listings` — `429 Too Many Requests` from clean IPs (real estate vertical aggressive-throttles anon traffic). Out of scope for v1.
- `/jobb/api/search` — 404 (jobs vertical uses different routing, no public API entry verified).
- `/recommerce/forsale/api/{categories,locations,trending,autocomplete}` — all 404. Categories are returned inline in every search response's `filters[]` tree; no separate categories endpoint needed.

## Reachability

`printing-press probe-reachability https://www.blocket.se` returned `mode: standard_http`, confidence 0.95 — plain stdlib HTTP works fine. No clearance cookies, no Surf-style Chrome fingerprint needed. Generated CLI ships standard transport.

## Replayability

All five verified endpoints are simple HTTP GETs with stable URL patterns and JSON responses. Pagination is `?page=N` with `metadata.is_end_of_paging` and `metadata.paging.last`. Full replayability — no GraphQL persisted hashes, no proxy envelopes, no live page-context execution.

## Categories Tree

Returned inline in every search response's `filters[]` array under `name=category`. Three-level hierarchy with Swedish display names:
- Level 0: top category (e.g., `0.91` Affärsverksamhet)
- Level 1: sub_category (e.g., `1.91.3108` Butik och detaljhandel)
- Level 2: product_category (e.g., `2.91.3108.362` Butiksbelysning)

The CLI populates a local `categories` table from the first sync response and uses it for offline category browse + auto-completion of `--category` flag values.

## Recommendations for Generation

1. Use a single internal spec covering all 14 SEARCH_IDs as a `vertical` enum.
2. Endpoints: `search`, `search-mobility` (with vertical flag), `get-ad`.
3. Local store: `ads`, `categories`, `dealers`, `query_runs` (for delta tracking).
4. Skip Bevakningar (`api.blocket.se`) per user choice.
5. Skip `bostad.blocket.se` per rate-limit risk.
