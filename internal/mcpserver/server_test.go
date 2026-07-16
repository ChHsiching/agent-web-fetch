package mcpserver

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ChHsiching/agent-web-fetch/internal/fetch"
)

var _ fetch.HTTPClient = (*fakeFetchClient)(nil)

// fakeFetchClient records calls and returns canned HTML, satisfying
// fetch.HTTPClient so handleFetch can be tested without the network.
type fakeFetchClient struct {
	calls   int
	lastReq *http.Request
	body    string
	err     error // if set, returned instead of a response
}

func (f *fakeFetchClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	f.calls++
	f.lastReq = req
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": {"text/html; charset=utf-8"}},
		Body:       io.NopCloser(strings.NewReader(f.body)),
	}, nil
}

// TestHandleFetch_MapsParamsAndReturnsContent asserts handleFetch maps the
// tool input to fetch.Params (URL, format, timeout) and returns the extracted
// content.
func TestHandleFetch_MapsParamsAndReturnsContent(t *testing.T) {
	client := &fakeFetchClient{body: "<html><body><article><h1>Hi</h1><p>Body text long enough to pass the readability threshold for extraction.</p></article></body></html>"}

	result, err := handleFetch(context.Background(), toolInput{
		URL:          "https://example.com/mcp-maps",
		ReturnFormat: "markdown",
		Timeout:      "15s",
	}, client)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client.calls != 1 {
		t.Errorf("expected 1 fetch, got %d", client.calls)
	}
	if client.lastReq == nil || client.lastReq.URL.String() != "https://example.com/mcp-maps" {
		t.Errorf("request URL mismatch")
	}
	if !strings.Contains(result.Content, "Body text") {
		t.Errorf("expected extracted body; got %q", result.Content)
	}
}

// TestHandleFetch_InvalidTimeoutReturnsError asserts a bad timeout string is
// rejected cleanly rather than panicking.
func TestHandleFetch_InvalidTimeoutReturnsError(t *testing.T) {
	client := &fakeFetchClient{}

	_, err := handleFetch(context.Background(), toolInput{
		URL:     "https://example.com/mcp-badtimeout",
		Timeout: "not-a-duration",
	}, client)

	if err == nil {
		t.Fatal("expected an error for invalid timeout, got nil")
	}
	if client.calls != 0 {
		t.Errorf("expected no fetch for invalid timeout, got %d", client.calls)
	}
}

// TestHandleFetch_MalformedURLReturnsError asserts validation errors propagate.
func TestHandleFetch_MalformedURLReturnsError(t *testing.T) {
	client := &fakeFetchClient{}

	_, err := handleFetch(context.Background(), toolInput{URL: "not a url"}, client)

	if err == nil {
		t.Fatal("expected an error for malformed URL, got nil")
	}
	if client.calls != 0 {
		t.Errorf("expected no fetch for malformed URL, got %d", client.calls)
	}
}
