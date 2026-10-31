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

// UnsupportedContentError is returned when the response is a binary or other
// non-text type (e.g. a PDF, image, audio/video, or generic binary) or has no
// Content-Type at all. HTML goes through Extraction and raw text documents go
// through Passthrough (ADR-0006); the remaining types are reported clearly
// rather than producing garbage.
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

// InvalidMaxMBError is returned when the max_mb parameter falls outside its
// allowed range: a positive integer no greater than the hard ceiling. The
// message names the accepted set of the surface that rejected the value so
// the agent learns the boundary instead of guessing (ADR-0007).
type InvalidMaxMBError struct {
	MaxMB   int
	Ceiling int
	// ZeroMeansDefault records which surface rejected the value. The Go API
	// treats 0 as "use the default cap" and never rejects it; the MCP tool
	// surface rejects an explicit 0 because an absent field, not 0, means
	// the default there. Error() names each surface's real accepted set so
	// the message stays truthful at both boundaries.
	ZeroMeansDefault bool
}

func (e *InvalidMaxMBError) Error() string {
	if e.ZeroMeansDefault {
		return fmt.Sprintf("invalid max_mb %d: must be between 1 and %d, or 0 for the default cap", e.MaxMB, e.Ceiling)
	}
	return fmt.Sprintf("invalid max_mb %d: must be between 1 and %d", e.MaxMB, e.Ceiling)
}

// IsInvalidMaxMBError reports whether err is an InvalidMaxMBError.
func IsInvalidMaxMBError(err error) bool {
	var target *InvalidMaxMBError
	return errors.As(err, &target)
}

// ValidateMaxMB is the single authority on which max_mb values are allowed,
// called by both fetch.Fetch and the MCP adapter so the rule and its message
// cannot drift between the layers (ADR-0007). zeroMeansDefault selects the
// surface's rule: the Go API accepts 0 as "use the default cap", while the
// MCP tool surface rejects an explicit 0 as a caller mistake and keeps only
// its own pointer-based absent-vs-explicit distinction.
func ValidateMaxMB(maxMB int, zeroMeansDefault bool) error {
	if maxMB < 0 || maxMB > CeilingMB || (maxMB == 0 && !zeroMeansDefault) {
		return &InvalidMaxMBError{MaxMB: maxMB, Ceiling: CeilingMB, ZeroMeansDefault: zeroMeansDefault}
	}
	return nil
}

// InvalidReturnFormatError is returned when the return_format parameter is
// not one of the accepted values ("markdown", "text"; empty means the
// markdown default). Like InvalidMaxMBError, a typo'd parameter is a caller
// mistake the reader surfaces with a typed error naming the accepted values
// rather than silently defaulting (spec #16).
type InvalidReturnFormatError struct {
	Value string
}

func (e *InvalidReturnFormatError) Error() string {
	return fmt.Sprintf("invalid return_format %q: must be \"markdown\" or \"text\"", e.Value)
}

// IsInvalidReturnFormatError reports whether err is an InvalidReturnFormatError.
func IsInvalidReturnFormatError(err error) bool {
	var target *InvalidReturnFormatError
	return errors.As(err, &target)
}

// SizeLimitExceededError is returned when a response body is larger than the
// active size cap. It states the document's size and the limit so the agent
// can decide whether to raise max_mb or narrow the fetch (ADR-0007). The body
// is never truncated: a truncated document would let the agent believe it has
// the whole file when it does not.
type SizeLimitExceededError struct {
	// SizeBytes is the document's size in bytes when SizeKnown is true (a
	// Content-Length declared it). Otherwise it is the lower bound the
	// bounded read reached before stopping (always LimitBytes + 1).
	SizeBytes int64
	// LimitBytes is the active cap in bytes (MaxMB * 1 MiB).
	LimitBytes int64
	// SizeKnown reports whether the exact size was known up front.
	SizeKnown bool
}

func (e *SizeLimitExceededError) Error() string {
	if e.SizeKnown {
		return fmt.Sprintf("size cap exceeded: document is %d bytes (%.1f MB), over the %.1f MB cap; raise max_mb (up to %d) or narrow the fetch. The body is never truncated",
			e.SizeBytes, mbFloat(e.SizeBytes), mbFloat(e.LimitBytes), CeilingMB)
	}
	return fmt.Sprintf("size cap exceeded: document exceeds the %.1f MB cap (stopped reading at %d bytes); raise max_mb (up to %d) or narrow the fetch. The body is never truncated",
		mbFloat(e.LimitBytes), e.LimitBytes+1, CeilingMB)
}

// IsSizeLimitExceededError reports whether err is a SizeLimitExceededError.
func IsSizeLimitExceededError(err error) bool {
	var target *SizeLimitExceededError
	return errors.As(err, &target)
}

// mbFloat converts a byte count to megabytes for error-message formatting.
func mbFloat(b int64) float64 {
	return float64(b) / float64(megabyte)
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
// (or XHTML, which the extractor also handles). The value is parsed with
// mediaType — the single content-type parser, shared with the raw-text
// classification — so every arm of the fetch switch dispatches on the same
// normalized media type. Matching stays substring-based on the parsed media
// type rather than exact equality so no historically accepted type is
// dropped: bare "application/xhtml" and "application/xhtml+xml" both match,
// exactly as under the pre-passthrough raw-header check.
func contentTypeIsHTML(ct string) bool {
	media := mediaType(ct)
	return strings.Contains(media, "text/html") || strings.Contains(media, "application/xhtml")
}
