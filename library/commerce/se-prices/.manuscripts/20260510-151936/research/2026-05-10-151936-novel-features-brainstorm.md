# Novel-Features Brainstorm — se-prices

Audit trail of the Phase 1.5 Step 1.5c.5 subagent invocation. The Customer model
and Killed candidates do NOT go into the absorb manifest, but they are persisted
here for retro and dogfood debugging.

## Customer model

### Persona 1: Anders, the Black Week tech-spec hunter
- 34, IT consultant, Göteborg. Buys big-ticket electronics 1-2x/year.
- **Today:** Opens Prisjakt + PriceRunner side-by-side, manually pastes EANs,
  screenshots sparklines, keeps shortlists in Notes. Cannot answer "is this
  Elgiganten Black Week price actually below the 30-day median, or sticker-stuffing?"
- **Weekly ritual:** During sale season, nightly re-checks of 3-5 product IDs
  across both sites.
- **Frustration:** Manual cross-site EAN paste; can't spot fake sales.

### Persona 2: Maja, the Swedish-retail arbitrage hunter
- 29, dropshipping side-business, Malmö.
- **Today:** Two browser tabs + a spreadsheet. Squints at product names to
  match SKUs across sites because EANs aren't always shown.
- **Weekly ritual:** Sweeps 4-5 categories twice a week looking for ≥10%
  cross-site gaps.
- **Frustration:** Manual cross-site product matching is the bottleneck.

### Persona 3: Elin, the household price-drop watcher
- 41, parent, Stockholm. ~15-item wishlist of household goods.
- **Today:** Bookmarks Prisjakt product pages; PriceRunner has separate list.
- **Weekly ritual:** Sunday evening bookmark spot-checks.
- **Frustration:** Two parallel wishlists that never reconcile.

### Persona 4: Karl, the agent-driving developer
- 38, solo dev building a Telegram bot for a Swedish gadget-deals subreddit.
- **Today:** Two half-broken scrapers in a private repo, both fragile to SSR
  drift. Cannot answer "across both sites, which merchants have stock under
  threshold right now?"
- **Weekly ritual:** Patches scrapers when they break.
- **Frustration:** Maintaining two parsers for the same logical entities;
  needs unified MCP-shaped surface for agents.

## Candidates (pre-cut)

(See subagent output for full table; killed inline before Pass 3:
C13 mood — LLM dependency; C14 notify --watch — scope creep;
C15 open — wrapper; C16 explain — LLM dependency.)

## Killed in cut pass (Pass 3)

| Feature | Kill reason | Closest surviving sibling |
|---------|-------------|---------------------------|
| C1 `lowest` standalone | Subsumed by C7 stock-aware lowest (flag-less invocation = same shape) | Survivor #2 stock-aware lowest |
| C10 `merchant-trust <merchant>` | Fails weekly-use bar; per-site rating already absorbed; combo value thin | Survivor #4 catalogue-diff |
| C12 `best-merchant --category` | Cadence shaky for every persona; insight overlaps with C2 arbitrage repeated runs | Survivor #1 arbitrage |
| C13 `mood <id>` review sentiment | LLM dependency, no useful mechanical version | Already-absorbed `userReviewSummary` |
| C14 `notify --watch` push daemon | Scope creep (background process, cross-platform notif stack); reframed to one-shot `drops` | Survivor #5 drops |
| C15 `open <id>` browser opener | Wrapper-thin; trivial via `xdg-open $(... --json | jq -r .url)` | Already-absorbed `prisjakt product` |
| C16 `explain <id>` LLM explainer | LLM dependency, no mechanical core; users pipe to their own LLM | Survivor #8 is-sale |

## Survivors (8) — see absorb manifest for the rendered transcendence table.
