package mcpserver

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

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
		fmt.Fprintf(os.Stderr, "agent-web-fetch: cannot open log file %s: %v\n", logPath, err)
		return
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags)
}
