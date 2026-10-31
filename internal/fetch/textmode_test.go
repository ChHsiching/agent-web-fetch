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

// TestFetch_TextFormatReturnsPlainText asserts the core text-mode behaviour:
// the same article page that markdown mode renders with headings, links, and
// emphasis markers comes back as plain text — the readability article's text
// content, no markdown syntax anywhere.
func TestFetch_TextFormatReturnsPlainText(t *testing.T) {
	resetCache()
	client := &fakeHTTPClient{respBody: sampleArticleHTML}

	result, err := Fetch(context.Background(), Params{URL: "https://example.com/text-article", ReturnFormat: "text"}, client)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(result.Content, "first paragraph of the article") {
		t.Errorf("text mode missing article substance; got:\n%s", result.Content)
	}
	if !strings.Contains(result.Title, "Real Article Title") {
		t.Errorf("text mode should keep the article title; got %q", result.Title)
	}
	for _, marker := range []string{"# ", "**", "](https://example.com/ref", "["} {
		if strings.Contains(result.Content, marker) {
			t.Errorf("text mode must not contain markdown marker %q; got:\n%s", marker, result.Content)
		}
	}
	if strings.Contains(result.Content, "<") {
		t.Errorf("text mode must not leak raw HTML tags; got:\n%s", result.Content)
	}
}

// TestFetch_TextFormatFallsBackToWholeDocument asserts the no-article case:
// a list page that markdown mode renders as bulleted links comes back as the
// whole document's plain text — the list items' text, no link or bullet
// markdown — honouring the same never-empty contract as markdown mode.
func TestFetch_TextFormatFallsBackToWholeDocument(t *testing.T) {
	resetCache()
	client := &fakeHTTPClient{respBody: sampleListPageHTML}

	result, err := Fetch(context.Background(), Params{URL: "https://example.com/text-list", ReturnFormat: "text"}, client)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if strings.TrimSpace(result.Content) == "" {
		t.Fatal("expected non-empty body via fallback, got empty string")
	}
	for _, item := range []string{"Item A", "Item B", "Item C"} {
		if !strings.Contains(result.Content, item) {
			t.Errorf("fallback text should contain %q; got:\n%s", item, result.Content)
		}
	}
	for _, marker := range []string{"[", "](/", "- "} {
		if strings.Contains(result.Content, marker) {
			t.Errorf("text fallback must not contain markdown marker %q; got:\n%s", marker, result.Content)
		}
	}
}

// TestFetch_TextFormatNeverEmpty asserts the never-empty contract in text
// mode: a script/style-only document yields the placeholder, not "".
func TestFetch_TextFormatNeverEmpty(t *testing.T) {
	resetCache()
	client := &fakeHTTPClient{respBody: sampleEmptyBodyHTML}

	result, err := Fetch(context.Background(), Params{URL: "https://example.com/text-blank", ReturnFormat: "text"}, client)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if strings.TrimSpace(result.Content) == "" {
		t.Fatal("expected non-empty output for script/style-only document, got empty")
	}
	if strings.Contains(result.Content, "track()") {
		t.Errorf("text mode should drop script; got:\n%s", result.Content)
	}
}

// TestFetch_TextFormatCaseInsensitiveSharesEntry asserts the case-insensitive
// contract end to end: "TEXT" fetches the same plain text as "text", and the
// normalized spelling shares one cache entry rather than fragmenting it into
// a second HTTP call.
func TestFetch_TextFormatCaseInsensitiveSharesEntry(t *testing.T) {
	resetCache()
	client := &fakeHTTPClient{respBody: sampleArticleHTML}
	url := "https://example.com/case-format"

	asLower, err := Fetch(context.Background(), Params{URL: url, ReturnFormat: "text"}, client)
	if err != nil {
		t.Fatalf("text: %v", err)
	}
	asUpper, err := Fetch(context.Background(), Params{URL: url, ReturnFormat: "TEXT"}, client)
	if err != nil {
		t.Fatalf("TEXT: %v", err)
	}

	if asLower.Content != asUpper.Content {
		t.Errorf("case variants should return identical content; text=%q TEXT=%q", asLower.Content, asUpper.Content)
	}
	if client.calls != 1 {
		t.Errorf("case variants should share one cache entry (1 HTTP call), got %d call(s)", client.calls)
	}
}

// TestFetch_TextAndMarkdownCacheNeverCrossServes asserts the two formats are
// separate cache entries that never serve each other: fetching the same URL
// as markdown then as text makes two HTTP calls with different content, and a
// repeat text fetch is served the text entry, not the markdown one.
func TestFetch_TextAndMarkdownCacheNeverCrossServes(t *testing.T) {
	resetCache()
	client := &fakeHTTPClient{respBody: sampleArticleHTML}
	url := "https://example.com/dual-format"

	asMarkdown, err := Fetch(context.Background(), Params{URL: url}, client)
	if err != nil {
		t.Fatalf("markdown fetch: %v", err)
	}
	asText, err := Fetch(context.Background(), Params{URL: url, ReturnFormat: "text"}, client)
	if err != nil {
		t.Fatalf("text fetch: %v", err)
	}

	if client.calls != 2 {
		t.Fatalf("the two formats must be separate cache entries (2 HTTP calls), got %d", client.calls)
	}
	if !strings.Contains(asMarkdown.Content, "](https://example.com/ref") {
		t.Errorf("markdown entry should carry link markdown; got:\n%s", asMarkdown.Content)
	}
	if strings.Contains(asText.Content, "](https://example.com/ref") {
		t.Errorf("text entry must not contain markdown; got:\n%s", asText.Content)
	}

	asTextAgain, err := Fetch(context.Background(), Params{URL: url, ReturnFormat: "text"}, client)
	if err != nil {
		t.Fatalf("text refetch: %v", err)
	}
	if client.calls != 2 {
		t.Errorf("repeat text fetch should hit the text cache entry, got %d call(s)", client.calls)
	}
	if asTextAgain.Content != asText.Content {
		t.Errorf("repeat text fetch must serve the text entry, not markdown; got:\n%s", asTextAgain.Content)
	}
}

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
