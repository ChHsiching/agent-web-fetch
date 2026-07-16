package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestServer_ToolRegisteredAndCallable runs the MCP server in-process over an
// in-memory transport and exercises it with a real MCP client. This verifies
// the protocol layer end-to-end (AC 1, 2) without touching the network: a fake
// HTTP client stands in for the real one.
func TestServer_ToolRegisteredAndCallable(t *testing.T) {
	fake := &fakeFetchClient{body: "<html><body><article><h1>Real</h1><p>The article body is long enough to clear the readability extraction threshold used by the fetch module.</p></article></body></html>"}

	server := newServer(Deps{Client: fake})
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Run(context.Background(), serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, nil)
	ctx := context.Background()
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close()

	// AC 2: the fetch tool is registered with the expected name.
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	var found *mcp.Tool
	for _, tl := range tools.Tools {
		if tl.Name == "fetch" {
			found = tl
			break
		}
	}
	if found == nil {
		t.Fatalf("fetch tool not registered; got tools: %v", tools.Tools)
	}

	// AC 1 + 3: call the tool end-to-end and get extracted content back.
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "fetch",
		Arguments: map[string]any{
			"url": "https://example.com/integration",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned an error result: %v", result.Content)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content in tool result, got none")
	}
	text, _ := result.Content[0].(*mcp.TextContent)
	if text == nil || !strings.Contains(text.Text, "article body is long enough") {
		got := ""
		if text != nil {
			got = text.Text
		}
		t.Errorf("expected extracted markdown in result, got %q", got)
	}
}

// TestServer_ErrorsSurfacedAsToolResult asserts that a failed fetch comes back
// as a tool result with IsError=true, not as a protocol error — so the model
// can read why the fetch failed rather than the call being rejected.
func TestServer_ErrorsSurfacedAsToolResult(t *testing.T) {
	fake := &fakeFetchClient{err: errNetwork}

	server := newServer(Deps{Client: fake})
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	go server.Run(context.Background(), serverTransport)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, nil)
	ctx := context.Background()
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close()

	// A fetch failure must come back as a tool result (IsError), not a
	// protocol error — so the model can read why it failed.
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "fetch",
		Arguments: map[string]any{"url": "https://example.com/fail"},
	})
	if err != nil {
		t.Fatalf("CallTool should not return a protocol error, got: %v", err)
	}
	if !result.IsError {
		t.Errorf("expected IsError=true for a failed fetch, got false (content: %v)", result.Content)
	}
}

// errNetwork is a stand-in transport error for the failure-path test.
var errNetwork = &networkErr{}

type networkErr struct{}

func (e *networkErr) Error() string { return "simulated network error" }
