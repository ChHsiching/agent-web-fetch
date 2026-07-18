//go:build integration

package mcpserver

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestSmoke_RealNetworkFetch (run with: go test -tags=integration -run RealNetwork)
// verifies the full pipeline against a live public site: real http.Client ->
// Fetch -> extract -> tool result. Not run by default (depends on network).
func TestSmoke_RealNetworkFetch(t *testing.T) {
	deps := Deps{Client: &httpClientAdapter{client: &http.Client{Timeout: 30 * time.Second}}}
	server := newServer(deps)
	srvT, cliT := mcp.NewInMemoryTransports()
	go server.Run(context.Background(), srvT)

	cli := mcp.NewClient(&mcp.Implementation{Name: "smoke", Version: "0"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	sess, err := cli.Connect(ctx, cliT, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer sess.Close()

	res, err := sess.CallTool(ctx, &mcp.CallToolParams{
		Name:      "fetch",
		Arguments: map[string]any{"url": "https://example.com"},
	})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool error: %v", res.Content)
	}
	if len(res.Content) == 0 {
		t.Fatal("no content")
	}
	tc, _ := res.Content[0].(*mcp.TextContent)
	if tc == nil || !strings.Contains(strings.ToLower(tc.Text), "documentation examples") {
		t.Fatalf("expected example.com content, got: %v", tc)
	}
	t.Logf("fetched OK, preview: %.200s", tc.Text)
}
