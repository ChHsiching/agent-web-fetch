package fetch

import (
	"context"
	"strings"
	"testing"
)

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

func TestFetch_UnknownReturnFormatRejected(t *testing.T) {
	for _, value := range []string{"html", "plain", "TEXT "} {
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
