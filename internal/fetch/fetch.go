// Package fetch implements the Web Reader's core retrieval module.
//
// Fetch is the single deep module behind the tool: it takes a URL and returns
// model-friendly content. The HTTP client is injected so tests never touch the
// network. See CONTEXT.md for the Web Reader / Extraction vocabulary and
// docs/adr/ for the decisions this module honours.
package fetch

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultTimeout is applied when Params.Timeout is zero. Per the spec it is 30
// seconds — generous enough for a slow public page, short enough that a hung
// site never blocks the server.
const DefaultTimeout = 30 * time.Second

// Params holds the inputs to a single fetch. Timeout bounds the request (zero
// means use DefaultTimeout). ReturnFormat selects "markdown" (default, when
// empty) or "text". NoCache requests a fresh fetch that bypasses the cache.
type Params struct {
	URL          string
	Timeout      time.Duration
	ReturnFormat string
	NoCache      bool
}

// Result holds the output of a fetch. Title is the extracted article title
// (possibly empty for non-article pages). Content is the model-friendly
// markdown body — extracted by Readability, falling back to full-document
// markdown so the reader never returns empty.
type Result struct {
	Title   string
	Content string
}

// HTTPClient is the seam at which the network is injected. It executes a
// fully-formed request, so the caller controls headers and method; the client
// only provides transport. Tests substitute a fake so no test ever reaches the
// network.
type HTTPClient interface {
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
}

// browserHeaders are the realistic headers a modern browser sends, so public
// sites with light anti-bot defenses treat the request as ordinary traffic.
// Defined once here per ADR-0002 (anonymous + stealth: the disguise is part
// of the fetch path, absorbed into the binary rather than configured by users).
var browserHeaders = http.Header{
	"User-Agent":      {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"},
	"Accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
	"Accept-Language": {"en-US,en;q=0.9"},
}

// defaultCache is the process-wide in-memory result cache. Entries expire after
// DefaultCacheTTL; a process restart clears it. Honoured by Fetch unless
// params.NoCache is set.
var defaultCache = newCache(DefaultCacheTTL)

// Fetch retrieves the content at params.URL. It validates the URL first: a
// malformed, non-absolute, or non-http(s) URL returns an InvalidURLError
// without issuing any request.
//
// Validated results are served from the in-memory cache keyed by
// (url, return_format); params.NoCache forces a fresh fetch. The request is
// bounded by params.Timeout (default 30s). A non-2xx response returns an
// HTTPError; a non-HTML content type returns an UnsupportedContentError; a
// transport failure returns a FetchError. Any panic during the fetch or
// extract path is recovered and returned as a PanicError, so the process never
// crashes (ADR-0005).
func Fetch(ctx context.Context, params Params, client HTTPClient) (result *Result, err error) {
	// Recover any panic so the server process stays alive. A crash-restart is
	// itself a connection-failure risk (ADR-0005).
	defer func() {
		if r := recover(); r != nil {
			result, err = nil, &PanicError{Recovered: r}
		}
	}()

	parsed, err := url.Parse(params.URL)
	if err != nil || !isFetchableScheme(parsed.Scheme) || parsed.Host == "" {
		return nil, &InvalidURLError{URL: params.URL}
	}

	// Serve from cache unless the caller forces a fresh fetch. The miss handler
	// closes over ctx and client so the cache itself stays client-agnostic.
	return defaultCache.get(params, params.NoCache, func(p Params) (*Result, error) {
		return fetchUncached(ctx, p, client)
	})
}

// fetchUncached performs a single live fetch + extract, with no cache. It is
// the miss handler for Fetch's cache. All error types (HTTPError,
// UnsupportedContentError, FetchError, TimeoutError) are produced here.
func fetchUncached(ctx context.Context, params Params, client HTTPClient) (*Result, error) {
	timeout := params.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, params.URL, nil)
	if err != nil {
		return nil, &FetchError{Err: err}
	}
	req.Header = browserHeaders.Clone()

	resp, err := client.Do(ctx, req)
	if err != nil {
		return nil, classifyTransportErr(err, params.URL)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPError{StatusCode: resp.StatusCode, Reason: http.StatusText(resp.StatusCode)}
	}

	ct := resp.Header.Get("Content-Type")
	if !contentTypeIsHTML(ct) {
		return nil, &UnsupportedContentError{ContentType: ct}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, classifyTransportErr(err, params.URL)
	}

	title, markdown, err := extract(string(body), params.URL)
	if err != nil {
		return nil, &FetchError{Err: err}
	}
	return &Result{Title: title, Content: markdown}, nil
}

// isFetchableScheme reports whether a scheme is one the Web Reader will fetch.
// Only http and https are supported; ftp, file, mailto, etc. are rejected.
func isFetchableScheme(scheme string) bool {
	switch strings.ToLower(scheme) {
	case "http", "https":
		return true
	default:
		return false
	}
}
