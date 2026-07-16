package fetch

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// fakeHTTPClient is a stand-in for a real HTTP client so tests never reach the
// network. It records how many times Do was called, with which URL and headers,
// and returns the configured body/status.
type fakeHTTPClient struct {
	calls       int
	lastURL     string
	lastHeaders http.Header
	respBody    string
	respStatus  int
}

func (f *fakeHTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	f.calls++
	f.lastURL = req.URL.String()
	f.lastHeaders = req.Header.Clone()
	status := f.respStatus
	if status == 0 {
		status = 200
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(f.respBody)),
	}, nil
}

// TestFetch_RejectsMalformedURL asserts the core T1 behaviour: a malformed URL
// is rejected with a structured "invalid url" error AND no HTTP request is
// issued. The fake client's call count is the proof that no request went out.
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

// TestFetch_RejectsNonAbsoluteURL asserts that a URL missing a scheme or host
// is also rejected: a Web Reader needs an absolute target, not a bare path.
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

// TestFetch_AcceptsValidURL asserts that a well-formed absolute URL passes
// validation and returns a non-nil result without error. (Raw-HTML retrieval
// is asserted separately in TestFetch_ValidURLReturnsRawHTML.)
func TestFetch_AcceptsValidURL(t *testing.T) {
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

// TestFetch_ValidURLCallsClientOnce asserts the core retrieval behaviour: a
// valid URL is fetched exactly once through the injected client with the
// original URL. (Content shape — extracted markdown — is asserted in
// TestFetch_ReturnsExtractedMarkdown and the extract tests.)
func TestFetch_ValidURLCallsClientOnce(t *testing.T) {
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

// TestFetch_SendsRealisticBrowserHeaders asserts that the outgoing request
// carries headers that make it look like a real browser, so public sites with
// light anti-bot defenses stay fetchable (per ADR-0002).
func TestFetch_SendsRealisticBrowserHeaders(t *testing.T) {
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

// TestFetch_ReturnsExtractedMarkdown asserts that Fetch runs the extraction
// pipeline: given an article-type HTML response, it returns the extracted
// title and markdown body (not raw HTML), with boilerplate dropped.
func TestFetch_ReturnsExtractedMarkdown(t *testing.T) {
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
