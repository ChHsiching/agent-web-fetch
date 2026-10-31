package fetch

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// These tests lock down the size cap (ADR-0007): a response body larger than
// the active cap is rejected with a structured error stating the document's
// size and the limit, never truncated. The cap defaults to 2 MB; max_mb raises
// it for large-context consumers, hard-capped at 8. They exercise the same
// public Fetch seam with the injected fake client as the fetch / errors /
// passthrough tests. Over-limit bodies are built with bytes.Repeat so the
// tests stay fast despite multi-megabyte fixtures.

// TestFetch_MaxMBAboveCeilingRejected asserts that a max_mb above the hard
// ceiling is rejected with an error naming the ceiling, before any request is
// issued: the agent learns the boundary instead of guessing.
func TestFetch_MaxMBAboveCeilingRejected(t *testing.T) {
	resetCache()
	client := &fakeHTTPClient{respCT: "text/plain; charset=utf-8", respBody: "tiny"}

	_, err := Fetch(context.Background(), Params{URL: "https://example.com/doc.txt", MaxMB: 9}, client)

	if !IsInvalidMaxMBError(err) {
		t.Fatalf("expected InvalidMaxMBError for max_mb=9, got %T %v", err, err)
	}
	if !strings.Contains(err.Error(), "between 1 and 8") {
		t.Errorf("error should name the accepted range 1 to 8, got %q", err.Error())
	}
	if client.calls != 0 {
		t.Errorf("expected no HTTP request for invalid max_mb, got %d call(s)", client.calls)
	}
}

// TestFetch_MaxMBNegativeRejected asserts the invalid range below zero: a
// negative cap would be nonsense (unlimited is deliberately not offered, the
// ceiling is the escape hatch).
func TestFetch_MaxMBNegativeRejected(t *testing.T) {
	resetCache()
	client := &fakeHTTPClient{respCT: "text/plain; charset=utf-8", respBody: "tiny"}

	_, err := Fetch(context.Background(), Params{URL: "https://example.com/doc.txt", MaxMB: -1}, client)

	if !IsInvalidMaxMBError(err) {
		t.Fatalf("expected InvalidMaxMBError for max_mb=-1, got %T %v", err, err)
	}
	if client.calls != 0 {
		t.Errorf("expected no HTTP request for invalid max_mb, got %d call(s)", client.calls)
	}
}

// TestFetch_OversizedRawTextRejected asserts the core cap behaviour on the
// Passthrough path: a raw-text body over the default 2 MB cap is rejected with
// a SizeLimitExceededError carrying the limit, stating size and cap in the
// message. The body is served with no Content-Length (the fake leaves it
// unset), so the bounded read is what enforces the cap.
func TestFetch_OversizedRawTextRejected(t *testing.T) {
	resetCache()
	body := string(bytes.Repeat([]byte("a"), 2*megabyte+512))
	client := &fakeHTTPClient{respCT: "text/plain; charset=utf-8", respBody: body}

	result, err := Fetch(context.Background(), Params{URL: "https://example.com/big.txt"}, client)

	if result != nil || err == nil {
		t.Fatalf("expected rejection for oversized raw text, got result=%v err=%v", result, err)
	}
	var capErr *SizeLimitExceededError
	if !errors.As(err, &capErr) {
		t.Fatalf("expected SizeLimitExceededError, got %T %v", err, err)
	}
	if want := int64(DefaultMaxMB * megabyte); capErr.LimitBytes != want {
		t.Errorf("LimitBytes = %d, want the default cap %d", capErr.LimitBytes, want)
	}
	if capErr.SizeKnown {
		t.Error("SizeKnown should be false when no Content-Length declared the size")
	}
	if !strings.Contains(err.Error(), "2.0 MB") || !strings.Contains(err.Error(), "max_mb") {
		t.Errorf("error should state the limit and the max_mb escape hatch; got %q", err.Error())
	}
}

// TestFetch_OversizedHTMLRejected asserts the cap protects the Extraction path
// too: a giant HTML page is rejected with the same shaped error as a giant raw
// file, before extraction ever runs.
func TestFetch_OversizedHTMLRejected(t *testing.T) {
	resetCache()
	html := "<html><body><article><p>" + string(bytes.Repeat([]byte("paragraph "), 2*megabyte/10)) + "</p></article></body></html>"
	client := &fakeHTTPClient{respCT: "text/html; charset=utf-8", respBody: html}

	result, err := Fetch(context.Background(), Params{URL: "https://example.com/huge.html"}, client)

	if result != nil || err == nil {
		t.Fatalf("expected rejection for oversized HTML, got result=%v err=%v", result, err)
	}
	var capErr *SizeLimitExceededError
	if !errors.As(err, &capErr) {
		t.Fatalf("expected SizeLimitExceededError, got %T %v", err, err)
	}
	if capErr.LimitBytes != int64(DefaultMaxMB*megabyte) {
		t.Errorf("LimitBytes = %d, want the default cap", capErr.LimitBytes)
	}
	if !strings.Contains(err.Error(), "2.0 MB") {
		t.Errorf("error should state the active limit; got %q", err.Error())
	}
}

// recordingBody is an io.ReadCloser that counts Read calls, so a test can
// prove the body was never touched when Content-Length allows an early reject.
type recordingBody struct {
	reader io.Reader
	reads  int
}

func (b *recordingBody) Read(p []byte) (int, error) {
	b.reads++
	return b.reader.Read(p)
}

func (b *recordingBody) Close() error { return nil }

// contentLengthClient is a minimal fake (in the gbkClient tradition) whose
// response carries an explicit ContentLength and a recording body, proving
// whether the fetch path read from the body at all.
type contentLengthClient struct {
	contentLength int64
	body          *recordingBody
}

func (c *contentLengthClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode:    200,
		ContentLength: c.contentLength,
		Header:        http.Header{"Content-Type": {"text/plain; charset=utf-8"}},
		Body:          c.body,
	}, nil
}

// TestFetch_ContentLengthRejectsBeforeRead asserts the early-reject arm: when
// Content-Length declares an over-cap document, the error is returned before a
// single body byte is read, with the declared size in the error.
func TestFetch_ContentLengthRejectsBeforeRead(t *testing.T) {
	resetCache()
	const declared = int64(3*megabyte) + 12345
	client := &contentLengthClient{
		contentLength: declared,
		body:          &recordingBody{reader: bytes.NewReader(bytes.Repeat([]byte("x"), 3*megabyte+12345))},
	}

	result, err := Fetch(context.Background(), Params{URL: "https://example.com/declared-big.txt"}, client)

	if result != nil || err == nil {
		t.Fatalf("expected rejection from declared Content-Length, got result=%v err=%v", result, err)
	}
	var capErr *SizeLimitExceededError
	if !errors.As(err, &capErr) {
		t.Fatalf("expected SizeLimitExceededError, got %T %v", err, err)
	}
	if !capErr.SizeKnown || capErr.SizeBytes != declared {
		t.Errorf("SizeKnown=%v SizeBytes=%d, want the declared %d", capErr.SizeKnown, capErr.SizeBytes, declared)
	}
	if client.body.reads != 0 {
		t.Errorf("body must not be read when Content-Length allows early reject, got %d Read call(s)", client.body.reads)
	}
	if !strings.Contains(err.Error(), "3.0 MB") {
		t.Errorf("error should state the actual document size; got %q", err.Error())
	}
}

// TestFetch_MaxMBOptOutAdmitsLargeDocument asserts the opt-out: a document
// between the default cap and the ceiling (here 2.5 MB under max_mb=8) comes
// back whole and verbatim. The cap is raised, the body is not clipped.
func TestFetch_MaxMBOptOutAdmitsLargeDocument(t *testing.T) {
	resetCache()
	body := string(bytes.Repeat([]byte("z"), 2*megabyte+512*1024))
	client := &fakeHTTPClient{respCT: "text/plain; charset=utf-8", respBody: body}

	result, err := Fetch(context.Background(), Params{URL: "https://example.com/research-dump.txt", MaxMB: 8}, client)

	if err != nil {
		t.Fatalf("expected max_mb=8 to admit a 2.5 MB document, got %v", err)
	}
	if result.Content != body {
		t.Errorf("content must be the whole body verbatim; got %d bytes, want %d", len(result.Content), len(body))
	}
}

// TestFetch_MaxMBCacheNeverCrossServes asserts max_mb participates in the
// cache key: a document admitted under a raised cap (2.5 MB with max_mb=3) is
// not served to a default-cap fetch of the same URL, which must re-fetch and
// reject. The reverse order is covered by construction: errors are never
// cached.
func TestFetch_MaxMBCacheNeverCrossServes(t *testing.T) {
	resetCache()
	body := string(bytes.Repeat([]byte("q"), 2*megabyte+512*1024))
	client := &fakeHTTPClient{respCT: "text/plain; charset=utf-8", respBody: body}
	url := "https://example.com/shared-cache.txt"

	if _, err := Fetch(context.Background(), Params{URL: url, MaxMB: 3}, client); err != nil {
		t.Fatalf("raised-cap fetch: %v", err)
	}
	if _, err := Fetch(context.Background(), Params{URL: url, MaxMB: 3}, client); err != nil {
		t.Fatalf("raised-cap refetch: %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("second identical-cap fetch should hit cache, got %d call(s)", client.calls)
	}

	// Default cap on the same URL: must NOT receive the 3 MB-cap entry.
	_, err := Fetch(context.Background(), Params{URL: url}, client)
	if !IsSizeLimitExceededError(err) {
		t.Fatalf("default-cap fetch of the same URL must reject, got result from cache or wrong error: %v", err)
	}
	if client.calls != 2 {
		t.Errorf("default-cap fetch must be a separate cache entry and re-fetch, got %d call(s)", client.calls)
	}
}
