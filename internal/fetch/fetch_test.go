package fetch

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type fakeHTTPClient struct {
	calls       int
	lastURL     string
	lastHeaders http.Header
	respBody    string
	respStatus  int
	respCT      string // Content-Type; defaults to text/html
	err         error  // if set, returned instead of a response
	delay       time.Duration
}

func (f *fakeHTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	f.calls++
	f.lastURL = req.URL.String()
	f.lastHeaders = req.Header.Clone()
	if f.err != nil {
		return nil, f.err
	}
	// Honour cancellation: if the context deadline passes during a simulated
	// delay, report it like a real client would.
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	status := f.respStatus
	if status == 0 {
		status = 200
	}
	ct := f.respCT
	if ct == "" {
		ct = "text/html; charset=utf-8"
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {ct}},
		Body:       io.NopCloser(strings.NewReader(f.respBody)),
	}, nil
}

// resetCache swaps in a fresh defaultCache so a test that populates the cache
// (or relies on a miss) is isolated from tests that ran before it. The package
// shares one process-wide cache, so without this, test order can leak results.
func resetCache() {
	defaultCache = newCache(DefaultCacheTTL)
}

func TestFetch_RejectsMalformedURL(t *testing.T) {
	client := &fakeHTTPClient{}
	params := Params{URL: "not a valid url"}

	result, err := Fetch(context.Background(), params, client)

	if err == nil {
		t.Fatalf("expected an error for malformed URL, got nil (result=%v)", result)
	}
	if !IsInvalidURLError(err) {
		t.Errorf("expected an InvalidURLError, got %v", err)
	}
	if client.calls != 0 {
		t.Errorf("expected no HTTP request for malformed URL, got %d call(s)", client.calls)
	}
}

func TestFetch_RejectsNonAbsoluteURL(t *testing.T) {
	cases := []string{"example.com", "/some/path", "ftp://files.example.com/x"}
	for _, u := range cases {
		client := &fakeHTTPClient{}
		_, err := Fetch(context.Background(), Params{URL: u}, client)
		if !IsInvalidURLError(err) {
			t.Errorf("URL %q: expected InvalidURLError, got %v", u, err)
		}
		if client.calls != 0 {
			t.Errorf("URL %q: expected no request, got %d call(s)", u, client.calls)
		}
	}
}

func TestFetch_AcceptsValidURL(t *testing.T) {
	resetCache()
	client := &fakeHTTPClient{}
	params := Params{URL: "https://example.com/page"}

	result, err := Fetch(context.Background(), params, client)

	if err != nil {
		t.Fatalf("expected no error for valid URL %q, got %v", params.URL, err)
	}
	if result == nil {
		t.Fatal("expected a non-nil result for a valid URL")
	}
}

func TestFetch_ValidURLCallsClientOnce(t *testing.T) {
	resetCache()
	client := &fakeHTTPClient{respBody: "<html><body><article><h1>Hi</h1><p>Body text here.</p></article></body></html>"}
	params := Params{URL: "https://example.com/page"}

	_, err := Fetch(context.Background(), params, client)

	if err != nil {
		t.Fatalf("expected no error for valid URL, got %v", err)
	}
	if client.calls != 1 {
		t.Errorf("expected exactly one HTTP request, got %d", client.calls)
	}
	if client.lastURL != "https://example.com/page" {
		t.Errorf("client called with %q, want the original URL", client.lastURL)
	}
}

func TestFetch_SendsRealisticBrowserHeaders(t *testing.T) {
	resetCache()
	client := &fakeHTTPClient{respBody: "<html></html>"}
	params := Params{URL: "https://example.com/"}

	_, err := Fetch(context.Background(), params, client)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	ua := client.lastHeaders.Get("User-Agent")
	if ua == "" || !strings.Contains(ua, "Mozilla") {
		t.Errorf("User-Agent = %q, want a realistic browser UA containing \"Mozilla\"", ua)
	}
	if accept := client.lastHeaders.Get("Accept"); accept == "" {
		t.Errorf("Accept header missing; expected a non-empty browser-style Accept")
	}
	if acceptLang := client.lastHeaders.Get("Accept-Language"); acceptLang == "" {
		t.Errorf("Accept-Language header missing; expected a non-empty value")
	}
}

func TestFetch_ReturnsExtractedMarkdown(t *testing.T) {
	resetCache()
	const articleHTML = `<!DOCTYPE html>
<html><head><title>My Article — Site</title></head>
<body>
<nav><a href="/">Home</a></nav>
<article><h1>My Article</h1><p>The main substance of this article goes here, and it is long enough to clear the readability threshold that separates real articles from nav bleed-through.</p></article>
<footer>© 2026</footer>
</body></html>`
	client := &fakeHTTPClient{respBody: articleHTML}

	result, err := Fetch(context.Background(), Params{URL: "https://example.com/a"}, client)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(result.Title, "My Article") {
		t.Errorf("title = %q, want it to contain \"My Article\"", result.Title)
	}
	if !strings.Contains(result.Content, "main substance of this article") {
		t.Errorf("content should be extracted markdown with the article body; got:\n%s", result.Content)
	}
	if strings.Contains(result.Content, "<html") {
		t.Errorf("content should be markdown, not raw HTML; got:\n%s", result.Content)
	}
}

func TestFetch_RepeatedCallHitsCache(t *testing.T) {
	resetCache()
	client := &fakeHTTPClient{respBody: "<html><body><article><h1>C</h1><p>Body text that is long enough to be readable by the extractor.</p></article></body></html>"}
	params := Params{URL: "https://example.com/cached"}

	_, _ = Fetch(context.Background(), params, client)
	_, _ = Fetch(context.Background(), params, client)

	if client.calls != 1 {
		t.Errorf("expected the second Fetch to hit cache (1 HTTP call), got %d", client.calls)
	}
}

func TestFetch_NoCacheBypassesCache(t *testing.T) {
	resetCache()
	client := &fakeHTTPClient{respBody: "<html><body><article><h1>C</h1><p>Body text that is long enough to be readable by the extractor.</p></article></body></html>"}
	url := "https://example.com/nocache"

	_, _ = Fetch(context.Background(), Params{URL: url}, client)
	_, _ = Fetch(context.Background(), Params{URL: url, NoCache: true}, client)

	if client.calls != 2 {
		t.Errorf("expected 2 HTTP calls with NoCache bypass, got %d", client.calls)
	}
}
