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

// Result holds the output of a fetch. For now its shape is declared but Fetch
// does not yet populate it — retrieval and extraction arrive in later tickets.
type Result struct {
	// Content will hold the retrieved content once the fetch path is wired in
	// by a later ticket. Left intentionally empty here.
	Content string
}

// HTTPClient is the seam at which the network is injected. It executes a
// fully-formed request, so the caller controls headers and method; the client
// only provides transport. Tests substitute a fake so no test ever reaches the
// network. Fetch does not yet call it — the fetch path arrives in the next
// ticket — but the seam is declared here so the interface is stable.
type HTTPClient interface {
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
}

// Fetch retrieves the content at params.URL. At this stage it only validates
// the URL: a malformed or non-absolute URL returns an InvalidURLError without
// issuing any request. A Web Reader needs an absolute URL (a scheme and a
// host) — bare paths or fragments are not fetchable targets.
//
// Actual retrieval through the injected client is added by the next ticket.
func Fetch(ctx context.Context, params Params, client HTTPClient) (*Result, error) {
	parsed, err := url.Parse(params.URL)
	if err != nil || !isFetchableScheme(parsed.Scheme) || parsed.Host == "" {
		return nil, &InvalidURLError{URL: params.URL}
	}
	// Retrieval arrives in the next ticket. Validation is the whole job of
	// this slice; reaching this point means the URL is valid.
	return &Result{}, nil
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
