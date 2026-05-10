# se-prices Live Dogfood Acceptance Report

**Level:** Full Dogfood
**Matrix:** 78 tests run, 49 skipped (no positional / no JSON variant)
**Result:** 78/78 PASS — verdict PASS
**Run ID:** 20260510-151936
**Auth context:** none required (both sites are read-only public web)

## Tests covered
- doctor / version / agent-context / which (framework commands)
- prisjakt: search, product, category — help + happy path + JSON fidelity + error paths
- pricerunner: search, product, category, deals — help + happy path + JSON fidelity + error paths
- All 8 novel commands: lowest, ean, arbitrage, catalogue-diff, drops, watchlist (+ subcommands), history-combo, is-sale
- sep-sync: cross-site population
- sync (generated): pricerunner deals fetch (after fix)
- workflow archive/status

## Live verifications
- prisjakt category mobiltelefoner returned 8 trending products with name, brand, price, image, rating
- pricerunner search "iphone 15" returned 20 products with name, URL, lowest price, category, rating
- pricerunner product 1-3208336567/Mobiltelefoner/.../-priser returned 6 offers with merchant, price, stock
- lowest "iphone 15" returned PriceRunner @ 7,229 SEK as best cross-site offer
- sep-sync prisjakt mobiltelefoner saved 8 products + 8 snapshots
- sync pricerunner --max-pages 1: synced 1 record successfully

## Fixes applied during this dogfood
1. Removed 5 dead generator helpers (extractResponseData, printProvenance, replacePathParam, wantsHumanTable, wrapWithProvenance)
2. Fixed validate-narrative examples in research.json (--window 90d → 90, sync example → sep-sync)
3. Added input validation to ean / history-combo / is-sale / pricerunner-category (returns error 4 on bad input instead of empty success)
4. Added Examples sections to watchlist list/remove subcommands
5. Patched sync.go to use full PriceRunner URL for /deals (resource base_url override wasn't propagating in the generated sync helper)
6. Patched .printing-press.json with run_id (was overwritten earlier and lost the manifest fields)

## Known gaps documented for users
- Prisjakt category listings show `trendingProducts` (~8 items) instead of the full paginated catalog. The full list is loaded by client-side React Query against an internal-only GraphQL host (`graphql.expert.prisjakt.nu`) that does not resolve from outside Prisjakt's network. Accept the gap or use `pricerunner category` for fuller listings.
- Some `prisjakt` and `pricerunner` sub-views from the absorb manifest (offers, history, specs, reviews, brands, expert content, FAQ, mobile contracts, popular/trending, variants, related, badges, deal info, campaigns) are not shipped as standalone commands. They are accessible as fields on the `prisjakt product` / `pricerunner product` JSON output via `--select`. The 8 novel commands (the headline differentiators) are all shipped.

## Gate
**PASS** — proceed to Phase 5.5 (Polish), then promote to library, then Phase 6 next-steps menu.
