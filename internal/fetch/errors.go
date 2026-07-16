package fetch

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

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

// FetchError wraps a transport-level failure (DNS, connection refused, TLS,
// etc.) as a structured tool-error so callers can tell it apart from HTTP
// status errors, content-type errors, and timeouts.
type FetchError struct {
	Err error
}

func (e *FetchError) Error() string {
	return fmt.Sprintf("fetch failed: %v", e.Err)
}

func (e *FetchError) Unwrap() error { return e.Err }

// HTTPError is returned when the server responds with a non-2xx status code.
// It carries the status code and a short reason so the agent knows what went
// wrong without seeing the raw response.
type HTTPError struct {
	StatusCode int
	Reason     string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("http %d: %s", e.StatusCode, e.Reason)
}

// TimeoutError is returned when the fetch exceeds its deadline. Distinct from
// FetchError so the agent can tell "slow/unreachable" from "rejected".
type TimeoutError struct {
	URL string
}

func (e *TimeoutError) Error() string {
	return "fetch timed out: " + e.URL
}

// UnsupportedContentError is returned when the response is not an HTML
// document (e.g. a PDF, image, or binary). The reader only extracts HTML;
// other types are reported clearly rather than producing garbage.
type UnsupportedContentError struct {
	ContentType string
}

func (e *UnsupportedContentError) Error() string {
	return "unsupported content type: " + e.ContentType
}

// PanicError is returned when a panic is recovered during the fetch/extract
// path. Surfacing it as an error keeps the server process alive (ADR-0005).
type PanicError struct {
	Recovered any
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("recovered panic: %v", e.Recovered)
}

// IsTimeoutError reports whether err is a TimeoutError or wraps a context
// deadline-exceeded. Either form is treated as a timeout for the agent.
func IsTimeoutError(err error) bool {
	var te *TimeoutError
	if errors.As(err, &te) {
		return true
	}
	return errors.Is(err, context.DeadlineExceeded)
}

// IsUnsupportedContentError reports whether err is an UnsupportedContentError.
func IsUnsupportedContentError(err error) bool {
	var uce *UnsupportedContentError
	return errors.As(err, &uce)
}

// classifyTransportErr turns a transport-level error into the structured error
// the reader returns: timeouts become TimeoutError, everything else becomes a
// wrapped FetchError. Used at both the request and body-read failure points.
func classifyTransportErr(err error, url string) error {
	if IsTimeoutError(err) {
		return &TimeoutError{URL: url}
	}
	return &FetchError{Err: err}
}

// contentTypeIsHTML reports whether a Content-Type header value denotes HTML
// (or XHTML, which the extractor also handles). Header values are matched
// case-insensitively; a charset suffix is tolerated.
func contentTypeIsHTML(ct string) bool {
	lower := strings.ToLower(ct)
	return strings.Contains(lower, "text/html") || strings.Contains(lower, "application/xhtml")
}
