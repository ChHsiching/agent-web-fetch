// Package fetch implements the Web Reader's core retrieval module.
//
// Fetch is the single deep module behind the tool: it takes a URL and returns
// model-friendly content. The HTTP client is injected so tests never touch the
// network. See CONTEXT.md for the Web Reader / Extraction vocabulary and
// docs/adr/ for the decisions this module honours.
package fetch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Params holds the inputs to a single fetch. Only URL is consumed by the
// URL-validation path; later tickets add fields as they implement their
// behaviour.
type Params struct {
	URL string
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

// Fetch retrieves the content at params.URL. It validates the URL first: a
// malformed, non-absolute, or non-http(s) URL returns an InvalidURLError
// without issuing any request. A Web Reader needs an absolute http(s) target.
//
// Once validated, the URL is fetched through the injected client with
// realistic browser headers, and the raw response body is returned.
// Error/timeout/panic handling arrives in a later ticket.
func Fetch(ctx context.Context, params Params, client HTTPClient) (*Result, error) {
	parsed, err := url.Parse(params.URL)
	if err != nil || !isFetchableScheme(parsed.Scheme) || parsed.Host == "" {
		return nil, &InvalidURLError{URL: params.URL}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, params.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header = browserHeaders.Clone()

	resp, err := client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	title, markdown, err := extract(string(body), params.URL)
	if err != nil {
		return nil, err
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

// InvalidURLError is the structured error returned for a malformed URL. It is
// the "invalid url" tool-error the agent receives — distinct from transport or
// HTTP errors so callers can tell failure modes apart.
type InvalidURLError struct {
	URL string
}

func (e *InvalidURLError) Error() string {
	return "invalid url: " + e.URL
}

// IsInvalidURLError reports whether err is an InvalidURLError. It is the typed
// check callers use rather than a brittle string comparison.
func IsInvalidURLError(err error) bool {
	var target *InvalidURLError
	return errors.As(err, &target)
}
