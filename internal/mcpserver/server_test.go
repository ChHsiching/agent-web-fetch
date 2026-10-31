package mcpserver

import (
	"bytes"
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
	ct      string // Content-Type; defaults to text/html
	err     error  // if set, returned instead of a response
}

func (f *fakeFetchClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	f.calls++
	f.lastReq = req
	if f.err != nil {
		return nil, f.err
	}
	ct := f.ct
	if ct == "" {
		ct = "text/html; charset=utf-8"
	}
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": {ct}},
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
		t.Errorf("expected no fetch for malformed URL, got %d call(s)", client.calls)
	}
}

// TestHandleFetch_MaxMBRaisesSizeCap asserts max_mb flows through to the fetch
// params: a 2.5 MB body, over the 2 MB default, is admitted when the tool call
// raises the cap.
func TestHandleFetch_MaxMBRaisesSizeCap(t *testing.T) {
	const mb = 1 << 20
	body := string(bytes.Repeat([]byte("b"), 2*fetch.DefaultMaxMB*mb+512*1024))
	client := &fakeFetchClient{body: body, ct: "text/plain; charset=utf-8"}

	result, err := handleFetch(context.Background(), toolInput{
		URL:   "https://example.com/big.txt",
		MaxMB: intPtr(8),
	}, client)

	if err != nil {
		t.Fatalf("expected max_mb=8 to admit the 2.5 MB body, got %v", err)
	}
	if len(result.Content) != len(body) {
		t.Errorf("content = %d bytes, want the whole %d-byte body", len(result.Content), len(body))
	}
}

// TestHandleFetch_MaxMBAboveCeilingReturnsError asserts the adapter surfaces
// the ceiling error from the fetch layer: max_mb=9 fails before any request,
// with the ceiling named in the message.
func TestHandleFetch_MaxMBAboveCeilingReturnsError(t *testing.T) {
	client := &fakeFetchClient{}

	_, err := handleFetch(context.Background(), toolInput{
		URL:   "https://example.com/doc.txt",
		MaxMB: intPtr(9),
	}, client)

	if !fetch.IsInvalidMaxMBError(err) {
		t.Fatalf("expected InvalidMaxMBError for max_mb=9, got %T %v", err, err)
	}
	if !strings.Contains(err.Error(), "between 1 and 8") {
		t.Errorf("error should name the accepted range 1 to 8, got %q", err.Error())
	}
	if client.calls != 0 {
		t.Errorf("expected no fetch for invalid max_mb, got %d call(s)", client.calls)
	}
}

// TestHandleFetch_MaxMBNonPositiveReturnsError asserts an explicit zero or
// negative max_mb (distinguishable from absent only through the pointer) is
// rejected as an invalid parameter rather than silently becoming the default.
func TestHandleFetch_MaxMBNonPositiveReturnsError(t *testing.T) {
	for _, v := range []int{0, -2} {
		client := &fakeFetchClient{}

		_, err := handleFetch(context.Background(), toolInput{
			URL:   "https://example.com/doc.txt",
			MaxMB: intPtr(v),
		}, client)

		if !fetch.IsInvalidMaxMBError(err) {
			t.Fatalf("max_mb=%d: expected InvalidMaxMBError, got %T %v", v, err, err)
		}
		if client.calls != 0 {
			t.Errorf("max_mb=%d: expected no fetch, got %d call(s)", v, client.calls)
		}
	}
}

// intPtr is a small helper for building the optional max_mb tool input.
func intPtr(v int) *int { return &v }
