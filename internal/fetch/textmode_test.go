package fetch

import (
	"context"
	"strings"
	"testing"
)

// These tests lock down return_format: text (spec #16, ticket #18): an HTML
// page fetched with "text" comes back as plain text with no markdown syntax,
// a page with no article falls back to whole-document text (never empty),
// and unknown values are rejected with a typed error rather than silently
// defaulted. Raw-text passthrough is untouched (ADR-0006); its
// both-formats-identical guard lives in passthrough_test.go. Everything runs
// through the public Fetch seam with the injected fake client, like the
// fetch / sizecap tests.

// TestFetch_UnknownReturnFormatRejected asserts that an unknown non-empty
// return_format is a caller mistake the reader surfaces immediately, mirroring
// the max_mb validation philosophy (ADR-0007): reject with a typed error that
// names the accepted values, before any request is issued.
func TestFetch_UnknownReturnFormatRejected(t *testing.T) {
	for _, value := range []string{"html", "plain", "TEXT "}{
		resetCache()
		client := &fakeHTTPClient{respBody: sampleArticleHTML}

		_, err := Fetch(context.Background(), Params{URL: "https://example.com/fmt", ReturnFormat: value}, client)

		if !IsInvalidReturnFormatError(err) {
			t.Errorf("return_format %q: expected InvalidReturnFormatError, got %T %v", value, err, err)
		}
		if !strings.Contains(err.Error(), "markdown") || !strings.Contains(err.Error(), "text") {
			t.Errorf("return_format %q: error should name the accepted values; got %q", value, err.Error())
		}
		if client.calls != 0 {
			t.Errorf("return_format %q: expected no HTTP request, got %d call(s)", value, client.calls)
		}
	}
}
