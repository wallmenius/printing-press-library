---
name: pp-blocket
description: "Every Blocket vertical, every filter, plus the local store, price history, arbitrage detection, and cross-vertical... Trigger phrases: `find a deal on Blocket`, `search Swedish classifieds`, `find a used car on Blocket`, `compare Blocket prices`, `watch Blocket for new listings`, `use blocket`, `run blocket`."
author: "Johan Wallmén"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - blocket-pp-cli
    install:
      - kind: go
        bins: [blocket-pp-cli]
        module: github.com/mvanhorn/printing-press-library/library/commerce/blocket/cmd/blocket-pp-cli
---

# Blocket — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `blocket-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer:
   ```bash
   npx -y @mvanhorn/printing-press install blocket --cli-only
   ```
2. Verify: `blocket-pp-cli --version`
3. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

If the `npx` install fails (no Node, offline, etc.), fall back to a direct Go install (requires Go 1.26.3 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/commerce/blocket/cmd/blocket-pp-cli@latest
```

If `--version` reports "command not found" after install, the install step did not put the binary on `$PATH`. Do not proceed with skill commands until verification succeeds.

Blocket-pp-cli matches every existing wrapper's surface across the BAP and mobility verticals — including the ten niche ones (truck, bus, agriculture, caravan, ATV, scooter, mobile-home, a-tractor, agriculture-tractor, agriculture-tool) that no wrapper covers — and adds a SQLite-backed local store that unlocks workflows the live API cannot answer: what's new since yesterday, which listings are underpriced relative to their cohort, what a dealer's portfolio looks like, and how a vehicle's price compares to the rolling p10/p50/p90 distribution. Every command emits agent-native JSON with `--select` for nested fields and typed exit codes for cron and watcher pipelines.

## When to Use This CLI

Use blocket-pp-cli whenever an agent task involves Swedish second-hand listings: hunting deals across BAP and mobility verticals, comparison-shopping a vehicle, characterising a dealer's portfolio, watching for price drops on stored searches, or building a structured snapshot of a corner of the Swedish market for analysis. Prefer it over generic web fetching of blocket.se — every command emits structured JSON, exposes the full filter surface, and the local store keeps history that the live site discards.

## When Not to Use This CLI

Do not activate this CLI for requests that require creating, updating, deleting, publishing, commenting, upvoting, inviting, ordering, sending messages, booking, purchasing, or changing remote state. This printed CLI exposes read-only commands for inspection, export, sync, and analysis.

## Unique Capabilities

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

## HTTP Transport

This CLI uses Chrome-compatible HTTP transport for browser-facing endpoints. It does not require a resident browser process for normal API calls.

## Command Reference

**ads** — Browse and fetch general BAP (Buying And Selling) listings — everything that is not a vehicle.

- `blocket-pp-cli ads get` — Fetch full detail for a BAP listing.
- `blocket-pp-cli ads list` — List general BAP listings (SEARCH_ID_BAP_COMMON).

**boats** — Search used-boat listings (SEARCH_ID_BOAT_USED).

- `blocket-pp-cli boats` — List used-boat listings.

**cars** — Search and fetch used-car listings (SEARCH_ID_CAR_USED).

- `blocket-pp-cli cars get` — Fetch full detail for a vehicle listing.
- `blocket-pp-cli cars list` — List used-car listings.

**mobility** — Niche mobility verticals — trucks, buses, construction, agriculture, caravans, mobile homes, A-traktor, ATVs, scooters.

- `blocket-pp-cli mobility atractors` — A-tractors / EPA-traktors (SEARCH_ID_CAR_A_TRACTOR).
- `blocket-pp-cli mobility atvs` — ATVs / fyrhjulingar (SEARCH_ID_MC_ATV).
- `blocket-pp-cli mobility buses` — Buses (SEARCH_ID_CAR_BUS).
- `blocket-pp-cli mobility caravans` — Caravans (SEARCH_ID_CAR_CARAVAN).
- `blocket-pp-cli mobility combines` — Agriculture combines / threshing machines (SEARCH_ID_AGRICULTURE_THRESHING).
- `blocket-pp-cli mobility construction` — Construction equipment (SEARCH_ID_CAR_AGRI).
- `blocket-pp-cli mobility mobilehomes` — Mobile homes (SEARCH_ID_CAR_MOBILE_HOME).
- `blocket-pp-cli mobility scooters` — Scooters (SEARCH_ID_MC_SCOOTER).
- `blocket-pp-cli mobility tools` — Agriculture tools (SEARCH_ID_AGRICULTURE_TOOL).
- `blocket-pp-cli mobility tractors` — Agriculture tractors (SEARCH_ID_AGRICULTURE_TRACTOR).
- `blocket-pp-cli mobility trucks` — Heavy trucks (SEARCH_ID_CAR_TRUCK).

**motorcycles** — Search used-motorcycle listings (SEARCH_ID_MC_USED).

- `blocket-pp-cli motorcycles` — List used-motorcycle listings.


**Hand-written commands**

- `blocket-pp-cli search-all` — Cross-vertical free-text search across BAP, cars, boats, MCs, and the niche mobility verticals from the local store.
- `blocket-pp-cli since` — List ads added since a timestamp for a stored named-search; cron-friendly typed exit codes.
- `blocket-pp-cli arbitrage` — Find current listings priced ≤ threshold × median for their (make, model, year-band, mileage-band) cohort.
- `blocket-pp-cli price-history` — Show all snapshotted prices for an ad over time with deltas, populated by sync re-fetching tracked ads.
- `blocket-pp-cli dealer` — Dealer portfolio commands — list ads, aggregate stats, and turnover analysis by org_id.
- `blocket-pp-cli stale` — Listings older than N days that are still active in the most recent sync.
- `blocket-pp-cli watch` — Stored named-searches with sync, run, and price-drop notification commands.
- `blocket-pp-cli desc-grep` — Match a Go regular expression against ad descriptions backfilled into the local store.
- `blocket-pp-cli geo` — Geo radius search — filter synced ads by point + radius using haversine distance.
- `blocket-pp-cli appraise` — Compute p10/p50/p90 of asking prices for a comparable vehicle set from the synced corpus.
- `blocket-pp-cli filters` — List valid filter values for a vertical by probing one search response — agents introspect the filter set.


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
blocket-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Recipes


### Five fresh BAP listings as a JSON pipeline

```bash
blocket-pp-cli ads list --query "festool" --json --select docs.heading,docs.price.amount,docs.location,docs.canonical_url
```

Narrows a verbose search response down to the agent-relevant fields; canonical for shell pipelines.

### Used-car ranking with jq

```bash
blocket-pp-cli cars list --make Volvo --model XC70 --year-from 2012 --json --select docs.heading,docs.year,docs.mileage,docs.price.amount,docs.location | jq 'sort_by(.price.amount)[:10]'
```

Pipes structured JSON into jq for ranking — every wrapper requires a custom parser to do this.

### Daily delta on a saved search

```bash
blocket-pp-cli since --search xc70 --since 24h --json
```

Cron-friendly query for a 24-hour window; exits non-zero when there are matches so a wrapper script can react.

### Price-drop watcher

```bash
blocket-pp-cli watch run xc70 --notify-on price-drop --json
```

Diffs current prices against the prior snapshot; pair with a cron job or systemd timer to get notified only when the corpus actually moved.

### Description regex against the local store

```bash
blocket-pp-cli desc-grep --vertical car --pattern "\\bnyservad\\b" --json --select ad_id,heading,price.amount,canonical_url
```

Find listings whose description (not the title) mentions "nyservad" — the public search API will not match against descriptions.

## Auth Setup

No authentication required.

Run `blocket-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  blocket-pp-cli ads list --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Read-only** — do not use this CLI for create, update, delete, publish, comment, upvote, invite, order, send, or other mutating requests

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal — piped/agent consumers get pure JSON on stdout.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
blocket-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
blocket-pp-cli feedback --stdin < notes.txt
blocket-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.blocket-pp-cli/feedback.jsonl`. They are never POSTed unless `BLOCKET_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `BLOCKET_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename) |
| `webhook:<url>` | POST the output body to the URL (`application/json` or `application/x-ndjson` when `--compact`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled agent calls the same command every run with the same configuration - HeyGen's "Beacon" pattern.

```
blocket-pp-cli profile save briefing --json
blocket-pp-cli --profile briefing ads list
blocket-pp-cli profile list --json
blocket-pp-cli profile show briefing
blocket-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `blocket-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/commerce/blocket/cmd/blocket-pp-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add blocket-pp-mcp -- blocket-pp-mcp
   ```
3. Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which blocket-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   blocket-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `blocket-pp-cli <command> --help`.
