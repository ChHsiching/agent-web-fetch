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

// The body is served with no Content-Length (the fake leaves it unset), so
// the bounded read is what enforces the cap.
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

type recordingBody struct {
	reader io.Reader
	reads  int
}

func (b *recordingBody) Read(p []byte) (int, error) {
	b.reads++
	return b.reader.Read(p)
}

func (b *recordingBody) Close() error { return nil }

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

// The reverse order is covered by construction: errors are never cached.
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

	_, err := Fetch(context.Background(), Params{URL: url}, client)
	if !IsSizeLimitExceededError(err) {
		t.Fatalf("default-cap fetch of the same URL must reject, got result from cache or wrong error: %v", err)
	}
	if client.calls != 2 {
		t.Errorf("default-cap fetch must be a separate cache entry and re-fetch, got %d call(s)", client.calls)
	}
}
