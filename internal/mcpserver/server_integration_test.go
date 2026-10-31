package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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

func TestServer_ReportsVersionVar(t *testing.T) {
	connectVersion := func(t *testing.T) string {
		t.Helper()
		server := newServer(Deps{Client: &fakeFetchClient{}})
		serverTransport, clientTransport := mcp.NewInMemoryTransports()
		go server.Run(context.Background(), serverTransport)

		client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, nil)
		session, err := client.Connect(context.Background(), clientTransport, nil)
		if err != nil {
			t.Fatalf("client connect: %v", err)
		}
		defer session.Close()
		info := session.InitializeResult().ServerInfo
		if info == nil {
			t.Fatal("initialize result missing serverInfo")
		}
		return info.Version
	}

	if got := connectVersion(t); got != "dev" {
		t.Errorf("default version = %q, want %q", got, "dev")
	}

	old := Version
	Version = "v9.9.9-test"
	t.Cleanup(func() { Version = old })
	if got := connectVersion(t); got != "v9.9.9-test" {
		t.Errorf("injected version = %q, want %q", got, "v9.9.9-test")
	}
}

var errNetwork = &networkErr{}

type networkErr struct{}

func (e *networkErr) Error() string { return "simulated network error" }
