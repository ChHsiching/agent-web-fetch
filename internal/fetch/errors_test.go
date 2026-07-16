package fetch

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestFetch_TimeoutReturnsTimeoutError asserts that a fetch exceeding the
// timeout returns a structured TimeoutError and does not hang.
func TestFetch_TimeoutReturnsTimeoutError(t *testing.T) {
	client := &fakeHTTPClient{
		respBody: "<html><body><article><h1>Slow</h1><p>Body text that would arrive too late.</p></article></body></html>",
		delay:    200 * time.Millisecond,
	}
	params := Params{URL: "https://example.com/slow", Timeout: 20 * time.Millisecond}

	_, err := Fetch(context.Background(), params, client)

	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	if !IsTimeoutError(err) {
		t.Errorf("expected a TimeoutError, got %v", err)
	}
}

// TestFetch_Non2xxReturnsHTTPError asserts a non-2xx response surfaces as a
// structured HTTPError carrying the status code, with no retry.
func TestFetch_Non2xxReturnsHTTPError(t *testing.T) {
	client := &fakeHTTPClient{respStatus: 404, respBody: "Not Found"}
	params := Params{URL: "https://example.com/missing"}

	_, err := Fetch(context.Background(), params, client)

	if err == nil {
		t.Fatal("expected an error for 404, got nil")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected an *HTTPError, got %T: %v", err, err)
	}
	if httpErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", httpErr.StatusCode)
	}
	if httpErr.Reason == "" {
		t.Errorf("Reason should be populated (status text), got empty")
	}
	if client.calls != 1 {
		t.Errorf("expected exactly one request (no retry), got %d", client.calls)
	}
}

// TestFetch_TransportErrorReturnsFetchError asserts a transport failure surfaces
// as a structured FetchError, not a raw error.
func TestFetch_TransportErrorReturnsFetchError(t *testing.T) {
	client := &fakeHTTPClient{err: errors.New("connection refused")}
	params := Params{URL: "https://example.com/down"}

	_, err := Fetch(context.Background(), params, client)

	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var fe *FetchError
	if !errors.As(err, &fe) {
		t.Fatalf("expected a *FetchError, got %T: %v", err, err)
	}
	if !strings.Contains(fe.Error(), "connection refused") {
		t.Errorf("FetchError should wrap the transport cause; got %v", fe)
	}
}

// TestFetch_UnsupportedContentTypeReturnsError asserts that a non-HTML response
// (e.g. a PDF) returns a clear UnsupportedContentError.
func TestFetch_UnsupportedContentTypeReturnsError(t *testing.T) {
	client := &fakeHTTPClient{respCT: "application/pdf", respBody: "%PDF-1.4 ..."}
	params := Params{URL: "https://example.com/doc.pdf"}

	_, err := Fetch(context.Background(), params, client)

	if err == nil {
		t.Fatal("expected an error for PDF content type, got nil")
	}
	if !IsUnsupportedContentError(err) {
		t.Errorf("expected an UnsupportedContentError, got %v", err)
	}
}

// TestFetch_PanicIsRecovered asserts that a panic during the fetch/extract path
// is caught and returned as a tool-error, leaving the process alive.
func TestFetch_PanicIsRecovered(t *testing.T) {
	// A nil client panics on Do (nil pointer dereference). Fetch must recover.
	params := Params{URL: "https://example.com/panic"}

	var err error
	var result *Result
	// Run in a goroutine-free way: Fetch itself must defer recover so this
	// call returns normally instead of crashing the test binary.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Fetch did not recover; panic escaped: %v", r)
			}
		}()
		result, err = Fetch(context.Background(), params, nil)
	}()

	if err == nil {
		t.Fatal("expected an error from the recovered panic, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result on panic, got %v", result)
	}
}
