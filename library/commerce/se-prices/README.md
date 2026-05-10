# SE Prices CLI

**Combined CLI for Prisjakt and PriceRunner with cross-site arbitrage, price-drop digests, and EAN-keyed lookups no single-site tool offers.**

se-prices syncs both Swedish price-comparison aggregators into a local SQLite store and exposes commands neither site can: `arbitrage` finds gaps between the two indexes, `lowest` returns the cheapest in-stock offer across both in one call, `history-combo` merges price-snapshot time series, and `watchlist` cross-checks your bookmarks against both sources. Per-site commands stay first-class so agents can drill in when needed.

## Install

The recommended path installs both the `se-prices-pp-cli` binary and the `pp-se-prices` agent skill in one shot:

```bash
npx -y @mvanhorn/printing-press install se-prices
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press install se-prices --cli-only
```


### Without Node

The generated install path is category-agnostic until this CLI is published. If `npx` is not available before publish, install Node or use the category-specific Go fallback from the public-library entry after publish.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/se-prices-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-se-prices --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-se-prices --force
```

## Install for OpenClaw

Tell your OpenClaw agent (copy this):

```
Install the pp-se-prices skill from https://github.com/mvanhorn/printing-press-library/tree/main/cli-skills/pp-se-prices. The skill defines how its required CLI can be installed.
```

## Authentication

No authentication required. Both sites are read from their public SSR HTML state. Prisjakt's search route clears its Cloudflare challenge via Surf with a Chrome TLS fingerprint; everything else uses plain stdlib HTTP with a browser user agent.

## Quick Start

```bash
# Verify reachability for both sources and inspect the local store.
se-prices-pp-cli doctor


# Populate the local store from one category on both sites — fastest path to a useful local index.
se-prices-pp-cli sync --source both --category mobiltelefoner


# Cross-site cheapest in-stock offer in one call — the headline workflow.
se-prices-pp-cli lowest "iPhone 15 Pro Max" --in-stock


# Find products where one site undercuts the other by at least 10% — ranked by absolute SEK saved.
se-prices-pp-cli arbitrage --category mobiltelefoner --min-gap 10 --json


# Track an iPhone product across both sites; the next `watchlist check` reports whether either site offers it at your max.
se-prices-pp-cli watchlist add 14969878 --max 9990

```

## Unique Features

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

## Usage

Run `se-prices-pp-cli --help` for the full command reference and flag list.

## Commands

### pricerunner

PriceRunner.se — Klarna-owned price-comparison aggregator (read-only via SSR initial_payload)

- **`se-prices-pp-cli pricerunner category`** - Browse a PriceRunner category. ID and slug come from the /cl/ URL pattern.
- **`se-prices-pp-cli pricerunner deals`** - List PriceRunner curated deals.
- **`se-prices-pp-cli pricerunner product`** - Get PriceRunner product detail. ID path is the <group>-<numeric>/<category>/<slug>-priser segment (e.g., 1-3208336567/Mobiltelefoner/Apple-iPhone-15-Pro-Max-256GB-Natural-Titanium-priser).
- **`se-prices-pp-cli pricerunner search`** - Search PriceRunner products by name. Returns the search results from initial_payload.

### prisjakt

Prisjakt.nu — Sweden's leading price-comparison aggregator (read-only via SSR HTML state)

- **`se-prices-pp-cli prisjakt category`** - Browse a Prisjakt category page. Returns the productCollection from window.__REACT_QUERY_STATE__.
- **`se-prices-pp-cli prisjakt product`** - Get Prisjakt product detail by ID. Returns the full product object including offers, ratings, brand, category, and badges.
- **`se-prices-pp-cli prisjakt search`** - Search Prisjakt products by name. Returns the matching product list.


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
se-prices-pp-cli pricerunner search --query example-value

# JSON for scripting and agents
se-prices-pp-cli pricerunner search --query example-value --json

# Filter to specific fields
se-prices-pp-cli pricerunner search --query example-value --json --select id,name,status

# Dry run — show the request without sending
se-prices-pp-cli pricerunner search --query example-value --dry-run

# Agent mode — JSON + compact + no prompts in one flag
se-prices-pp-cli pricerunner search --query example-value --agent
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
npx skills add mvanhorn/printing-press-library/cli-skills/pp-se-prices -g
```

Then invoke `/pp-se-prices <query>` in Claude Code. The skill is the most efficient path — Claude Code drives the CLI directly without an MCP server in the middle.

<details>
<summary>Use as an MCP server in Claude Code (advanced)</summary>

If you'd rather register this CLI as an MCP server in Claude Code, install the MCP binary first:


Install the MCP binary from this CLI's published public-library entry or pre-built release.

Then register it:

```bash
claude mcp add se-prices se-prices-pp-mcp
```

</details>

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/se-prices-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


Install the MCP binary from this CLI's published public-library entry or pre-built release.

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "se-prices": {
      "command": "se-prices-pp-mcp"
    }
  }
}
```

</details>

## Health Check

```bash
se-prices-pp-cli doctor
```

Verifies configuration and connectivity to the API.

## Configuration

Config file: `~/.config/se-prices-pp-cli/config.toml`

Static request headers can be configured under `headers`; per-command header overrides take precedence.

## Troubleshooting
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific

- **Prisjakt search returns 403 or a Cloudflare "Just a moment" page.** — Run `se-prices-pp-cli doctor` to confirm Surf transport is active for Prisjakt search; the CLI auto-selects Surf for that route. If it still fails, set `SE_PRICES_TRANSPORT=surf` to force the Chrome TLS fingerprint.
- **`arbitrage` or `lowest` returns empty results.** — Run `se-prices-pp-cli sync --source both --category <slug>` first; the local store is empty until you sync at least one category from each site.
- **Cross-site match not found for a known product.** — Inspect the EAN with `prisjakt product <id> --select ean` and `pricerunner product <id> --select ean`; if either side is missing the EAN, fall back to `lowest "<product name>" --fuzzy-name`.
- **`pricerunner search` returns 403.** — PriceRunner blocks raw curl-style user agents on `/results`; the CLI sends a real browser UA by default. If you've set `SE_PRICES_USER_AGENT` to something terse, unset it.
- **Price snapshots are stale.** — Snapshots are written by `sync` and the watch commands; schedule `se-prices-pp-cli sync --source both --watched-only` on a cron to keep `drops`, `is-sale`, and `history-combo` accurate.

## HTTP Transport

This CLI uses Chrome-compatible HTTP transport for browser-facing endpoints. It does not require a resident browser process for normal API calls.

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
