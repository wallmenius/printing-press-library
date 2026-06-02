// Copyright 2026 johan-wallmn. Licensed under Apache-2.0. See LICENSE.
// Hand-authored (library patch): honest-failure handler for the prisjakt_search
// MCP tool.
//
// PATCH: the generic makeAPIHandler fetches /search and returns the raw response
// body. Prisjakt's /search moved to Cloudflare-challenged client-side rendering,
// so that body is now a challenge/JS shell with no product data — the tool was
// returning an empty/null-shaped result that read as "no matches" rather than
// "search is unavailable". This dedicated handler runs the same parse path the
// CLI uses and, on the now-expected ErrSearchUnavailable, returns a clear,
// actionable message that points agents at PriceRunner search (which still
// works). When/if Prisjakt search becomes reachable again this handler will
// transparently start returning live results.

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mvanhorn/printing-press-library/library/commerce/se-prices/internal/source/prisjakt"
)

// handlePrisjaktSearch is the dedicated handler for the prisjakt_search tool.
func handlePrisjaktSearch(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	args := req.GetArguments()
	query, _ := args["query"].(string)
	if query == "" {
		return mcplib.NewToolResultError("query is required"), nil
	}

	c, err := newMCPClient()
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	html, err := c.Get("/search", map[string]string{"search": query})
	if err != nil {
		// A transport/HTTP error (e.g. the Cloudflare 403 challenge) is itself
		// a signal the search is unavailable; report it honestly rather than
		// as an empty result.
		return mcplib.NewToolResultError(fmt.Sprintf(
			"%s (underlying fetch error: %v)", prisjakt.ErrSearchUnavailable, err)), nil
	}

	result, perr := prisjakt.ParseSearch(html, query)
	if perr != nil {
		if errors.Is(perr, prisjakt.ErrSearchUnavailable) {
			return mcplib.NewToolResultError(fmt.Sprintf(
				"%s. Use the pricerunner_search tool with the same query instead.",
				prisjakt.ErrSearchUnavailable)), nil
		}
		return mcplib.NewToolResultError("parsing Prisjakt search: " + perr.Error()), nil
	}

	out, mErr := json.Marshal(result)
	if mErr != nil {
		return mcplib.NewToolResultError(mErr.Error()), nil
	}
	return mcplib.NewToolResultText(string(out)), nil
}
