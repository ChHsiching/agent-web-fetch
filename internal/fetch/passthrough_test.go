package fetch

import (
	"context"
	"strings"
	"testing"
	"time"
)

// These tests lock down the raw-text passthrough path (ADR-0006, CONTEXT.md
// "Passthrough"): responses that are already text are decoded to UTF-8 and
// returned verbatim — no extraction, no code fences, no prefixes. They
// exercise the same public Fetch seam with the injected fake client as the
// fetch / errors / encoding tests. Expected decodings follow the documented
// behaviour of golang.org/x/net/html/charset (header charset → BOM → sniff →
// windows-1252 fallback).

// TestFetch_PlainTextPassthroughVerbatim asserts the issue-#11 repro: a
// text/plain response (how raw.githubusercontent.com serves .md/.txt/.json
// files) returns its full body verbatim, with the title derived from the URL
// path basename.
func TestFetch_PlainTextPassthroughVerbatim(t *testing.T) {
	resetCache()
	const body = "# agent-web-fetch\n\nA web reader for agents.\n\n```go\nfmt.Println(\"hi\")\n```\n\n- one\n- two\n"
	client := &fakeHTTPClient{respCT: "text/plain; charset=utf-8", respBody: body}

	result, err := Fetch(context.Background(), Params{URL: "https://raw.githubusercontent.com/golang/go/master/README.md"}, client)

	if err != nil {
		t.Fatalf("expected no error for text/plain, got %v", err)
	}
	if result.Content != body {
		t.Errorf("content must match the body verbatim (no fences, no prefix); got:\n%q\nwant:\n%q", result.Content, body)
	}
	if result.Title != "README.md" {
		t.Errorf("title = %q, want the URL path basename %q", result.Title, "README.md")
	}
	if strings.Contains(result.Content, "```markdown") {
		t.Errorf("content must not be wrapped in a code fence; got:\n%s", result.Content)
	}
}

// TestFetch_RawTextTypesPassThroughVerbatim asserts the full ADR-0006 accepted
// type list: every raw-text media type returns the body verbatim. Structured
// bodies (JSON/XML/YAML/JS) are content, not accidents — a JSON API response
// must reach the agent exactly as served. Header case and parameters must not
// affect classification.
func TestFetch_RawTextTypesPassThroughVerbatim(t *testing.T) {
	cases := []struct {
		name string
		ct   string
		body string
	}{
		{"json", "application/json", `{"name": "agent-web-fetch", "ok": true}` + "\n"},
		{"json with charset", "application/json; charset=utf-8", "[1, 2, 3]\n"},
		{"vendor +json", "application/vnd.api+json", `{"data": {"type": "posts"}}`},
		{"xml", "application/xml", "<root><item>value</item></root>"},
		{"text/xml", "text/xml; charset=utf-8", "<?xml version=\"1.0\"?><feed/>"},
		{"atom +xml", "application/atom+xml; charset=utf-8", "<feed xmlns=\"http://www.w3.org/2005/Atom\"><title>t</title></feed>"},
		{"yaml", "application/yaml", "key: value\nlist:\n  - a\n  - b\n"},
		{"x-yaml", "application/x-yaml", "key: value\n"},
		{"vendor +yaml", "application/vnd.config+yaml", "replicas: 2\n"},
		{"javascript", "application/javascript; charset=utf-8", "function hello() { return \"hi\"; }\n"},
		{"csv", "text/csv", "a,b,c\n1,2,3\n"},
		{"markdown", "text/markdown; charset=utf-8", "# Heading\n\nBody.\n"},
		{"uppercase header", "TEXT/PLAIN; CharSet=UTF-8", "case insensitive\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetCache()
			client := &fakeHTTPClient{respCT: tc.ct, respBody: tc.body}

			result, err := Fetch(context.Background(), Params{URL: "https://example.com/files/doc"}, client)

			if err != nil {
				t.Fatalf("content type %q: expected passthrough, got %v", tc.ct, err)
			}
			if result.Content != tc.body {
				t.Errorf("content type %q: content must match the body verbatim; got:\n%q\nwant:\n%q", tc.ct, result.Content, tc.body)
			}
		})
	}
}

// TestFetch_RawTextTitleIsURLBasename asserts the passthrough title rule: the
// URL path basename, falling back to the host when the path has none.
func TestFetch_RawTextTitleIsURLBasename(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"https://example.com/dir/file.txt", "file.txt"},
		{"https://example.com/config.json", "config.json"},
		{"https://example.com/dir/", "dir"},
		{"https://example.com", "example.com"},
	}
	for _, tc := range cases {
		t.Run(tc.url, func(t *testing.T) {
			resetCache()
			client := &fakeHTTPClient{respCT: "text/plain", respBody: "body\n"}

			result, err := Fetch(context.Background(), Params{URL: tc.url}, client)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if result.Title != tc.want {
				t.Errorf("url %q: title = %q, want %q", tc.url, result.Title, tc.want)
			}
		})
	}
}

// TestFetch_BinaryTypesStillRejected asserts the rejection side of the
// classification: PDFs, images, media, and generic binaries keep failing with
// the existing UnsupportedContentError, whose message names the type. (The PDF
// case is also covered by errors_test.go; this table pins the boundary of the
// newly accepted raw-text list.)
func TestFetch_BinaryTypesStillRejected(t *testing.T) {
	cases := []string{
		"application/pdf",
		"image/png",
		"image/vnd.microsoft.icon",
		"audio/mpeg",
		"video/mp4",
		"application/octet-stream",
		"application/zip",
		"font/woff2",
	}
	for _, ct := range cases {
		t.Run(ct, func(t *testing.T) {
			resetCache()
			client := &fakeHTTPClient{respCT: ct, respBody: "\x00\x01binary bytes"}

			result, err := Fetch(context.Background(), Params{URL: "https://example.com/file.bin"}, client)

			if result != nil || err == nil {
				t.Fatalf("content type %q: expected rejection, got result=%v err=%v", ct, result, err)
			}
			if !IsUnsupportedContentError(err) {
				t.Fatalf("content type %q: expected UnsupportedContentError, got %T %v", ct, err, err)
			}
			if !strings.Contains(err.Error(), ct) {
				t.Errorf("content type %q: error should name the type; got %q", ct, err.Error())
			}
		})
	}
}

// TestFetch_EmptyContentTypeStillRejected asserts an empty Content-Type
// header stays rejected — an ambiguous response fails loudly instead of being
// guessed at. (gbkClient is used because fakeHTTPClient substitutes a default
// text/html Content-Type when the value is empty.)
func TestFetch_EmptyContentTypeStillRejected(t *testing.T) {
	resetCache()
	client := &gbkClient{body: []byte("some bytes"), ct: ""}

	result, err := Fetch(context.Background(), Params{URL: "https://example.com/unknown"}, client)

	if result != nil || err == nil {
		t.Fatalf("expected rejection for empty Content-Type, got result=%v err=%v", result, err)
	}
	if !IsUnsupportedContentError(err) {
		t.Errorf("expected UnsupportedContentError for empty Content-Type, got %T %v", err, err)
	}
}

// TestFetch_PlainTextGBKDeclaredCharsetDecodes asserts that a GBK-encoded
// plain-text file whose Content-Type declares charset=gbk decodes to correct
// UTF-8 — the raw-text twin of the GBK-HTML regression tests in
// encoding_test.go. The declared charset parameter must be honoured first, so
// correctly-labelled Chinese text is never mis-decoded.
func TestFetch_PlainTextGBKDeclaredCharsetDecodes(t *testing.T) {
	resetCache()
	const chinese = "这是一段用 GBK 编码的纯文本正文。"
	body, err := encodeGBK(chinese)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	client := &fakeHTTPClient{respCT: "text/plain; charset=gbk", respBody: string(body)}

	result, err := Fetch(context.Background(), Params{URL: "https://example.cn/notes.txt"}, client)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Content != chinese {
		t.Errorf("expected the GBK body decoded to exact UTF-8; got:\n%q\nwant:\n%q", result.Content, chinese)
	}
}

// TestFetch_PlainTextLegacyBytesDecodeWithoutError asserts the BOM / sniff /
// windows-1252 fallback chain for text that is undeclared or mislabelled: a
// UTF-16 body is recognized by its BOM, unlabelled single-byte-high text
// falls back to windows-1252 (browser behaviour), and undeclared GBK decodes
// to replacement text without failing the fetch.
func TestFetch_PlainTextLegacyBytesDecodeWithoutError(t *testing.T) {
	t.Run("utf16 BOM", func(t *testing.T) {
		resetCache()
		// "café" as UTF-16LE with a BOM, served as bare text/plain.
		utf16le := []byte{0xFF, 0xFE, 'c', 0x00, 'a', 0x00, 'f', 0x00, 0xE9, 0x00}
		client := &fakeHTTPClient{respCT: "text/plain", respBody: string(utf16le)}

		result, err := Fetch(context.Background(), Params{URL: "https://example.com/legacy.txt"}, client)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		// The decoder transcodes UTF-16 to UTF-8; the BOM rune survives as
		// U+FEFF in the decoded text.
		if want := "\uFEFFcafé"; result.Content != want {
			t.Errorf("BOM-decoded content = %q, want %q", result.Content, want)
		}
	})
	t.Run("cp1252 fallback", func(t *testing.T) {
		resetCache()
		// "caf<0xE9>" — windows-1252 bytes for "café", no charset declared.
		client := &fakeHTTPClient{respCT: "text/plain", respBody: "caf\xe9 au lait"}

		result, err := Fetch(context.Background(), Params{URL: "https://example.com/legacy.txt"}, client)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.Content != "café au lait" {
			t.Errorf("windows-1252 fallback content = %q, want %q", result.Content, "café au lait")
		}
	})
	t.Run("undeclared GBK does not error", func(t *testing.T) {
		resetCache()
		body, err := encodeGBK("中文正文")
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		client := &fakeHTTPClient{respCT: "text/plain", respBody: string(body)}

		result, err := Fetch(context.Background(), Params{URL: "https://example.cn/legacy.txt"}, client)

		if err != nil {
			t.Fatalf("undeclared legacy text must not fail the fetch, got %v", err)
		}
		if result.Content == "" {
			t.Error("expected non-empty decoded content for undeclared legacy text")
		}
	})
}

// TestFetch_PlainTextEmptyBodyReturnsEmpty asserts an empty raw-text body
// passes through as empty content rather than erroring (empty files exist;
// there is nothing to decode).
func TestFetch_PlainTextEmptyBodyReturnsEmpty(t *testing.T) {
	resetCache()
	client := &fakeHTTPClient{respCT: "text/plain; charset=utf-8", respBody: ""}

	result, err := Fetch(context.Background(), Params{URL: "https://example.com/empty.txt"}, client)

	if err != nil {
		t.Fatalf("expected no error for an empty text file, got %v", err)
	}
	if result.Content != "" {
		t.Errorf("content = %q, want empty (verbatim empty body)", result.Content)
	}
}

// TestFetch_RawTextBothFormatsIdentical asserts that for a raw file the two
// return_format values return the identical body: the document already is its
// own model-friendly form, so markdown and text coincide (ADR-0006).
func TestFetch_RawTextBothFormatsIdentical(t *testing.T) {
	resetCache()
	const body = "{\"schema\": 2, \"notes\": [\"a\", \"b\"]}\n"
	client := &fakeHTTPClient{respCT: "text/plain; charset=utf-8", respBody: body}
	url := "https://example.com/data.json"

	asMarkdown, err := Fetch(context.Background(), Params{URL: url, ReturnFormat: "markdown"}, client)
	if err != nil {
		t.Fatalf("markdown format: %v", err)
	}
	asText, err := Fetch(context.Background(), Params{URL: url, ReturnFormat: "text"}, client)
	if err != nil {
		t.Fatalf("text format: %v", err)
	}

	if asMarkdown.Content != body || asText.Content != body {
		t.Errorf("both formats must return the body verbatim; markdown=%q text=%q want=%q",
			asMarkdown.Content, asText.Content, body)
	}
}

// TestFetch_RawTextCacheBehaviorUnchanged asserts the cache treats passthrough
// results like extraction results: a repeated raw-text fetch is served from
// cache (one HTTP call), and NoCache forces a fresh fetch.
func TestFetch_RawTextCacheBehaviorUnchanged(t *testing.T) {
	resetCache()
	client := &fakeHTTPClient{respCT: "text/plain; charset=utf-8", respBody: "cached raw body\n"}
	url := "https://example.com/cached-raw.txt"

	if _, err := Fetch(context.Background(), Params{URL: url}, client); err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	if _, err := Fetch(context.Background(), Params{URL: url}, client); err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if client.calls != 1 {
		t.Errorf("expected the second raw-text Fetch to hit cache (1 HTTP call), got %d", client.calls)
	}

	if _, err := Fetch(context.Background(), Params{URL: url, NoCache: true}, client); err != nil {
		t.Fatalf("no-cache fetch: %v", err)
	}
	if client.calls != 2 {
		t.Errorf("expected NoCache to bypass the cache (2 HTTP calls), got %d", client.calls)
	}
}

// TestFetch_RawTextTimeoutStillApplies asserts the request timeout governs
// raw-text fetches exactly like HTML ones — a hung host cannot block the
// session behind a text URL.
func TestFetch_RawTextTimeoutStillApplies(t *testing.T) {
	client := &fakeHTTPClient{
		respCT:   "text/plain; charset=utf-8",
		respBody: "arrives too late",
		delay:    200 * time.Millisecond,
	}

	_, err := Fetch(context.Background(), Params{URL: "https://example.com/slow.txt", Timeout: 20 * time.Millisecond}, client)

	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	if !IsTimeoutError(err) {
		t.Errorf("expected a TimeoutError, got %v", err)
	}
}
