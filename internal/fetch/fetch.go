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

// DefaultMaxMB is the response size cap applied when Params.MaxMB is zero, in
// megabytes. 2 MB fits what a 200k-token context can usefully consume; larger
// needs go through max_mb, not the default (ADR-0007).
const DefaultMaxMB = 2

// MaxMaxMB is the hard ceiling on Params.MaxMB. 8 MB covers the ~4 MB a
// 1M-token research context needs with headroom; beyond that the agent should
// narrow the fetch instead (ADR-0007).
const MaxMaxMB = 8

// megabyte is the bytes-per-MB used everywhere the cap is computed. Powers of
// two match how memory is usually quoted; the exact choice only matters for
// consistency between the early reject and the bounded read.
const megabyte = 1 << 20

// Params holds the inputs to a single fetch. Timeout bounds the request (zero
// means use DefaultTimeout). ReturnFormat selects "markdown" (default, when
// empty) or "text" (both case-insensitive; any other value is rejected with
// InvalidReturnFormatError). NoCache requests a fresh fetch that bypasses the
// cache. MaxMB sets the response size cap in MB (zero means DefaultMaxMB);
// values outside 1..MaxMaxMB are rejected with InvalidMaxMBError.
type Params struct {
	URL          string
	Timeout      time.Duration
	ReturnFormat string
	NoCache      bool
	MaxMB        int
}

// Result holds the output of a fetch. Title is the extracted article title
// (possibly empty for non-article pages; for raw-text documents it is the URL
// path basename). Content is the model-friendly body — markdown by default,
// or plain text when Params.ReturnFormat is "text" — extracted by Readability
// and falling back to the whole document so the reader never returns empty.
// For raw-text documents Content is the decoded body verbatim, identical for
// both formats (Passthrough, ADR-0006).
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
// without issuing any request. An out-of-range params.MaxMB likewise returns
// an InvalidMaxMBError before any lookup, as does an unknown non-empty
// params.ReturnFormat (InvalidReturnFormatError).
//
// Validated results are served from the in-memory cache keyed by
// (url, return_format, max_mb); params.NoCache forces a fresh fetch. The
// request is bounded by params.Timeout (default 30s). A non-2xx response
// returns an HTTPError. A body over the size cap (params.MaxMB MB, default 2,
// ceiling 8) returns a SizeLimitExceededError: the body is rejected, never
// truncated (ADR-0007). An HTML response is extracted as today; a raw-text
// response (ADR-0006) is returned verbatim; any other content type returns an
// UnsupportedContentError. A transport failure returns a FetchError. Any panic
// during the fetch or extract path is recovered and returned as a PanicError,
// so the process never crashes (ADR-0005).
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

	// Validate max_mb before any cache lookup or request: an out-of-range cap
	// is a caller bug, and the agent should learn the ceiling immediately.
	if params.MaxMB < 0 || params.MaxMB > MaxMaxMB {
		return nil, &InvalidMaxMBError{MaxMB: params.MaxMB, Ceiling: MaxMaxMB}
	}

	// Validate and normalize return_format the same way: an unknown value is a
	// caller bug to reject with a typed error, never silently defaulted.
	// Accepted case-insensitively ("TEXT" == "text") like a sane API, and the
	// normalized form replaces the input so case variants share one cache
	// entry instead of fragmenting the key.
	switch format := strings.ToLower(params.ReturnFormat); format {
	case "", formatMarkdown, formatText:
		params.ReturnFormat = format
	default:
		return nil, &InvalidReturnFormatError{Value: params.ReturnFormat}
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

	// Enforce the size cap (ADR-0007). A declared Content-Length allows an
	// early reject before a single body byte is buffered; chunked or lying
	// responses are caught by the bounded read below. Both accepted arms
	// (Extraction and Passthrough) read through this cap, so neither a giant
	// page nor a giant raw file can flood the context.
	limitBytes := limitFor(params.MaxMB)
	if resp.ContentLength > limitBytes {
		return nil, &SizeLimitExceededError{
			SizeBytes:  resp.ContentLength,
			LimitBytes: limitBytes,
			SizeKnown:  true,
		}
	}

	// Classify the response three ways (ADR-0006): HTML → extraction as
	// today; raw text → passthrough verbatim; anything else → rejected.
	ct := resp.Header.Get("Content-Type")
	switch {
	case contentTypeIsHTML(ct):
		body, err := readBody(resp.Body, params.URL, limitBytes)
		if err != nil {
			return nil, err
		}
		title, content, err := extract(string(body), params.URL, params.ReturnFormat)
		if err != nil {
			return nil, &FetchError{Err: err}
		}
		return &Result{Title: title, Content: content}, nil
	case isRawTextContentType(ct):
		body, err := readBody(resp.Body, params.URL, limitBytes)
		if err != nil {
			return nil, err
		}
		return passthroughResult(body, ct, params.URL)
	default:
		return nil, &UnsupportedContentError{ContentType: ct}
	}
}

// limitFor returns the size cap in bytes for a max_mb value, applying the
// default when it is zero. Callers must have validated the value already
// (Fetch rejects out-of-range max_mb before the cache).
func limitFor(maxMB int) int64 {
	if maxMB <= 0 {
		maxMB = DefaultMaxMB
	}
	return int64(maxMB) * megabyte
}

// readBody reads at most limit+1 bytes of the response body, translating a
// read failure into the structured transport error for url (timeout vs fetch
// failure). Reading one byte past the limit proves the document is over the
// cap without buffering more of it (ADR-0007). It is the shared read step of
// both accepted arms in fetchUncached, Extraction and Passthrough, while
// rejected content types return above before any body is read (ADR-0006).
func readBody(body io.Reader, url string, limitBytes int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(body, limitBytes+1))
	if err != nil {
		return nil, classifyTransportErr(err, url)
	}
	if int64(len(b)) > limitBytes {
		return nil, &SizeLimitExceededError{
			SizeBytes:  int64(len(b)),
			LimitBytes: limitBytes,
			SizeKnown:  false,
		}
	}
	return b, nil
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
