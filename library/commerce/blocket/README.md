# Blocket CLI

**Every Blocket vertical, every filter, plus the local store, price history, arbitrage detection, and cross-vertical search no other Blocket tool offers.**

Blocket-pp-cli matches every existing wrapper's surface across the BAP and mobility verticals — including the ten niche ones (truck, bus, agriculture, caravan, ATV, scooter, mobile-home, a-tractor, agriculture-tractor, agriculture-tool) that no wrapper covers — and adds a SQLite-backed local store that unlocks workflows the live API cannot answer: what's new since yesterday, which listings are underpriced relative to their cohort, what a dealer's portfolio looks like, and how a vehicle's price compares to the rolling p10/p50/p90 distribution. Every command emits agent-native JSON with `--select` for nested fields and typed exit codes for cron and watcher pipelines.

Learn more at [Blocket](https://www.blocket.se).

## Install

The recommended path installs both the `blocket-pp-cli` binary and the `pp-blocket` agent skill in one shot:

```bash
npx -y @mvanhorn/printing-press install blocket
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press install blocket --cli-only
```


### Without Node (Go fallback)

If `npx` isn't available (no Node, offline), install the CLI directly via Go (requires Go 1.26.3 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/commerce/blocket/cmd/blocket-pp-cli@latest
```

This installs the CLI only — no skill.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/blocket-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-blocket --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-blocket --force
```

## Install for OpenClaw

Tell your OpenClaw agent (copy this):

```
Install the pp-blocket skill from https://github.com/mvanhorn/printing-press-library/tree/main/cli-skills/pp-blocket. The skill defines how its required CLI can be installed.
```

## Authentication

No authentication required. Every command uses Blocket's public, unauthenticated endpoints. Saved searches (Bevakningar) are intentionally out of scope in v1 — the local `watch` command is the agent-native equivalent.

## Quick Start

```bash
# Confirm the binary is healthy and the public endpoints are reachable from your network.
blocket-pp-cli doctor


# Most recent BAP listings matching airpods, JSON shape ready to pipe.
blocket-pp-cli ads list --query "airpods" --json


# Used Volvos from 2010+ under 80,000 SEK — the canonical used-car query.
blocket-pp-cli cars list --make Volvo --year-from 2010 --price-to 80000 --json


# Sample one of the ten niche mobility verticals nobody else exposes.
blocket-pp-cli mobility trucks --json


# Surface XC70 listings priced ≤ 80% of the local-corpus median for their year+mileage band.
blocket-pp-cli arbitrage --vertical car --make Volvo --model XC70 --threshold 0.8 --json


# Price a specific vehicle against the corpus distribution.
blocket-pp-cli appraise --vertical car --make Volvo --model XC70 --year 2014 --mileage 18000 --json

```

## Unique Features

These capabilities aren't available in any other tool for this API.

### Local state that compounds
- **`since`** — List ads added since a timestamp for any stored named-search, with cron-friendly exit codes when matches are found.

  _Use this when you need to know what is genuinely new versus what the user already saw, without rerunning a paginated search._

  ```bash
  blocket-pp-cli since --search xc70 --since 2026-05-09 --json
  ```
- **`arbitrage`** — Find current listings priced below threshold × median for their (make, model, year-band, mileage-band) cohort.

  _Use this to surface deals an agent could not detect from a single search — the comparison set lives only in the local store._

  ```bash
  blocket-pp-cli arbitrage --vertical car --make Volvo --threshold 0.8 --json
  ```
- **`price-history`** — Show every snapshotted price for an ad over time with deltas; sync populates `ad_price_snapshots` on each run.

  _Use this to confirm a price is genuinely lower than recent asking prices, not just lower than the user remembered._

  ```bash
  blocket-pp-cli price-history 22587669 --json
  ```
- **`dealer ads`** — Aggregate every current ad for an `org_id` plus inventory count, median price, oldest-listing-days, and percent price-dropped-this-week.

  _Use this to characterise a dealer's portfolio and turnover before contacting them, instead of scrolling their profile._

  ```bash
  blocket-pp-cli dealer ads 1234567 --stats --json
  ```
- **`stale`** — Listings older than N days that are still active in the most recent sync, optionally filtered by vertical or dealer.

  _Use this to find listings ripe for negotiation or to rule out stale corpora when comparing prices._

  ```bash
  blocket-pp-cli stale --vertical car --older-than 60d --json
  ```
- **`watch run`** — Run a stored named-search and exit non-zero with structured JSON when ads in the result set dropped in price since the previous run.

  _Use this in cron jobs or agent scheduled tasks to react only to genuine price drops, not every refresh._

  ```bash
  blocket-pp-cli watch run xc70 --notify-on price-drop --json
  ```
- **`desc-grep`** — Match a Go regular expression against ad description text backfilled by ad-detail sync.

  _Use this when the deal-defining keyword ("rökfri", "nyservad", "original") only appears in the description body, not the title._

  ```bash
  blocket-pp-cli desc-grep --vertical car --pattern "\\bnyservad\\b|\\brökfri\\b" --json
  ```
- **`appraise`** — Given a make/model/year/mileage, compute p10/p50/p90 of asking prices from the synced corpus and render the distribution.

  _Use this to price a hypothetical vehicle (yours or one you are about to bid on) against the current market without scraping listings by hand._

  ```bash
  blocket-pp-cli appraise --vertical car --make Volvo --model XC70 --year 2014 --mileage 18000 --json
  ```

### Cross-vertical leverage
- **`search-all`** — Free-text search that fans across BAP, used cars, boats, motorcycles, and the ten niche mobility verticals locally and merges results.

  _Use this when an item could live in any vertical (e.g., a Volvo could be a car, a part in BAP, or a model in MC)._

  ```bash
  blocket-pp-cli search-all "Volvo" --max-price 50000 --json
  ```
- **`geo near`** — Filter synced ads by point + radius using haversine distance against `coordinates`.

  _Use this when the user defines proximity as "within 30 km of my postcode", not as a Swedish administrative region._

  ```bash
  blocket-pp-cli geo near --lat 59.33 --lon 18.06 --radius 30 --vertical car --json
  ```

## Usage

Run `blocket-pp-cli --help` for the full command reference and flag list.

## Commands

### ads

Browse and fetch general BAP (Buying And Selling) listings — everything that is not a vehicle.

- **`blocket-pp-cli ads get`** - Fetch full detail for a BAP listing.
- **`blocket-pp-cli ads list`** - List general BAP listings (SEARCH_ID_BAP_COMMON).

### boats

Search used-boat listings (SEARCH_ID_BOAT_USED).

- **`blocket-pp-cli boats list`** - List used-boat listings.

### cars

Search and fetch used-car listings (SEARCH_ID_CAR_USED).

- **`blocket-pp-cli cars get`** - Fetch full detail for a vehicle listing.
- **`blocket-pp-cli cars list`** - List used-car listings.

### mobility

Niche mobility verticals — trucks, buses, construction, agriculture, caravans, mobile homes, A-traktor, ATVs, scooters.

- **`blocket-pp-cli mobility atractors`** - A-tractors / EPA-traktors (SEARCH_ID_CAR_A_TRACTOR).
- **`blocket-pp-cli mobility atvs`** - ATVs / fyrhjulingar (SEARCH_ID_MC_ATV).
- **`blocket-pp-cli mobility buses`** - Buses (SEARCH_ID_CAR_BUS).
- **`blocket-pp-cli mobility caravans`** - Caravans (SEARCH_ID_CAR_CARAVAN).
- **`blocket-pp-cli mobility combines`** - Agriculture combines / threshing machines (SEARCH_ID_AGRICULTURE_THRESHING).
- **`blocket-pp-cli mobility construction`** - Construction equipment (SEARCH_ID_CAR_AGRI).
- **`blocket-pp-cli mobility mobilehomes`** - Mobile homes (SEARCH_ID_CAR_MOBILE_HOME).
- **`blocket-pp-cli mobility scooters`** - Scooters (SEARCH_ID_MC_SCOOTER).
- **`blocket-pp-cli mobility tools`** - Agriculture tools (SEARCH_ID_AGRICULTURE_TOOL).
- **`blocket-pp-cli mobility tractors`** - Agriculture tractors (SEARCH_ID_AGRICULTURE_TRACTOR).
- **`blocket-pp-cli mobility trucks`** - Heavy trucks (SEARCH_ID_CAR_TRUCK).

### motorcycles

Search used-motorcycle listings (SEARCH_ID_MC_USED).

- **`blocket-pp-cli motorcycles list`** - List used-motorcycle listings.


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
blocket-pp-cli ads list

# JSON for scripting and agents
blocket-pp-cli ads list --json

# Filter to specific fields
blocket-pp-cli ads list --json --select id,name,status

# Dry run — show the request without sending
blocket-pp-cli ads list --dry-run

# Agent mode — JSON + compact + no prompts in one flag
blocket-pp-cli ads list --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Read-only by default** - this CLI does not create, update, delete, publish, send, or mutate remote resources
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `5` API error, `7` rate limited, `10` config error.

## Use with Claude Code

Install the focused skill — it auto-installs the CLI on first invocation:

```bash
npx skills add mvanhorn/printing-press-library/cli-skills/pp-blocket -g
```

Then invoke `/pp-blocket <query>` in Claude Code. The skill is the most efficient path — Claude Code drives the CLI directly without an MCP server in the middle.

<details>
<summary>Use as an MCP server in Claude Code (advanced)</summary>

If you'd rather register this CLI as an MCP server in Claude Code, install the MCP binary first:


```bash
go install github.com/mvanhorn/printing-press-library/library/commerce/blocket/cmd/blocket-pp-mcp@latest
```

Then register it:

```bash
claude mcp add blocket blocket-pp-mcp
```

</details>

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/blocket-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


```bash
go install github.com/mvanhorn/printing-press-library/library/commerce/blocket/cmd/blocket-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "blocket": {
      "command": "blocket-pp-mcp"
    }
  }
}
```

</details>

## Health Check

```bash
blocket-pp-cli doctor
```

Verifies configuration and connectivity to the API.

## Configuration

Config file: `~/.config/blocket-pp-cli/config.toml`

Static request headers can be configured under `headers`; per-command header overrides take precedence.

## Troubleshooting
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific

- **Search returns 0 results when you expect matches** — Run `blocket-pp-cli filters list --vertical <vertical>` to see the valid filter values; Blocket validates make/model/category enums strictly.
- **`HTTP 429` from any command** — Add `--page-delay 1s` to slow the request rate; the public endpoints are generous but a 429 means you have run too many parallel requests for too long.
- **`arbitrage` or `appraise` returns empty** — Run `blocket-pp-cli sync` first — these commands aggregate over the local store and need at least a few hundred rows in the relevant vertical.
- **`bostad` (real estate) commands missing** — Real estate is intentionally out of scope in v1; bostad.blocket.se rate-limits anonymous traffic aggressively. Use the website directly.

## HTTP Transport

This CLI uses Chrome-compatible HTTP transport for browser-facing endpoints. It does not require a resident browser process for normal API calls.

---

## Sources & Inspiration

This CLI was built by studying these projects and resources:

- [**dunderrrrrr/blocket_api**](https://github.com/dunderrrrrr/blocket_api) — Python (32 stars)
- [**siavashg/blocket-api**](https://github.com/siavashg/blocket-api) — Python (19 stars)
- [**bjesus/begagnad-mcp**](https://github.com/bjesus/begagnad-mcp) — TypeScript (4 stars)
- [**WilhelmvonArndt/scraper-blocket**](https://github.com/WilhelmvonArndt/scraper-blocket) — Python
- [**henrik/blocket_se_feeds**](https://github.com/henrik/blocket_se_feeds) — Ruby
- [**martinlarsalbert/blocket**](https://github.com/martinlarsalbert/blocket) — Python
- [**vonj/simplescraper**](https://github.com/vonj/simplescraper) — Python
- [**dan0/Blocket.se**](https://github.com/dan0/Blocket.se) — JavaScript

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
