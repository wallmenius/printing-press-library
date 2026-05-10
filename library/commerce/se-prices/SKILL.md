---
name: pp-se-prices
description: "Combined CLI for Prisjakt and PriceRunner with cross-site arbitrage, price-drop digests, and EAN-keyed lookups no... Trigger phrases: `compare prices on prisjakt and pricerunner`, `find arbitrage on swedish electronics`, `lowest swedish price for`, `watch this product for price drops in sweden`, `is this a real black week deal`, `use se-prices`, `run se-prices`."
author: "Johan Wallmén"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - se-prices-pp-cli
---

# SE Prices — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `se-prices-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer:
   ```bash
   npx -y @mvanhorn/printing-press install se-prices --cli-only
   ```
2. Verify: `se-prices-pp-cli --version`
3. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

If the `npx` install fails (no Node, offline, etc.), fall back to a direct Go install (requires Go 1.26.3 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/commerce/se-prices/cmd/se-prices-pp-cli@latest
```

If `--version` reports "command not found" after install, the install step did not put the binary on `$PATH`. Do not proceed with skill commands until verification succeeds.

se-prices syncs both Swedish price-comparison aggregators into a local SQLite store and exposes commands neither site can: `arbitrage` finds gaps between the two indexes, `lowest` returns the cheapest in-stock offer across both in one call, `history-combo` merges price-snapshot time series, and `watchlist` cross-checks your bookmarks against both sources. Per-site commands stay first-class so agents can drill in when needed.

## When to Use This CLI

Pick se-prices when an agent needs to compare Swedish retail prices across both Prisjakt and PriceRunner. It is the right tool when the user names a product, EAN/GTIN, or category and wants the lowest cross-site offer, an arbitrage gap, a merged price history, or a wishlist threshold check. Skip it for non-Swedish retail and for live realtime checks beyond the most recent sync — the analytical commands read the local snapshot store, not the live APIs.

## When Not to Use This CLI

Do not activate this CLI for requests that require creating, updating, deleting, publishing, commenting, upvoting, inviting, ordering, sending messages, booking, purchasing, or changing remote state. This printed CLI exposes read-only commands for inspection, export, sync, and analysis.

## Unique Capabilities

These capabilities aren't available in any other tool for this API.

### Cross-site combo
- **`arbitrage`** — Find products where one comparator's lowest offer beats the other by at least your gap threshold.

  _Reach for this when an agent needs to identify dropshipping or discount opportunities by comparing the two indexes; only this CLI has the cross-source lowest-per-site computation._

  ```bash
  se-prices-pp-cli arbitrage --category mobiltelefoner --min-gap 10 --agent --select product_name,prisjakt_price,pricerunner_price,gap_pct
  ```
- **`lowest`** — Cheapest current offer for a query across both sites, optionally filtered by stock and shipping cap.

  _Pick this when a user names a product and asks for the best price right now; one call returns the unified answer instead of an agent juggling two calls._

  ```bash
  se-prices-pp-cli lowest "iPhone 15 Pro Max" --in-stock --max-shipping 99 --agent
  ```
- **`ean`** — Resolve an EAN to both Prisjakt and PriceRunner product IDs and return the union of current offers.

  _Use when the user has an EAN/GTIN from a manufacturer page or barcode scan and wants to short-circuit name-based search._

  ```bash
  se-prices-pp-cli ean 0194253433927 --agent
  ```
- **`catalogue-diff`** — Products that appear in one site's category index but are missing from the other.

  _Reach for this to spot niche Swedish merchants or rare SKUs only one comparator indexes._

  ```bash
  se-prices-pp-cli catalogue-diff --category mobiltelefoner --only prisjakt --agent --select product_name,brand,prisjakt_lowest
  ```
- **`history-combo`** — Merged time series of price snapshots from both sources for one product, resolved via EAN.

  _Use when answering trend questions or producing graphs that need both sites' historical observations._

  ```bash
  se-prices-pp-cli history-combo --ean 0194253433927 --window 60 --csv
  ```

### Local watch + drops
- **`drops`** — Watched products whose latest snapshot is at least your percentage threshold below the snapshot at the start of the window.

  _Use as the daily or weekly digest when an agent is reporting deal flow on a user's wishlist._

  ```bash
  se-prices-pp-cli drops --since 7d --min-pct 10 --watched-only --agent
  ```
- **`watchlist`** — Track products by ID or EAN with a maximum price; check whether any tracked item is at or below threshold across either site.

  _Reach for this when an agent maintains a user's shopping list and surfaces threshold-met items without checking each site separately._

  ```bash
  se-prices-pp-cli watchlist check --agent --select product_name,best_offer.site,best_offer.price_sek
  ```
- **`is-sale`** — Compare the current price against the local 90-day median to flag sticker-stuffed sales.

  _Reach for this when a Black Week or summer-sale claim looks suspicious and you want a mechanical answer._

  ```bash
  se-prices-pp-cli is-sale 14969878 --window 90 --agent
  ```

## HTTP Transport

This CLI uses Chrome-compatible HTTP transport for browser-facing endpoints. It does not require a resident browser process for normal API calls.

## Command Reference

**pricerunner** — PriceRunner.se — Klarna-owned price-comparison aggregator (read-only via SSR initial_payload)

- `se-prices-pp-cli pricerunner category` — Browse a PriceRunner category. ID and slug come from the /cl/ URL pattern.
- `se-prices-pp-cli pricerunner deals` — List PriceRunner curated deals.
- `se-prices-pp-cli pricerunner product` — Get PriceRunner product detail. ID path is the <group>-<numeric>/<category>/<slug>-priser segment (e.g.,...
- `se-prices-pp-cli pricerunner search` — Search PriceRunner products by name. Returns the search results from initial_payload.

**prisjakt** — Prisjakt.nu — Sweden's leading price-comparison aggregator (read-only via SSR HTML state)

- `se-prices-pp-cli prisjakt category` — Browse a Prisjakt category page. Returns the productCollection from window.__REACT_QUERY_STATE__.
- `se-prices-pp-cli prisjakt product` — Get Prisjakt product detail by ID. Returns the full product object including offers, ratings, brand, category, and...
- `se-prices-pp-cli prisjakt search` — Search Prisjakt products by name. Returns the matching product list.


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
se-prices-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Recipes


### Tonight's lowest in-stock iPhone offer across both sites

```bash
se-prices-pp-cli lowest "iPhone 15 Pro Max" --in-stock --max-shipping 99 --agent --select best_offer.site,best_offer.merchant,best_offer.price_sek,best_offer.url
```

Cross-site min price filtered by stock and shipping cap; --select narrows the response so the agent doesn't pay for the full ProductList payload.

### Daily Black Week arbitrage sweep on TVs

```bash
se-prices-pp-cli arbitrage --category tv --min-gap 8 --min-sek 500 --agent --select product_name,prisjakt_price,pricerunner_price,gap_pct,gap_sek
```

Surfaces only ≥8% gaps worth at least 500 SEK; pair with a daily cron to catch Black Week and Mellandagsrea price moves.

### Wishlist check across both sites

```bash
se-prices-pp-cli watchlist check --agent --select product_name,target_max_sek,best_offer.site,best_offer.price_sek,below_threshold
```

Returns each tracked product's best current cross-site offer plus a boolean for threshold-met items.

### Is the current Black Week price actually a sale?

```bash
se-prices-pp-cli is-sale 14969878 --window 90d --agent
```

Compares current price against the 90-day median from the local snapshot store; flags sticker-stuffed sales as `actually_a_sale: false`.

### Unified 60-day price history for an EAN

```bash
se-prices-pp-cli history-combo --ean 0194253433927 --window 60d --csv
```

Merges price snapshots from both sources into one time series, CSV for spreadsheet plotting; works only if both sites have synced the EAN at least twice in the window.

## Auth Setup

No authentication required.

Run `se-prices-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  se-prices-pp-cli pricerunner search --query example-value --agent --select id,name,status
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
se-prices-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
se-prices-pp-cli feedback --stdin < notes.txt
se-prices-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.se-prices-pp-cli/feedback.jsonl`. They are never POSTed unless `SE_PRICES_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `SE_PRICES_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

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
se-prices-pp-cli profile save briefing --json
se-prices-pp-cli --profile briefing pricerunner search --query example-value
se-prices-pp-cli profile list --json
se-prices-pp-cli profile show briefing
se-prices-pp-cli profile delete briefing --yes
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

1. **Empty, `help`, or `--help`** → show `se-prices-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

Install the MCP binary from this CLI's published public-library entry or pre-built release, then register it:

```bash
claude mcp add se-prices-pp-mcp -- se-prices-pp-mcp
```

Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which se-prices-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   se-prices-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `se-prices-pp-cli <command> --help`.
