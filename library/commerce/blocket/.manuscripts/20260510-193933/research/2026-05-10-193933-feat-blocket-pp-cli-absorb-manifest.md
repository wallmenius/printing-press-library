# Blocket CLI — Absorb Manifest

> Phase 1.5 manifest. Combines the absorbed-features set (everything any existing tool offers) and the transcendence-features set (10 survivors from the brainstorming subagent, all scoring ≥7/10).

## Ecosystem inventory

| Tool | Type | Stars/Status | What it covers |
|------|------|--------------|----------------|
| `dunderrrrrr/blocket_api` | Python SDK | 32 ⭐ | Public unauth wrapper: `search`, `search_car`, `search_boat`, `search_mc`, `get_ad`. Most active. |
| `siavashg/blocket-api` | Python SDK | 19 ⭐ | Official wrapper requiring `app_id`+`api_key` from Blocket support. `search()` only. |
| `bjesus/begagnad-mcp` | MCP server | 4 ⭐ | Combined Blocket + Tradera search via Cloudflare Workers. `search_blocket`, `get_blocket_item`. |
| `blocket-spy` | npm package | small | Watches Blocket for new listings, posts OS notifications. |
| `dan0/Blocket.se` | Node.js script | small | Polls Blocket and emails on new ads. |
| `WilhelmvonArndt/scraper-blocket` | Python | small | Faster (sub-30-min) ad notifications via scraping. |
| `henrik/blocket_se_feeds` | Ruby CGI | small | Atom feed of Blocket search. |
| `martinlarsalbert/blocket` | Python | small | Car search → CSV. |
| `vonj/simplescraper` | Python | small | MacBook scraper → JSON+Excel with spec extraction. |
| `emmmile/blocket` | Node.js | small | Stockholm rental ad map. |
| `blocket-api.se` (web service) | Hosted API | – | REST wrapper with `/v1/search`, `/v1/search/car`, `/v1/ad/`. |
| `Apify logiover/blocket-se-scraper` | SaaS scraper | – | Paid scraping platform. |

No CLI exists. No tool has a local store. No tool tracks price history. No tool offers cross-vertical search. No tool covers the 10 niche mobility verticals (truck/bus/agriculture/caravan/atv/scooter/etc.).

## Absorbed (match or beat everything that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-------------|-------------------|-------------|
| 1 | Free-text search (general items) | dunderrrrrr/blocket_api `search()` | `search` with `--query --category --location --price-from/to --sort --page --limit` | Local FTS5 over synced corpus + `--json/--select/--csv/--compact` for agents |
| 2 | Used-car search with full filter set | dunderrrrrr/blocket_api `search_car()` | `search-car` with `--make --model --year-from/to --mileage-from/to --fuel --transmission --wheel-drive --horsepower-from/to --color --location --price-from/to --org-id --sort --page` | All 17 filters from API `filters[]` tree, `--dry-run`, dotted `--select` for nested fields |
| 3 | Used-boat search | dunderrrrrr/blocket_api `search_boat()` | `search-boat` with `--type --location --length-from/to --price-from/to --org-id --sort --page` | Same agent-native flag set |
| 4 | Used-motorcycle search | dunderrrrrr/blocket_api `search_mc()` | `search-mc` with `--model --type --location --price-from/to --engine-volume-from/to --org-id --sort --page` | Same |
| 5 | B2B truck search | (manual discovery; no wrapper) | `search-mobility --vertical truck` | First wrapper to cover this — `SEARCH_ID_CAR_TRUCK`, ~4,664 listings |
| 6 | B2B bus / construction / agriculture-tractor / agriculture-tool / agriculture-combines / caravan / mobile-home / a-tractor / ATV / scooter | (manual discovery) | `search-mobility --vertical {bus,construction,agriculturetractor,agriculturetool,agriculturecombines,caravan,mobilehome,a-tractor,atv,scooter}` | First-mover on all 10 niche verticals |
| 7 | Get full ad detail | dunderrrrrr/blocket_api `get_ad()` | `ad get <id>` (auto-routes to vertical from canonical_url) | `--json` default, `--select` for nested fields |
| 8 | Filter discovery | (in-band only — no wrapper exposes it) | `filters list --vertical {car,boat,mc,bap}` | Probes one search response and pretty-prints `filters[]` so agents can introspect valid values |
| 9 | Cross-marketplace composability | bjesus/begagnad-mcp `search_blocket` | (covered by `search` + `--json` output) | Composes with downstream tools (Tradera, Finn, etc.) via shell pipes — no proprietary worker |
| 10 | Watch for new ads matching criteria | blocket-spy / dan0 / scraper-blocket | `watch add/list/run/sync` — stored named-searches, sync into local store, exit non-zero on new matches | Cron-friendly typed exit codes; structured JSON for triggering downstream actions |
| 11 | Atom/RSS feed | henrik/blocket_se_feeds | `search ... --output atom` / `--output rss` | Power users with existing RSS readers |
| 12 | CSV export | martinlarsalbert/blocket | (covered by `--csv` global flag) | Universal across every list command, not bespoke per category |
| 13 | JSON export with field selection | vonj/simplescraper | (covered by `--json --select` global flags) | Dotted-path selectors, not bespoke per-product extractors |
| 14 | Specific dealer/seller inventory | (in API as `org_id` filter, not as primary in any wrapper) | `dealer ads <org_id>` (and `dealer info <org_id>`) | First-class dealer command surface |

## Transcendence (only possible with our approach)

10 survivors from the brainstorming subagent (all scored ≥7/10 with the 4-dimension absorb rubric). Personas: Linus (used-car shopper), Sara (deal-flipper), Erik (dealer analyst), Margareta (LLM agent).

| # | Feature | Command | Score | Why Only We Can Do This | Persona |
|---|---------|---------|-------|------------------------|---------|
| 1 | What's new since timestamp | `since --search <name> --since <ts>` | 9/10 | Local SQLite query over `ads.timestamp` for stored named-search WHERE clauses; cron-friendly typed exit codes. The live API has no "since" endpoint. | Linus, Erik |
| 2 | Arbitrage / underpriced detector | `arbitrage --vertical car --make <m> --threshold 0.8` | 9/10 | Median price per (make, model, year-band, mileage-band) computed locally over synced ads; list current listings ≤ threshold × median. No API median endpoint exists. | Sara, Linus |
| 3 | Price history per ad | `price-history <ad_id>` | 8/10 | Sync re-fetches tracked ads and snapshots `ad_price_snapshots(ad_id, ts, amount)`. The API exposes only the current price. | Erik, Linus |
| 4 | Dealer portfolio + stats | `dealer ads <org_id> --stats` | 8/10 | Local aggregate over `ads` by `org_id` joined with snapshots: count, median price, oldest-listing-days, % price-dropped-this-week. No wrapper computes aggregates. | Erik, Sara |
| 5 | Stale listings | `stale --vertical <v> --older-than <Nd>` | 7/10 | `ads.timestamp` vs now joined with last-sync presence — listings that have been active longer than N days. | Erik |
| 6 | Cross-vertical search | `search-all "<query>" --max-price <N>` | 8/10 | Local FTS5 UNION across all 14 vertical-typed ad tables. Every wrapper is per-vertical; nobody does this. | Sara, Margareta |
| 7 | Watch with price-drop notification | `watch run <named-search> --notify-on price-drop` | 8/10 | Extends absorbed `watch run` with snapshot diff: detects price drops since last run, exits non-zero with structured JSON. Depends on transcendence #3. | Linus, Erik, Sara |
| 8 | Description regex | `desc-grep --vertical <v> --pattern <re>` | 7/10 | Sync backfills `ads.description` by ad-detail; command runs Go regexp locally. The search API matches headings only, not descriptions. | Sara |
| 9 | Geo radius search | `geo near --lat <x> --lon <y> --radius <km>` | 7/10 | Local haversine over `ads.coordinates`. The public search API only filters by Swedish region IDs, not arbitrary lat/lon. | Linus, Sara |
| 10 | Vehicle appraisal | `appraise --vertical car --make <m> --model <m> --year <y> --mileage <km>` | 7/10 | Comparable-set selection from synced corpus + p10/p50/p90 percentile aggregation. No API endpoint returns price distributions. | Linus |

## Stubs

None. All 10 transcendence features are shipping scope.

## Out of scope

- **Bevakningar / saved searches via api.blocket.se** — requires Bearer token from logged-in browser session; user declined this scope in Phase 1.6.
- **bostad.blocket.se (real estate)** — separate domain with aggressive anonymous rate-limiting (`429` on first request from clean IPs); unsuitable for a single-binary CLI without browser auth/clearance setup.
- **Posting/messaging/sending offers** — would require write-side auth; not addressed in v1.
- **B2B official Bytbil API integration** — requires dealer credentials; out of scope.
