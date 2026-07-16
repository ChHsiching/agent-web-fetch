package mcpserver

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
)

// httpClientAdapter adapts the standard *http.Client to the fetch.HTTPClient
// interface (which uses Do(ctx, *http.Request)). It exists because Fetch takes
// its transport as an injected seam, and the real client is just net/http.
type httpClientAdapter struct {
	client *http.Client
}

func (a *httpClientAdapter) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return a.client.Do(req.WithContext(ctx))
}

// configureLogging routes the standard logger. When logPath is non-empty, logs
// are appended to that file; otherwise logging is discarded. Under no
// circumstance does it write to stdout — that is the MCP stdio transport, and
// any byte there corrupts the protocol (ADR-0005 row 8).
func configureLogging(logPath string) {
	if logPath == "" {
		log.SetOutput(io.Discard)
		return
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// Can't open the log file — stay silent rather than fall back to stdout.
		log.SetOutput(io.Discard)
		return
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags)
}
