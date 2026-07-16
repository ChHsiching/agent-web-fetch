// Package mcpserver exposes the fetch module as an MCP tool over stdio.
//
// The server is a thin adapter: it decodes the MCP tool call, maps it to a
// fetch.Params, calls fetch.Fetch, and wraps the result. All logging goes to
// a file (never stdout, which is the stdio transport — see ADR-0005 row 8).
package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ChHsiching/agent-web-fetch/internal/fetch"
)

// toolInput is the JSON schema for the fetch tool's parameters, inferred by
// the MCP SDK from struct tags. Mirrors the contract the spec defines. URL has
// no omitempty, so the SDK treats it as required.
type toolInput struct {
	URL          string `json:"url" jsonschema:"The URL to fetch"`
	Timeout      string `json:"timeout,omitempty" jsonschema:"Per-request timeout as a Go duration string (e.g. 30s). Defaults to 30s."`
	ReturnFormat string `json:"return_format,omitempty" jsonschema:"Output format: markdown or text. Defaults to markdown."`
	NoCache      bool   `json:"no_cache,omitempty" jsonschema:"Bypass the in-memory cache and force a fresh fetch."`
}

// Deps holds the injectable dependencies of the server, so tests can supply a
// fake fetch client instead of the real one.
type Deps struct {
	Client fetch.HTTPClient
}

// newServer builds an MCP server exposing a single "fetch" tool. The handler
// closes over deps so it can be tested with an injected client.
func newServer(deps Deps) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "agent-web-fetch", Version: "v0.1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "fetch",
		Description: "Fetch a URL and return its main content as model-friendly markdown. A free, uncapped replacement for paid web-reader tools.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in toolInput) (res *mcp.CallToolResult, _ any, _ error) {
		// Defense-in-depth: fetch.Fetch already recovers panics in the
		// fetch/extract path, but recover here too so the MCP server process
		// can never be killed by a panic anywhere in the handler (ADR-0005
		// row 6). A panic becomes an IsError tool result, not a crash.
		defer func() {
			if r := recover(); r != nil {
				res = &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("error: recovered panic: %v", r)}},
					IsError: true,
				}
			}
		}()
		result, err := handleFetch(ctx, in, deps.Client)
		if err != nil {
			// Surface the error to the agent as text content; the tool call
			// itself succeeds (no protocol error) so the model can read why.
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("error: %v", err)}},
				IsError: true,
			}, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result.Content}},
		}, nil, nil
	})
	return server
}

// handleFetch maps the tool input to fetch.Params and calls Fetch. Isolated
// from the MCP transport so it can be unit-tested.
func handleFetch(ctx context.Context, in toolInput, client fetch.HTTPClient) (*fetch.Result, error) {
	params := fetch.Params{
		URL:          in.URL,
		ReturnFormat: in.ReturnFormat,
		NoCache:      in.NoCache,
	}
	if in.Timeout != "" {
		d, err := time.ParseDuration(in.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout %q: %w", in.Timeout, err)
		}
		params.Timeout = d
	}
	return fetch.Fetch(ctx, params, client)
}

// Run starts the MCP server over stdio and blocks until the client disconnects.
// Logging is configured via the WEB_FETCH_LOG environment variable: when set
// to a file path, logs go there; otherwise the server is silent. Nothing is
// written to stdout (the stdio transport) under any circumstance.
//
// A closed stdin (client disconnect / EOF) is the normal shutdown path and
// returns nil rather than an error.
func Run(ctx context.Context) error {
	configureLogging(os.Getenv("WEB_FETCH_LOG"))
	client := &httpClientAdapter{client: &http.Client{}}
	server := newServer(Deps{Client: client})
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "EOF") {
			return nil
		}
		return err
	}
	return nil
}
