package fetch

import (
	"bytes"
	"io"
	"net/url"
	"path"
	"strings"

	"golang.org/x/net/html/charset"
)

// This file implements the Passthrough path (CONTEXT.md): a response that is
// already a raw text document is decoded to UTF-8 and returned verbatim — no
// extraction, no wrapping, no reformatting. It is the path parallel to
// Extraction (extract.go), taken when the body is already model-friendly
// as-is (ADR-0006). Binary types never reach here; they stay rejected.

// isRawTextContentType reports whether a Content-Type header value denotes a
// raw text document. Per ADR-0006 the accepted types are: all text/* media
// types; application/json and any +json suffix; application/xml and any +xml
// suffix; application/yaml, application/x-yaml and any +yaml suffix; and
// application/javascript. HTML/XHTML deliberately do not match — callers
// classify those first so they keep taking the Extraction path.
func isRawTextContentType(ct string) bool {
	media := mediaType(ct)
	if media == "" {
		return false
	}
	if strings.HasPrefix(media, "text/") {
		return true
	}
	switch media {
	case "application/json", "application/xml", "application/yaml",
		"application/x-yaml", "application/javascript":
		return true
	}
	return strings.HasSuffix(media, "+json") ||
		strings.HasSuffix(media, "+xml") ||
		strings.HasSuffix(media, "+yaml")
}

// mediaType extracts the lowercase media type from a Content-Type header
// value, dropping any parameters ("Text/Plain; charset=utf-8" → "text/plain").
// It is the single content-type parser: both classification checks
// (contentTypeIsHTML and isRawTextContentType) run on its output, so the
// fetch switch dispatches on consistent input.
func mediaType(ct string) string {
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return strings.ToLower(strings.TrimSpace(ct))
}

// passthroughResult builds the verbatim result for a raw-text response: the
// body decoded to UTF-8 becomes Content unchanged; Title comes from the URL
// path basename.
func passthroughResult(body []byte, contentType, rawURL string) (*Result, error) {
	decoded, err := decodeBody(body, contentType)
	if err != nil {
		return nil, &FetchError{Err: err}
	}
	return &Result{Title: titleFromURL(rawURL), Content: decoded}, nil
}

// decodeBody converts raw-text bytes to a UTF-8 string using the browser's
// encoding-determination rules via golang.org/x/net/html/charset: a declared
// HTTP charset parameter wins, then BOM detection, then content sniffing,
// with windows-1252 as the fallback. Valid UTF-8 passes through byte-for-byte.
// An empty body decodes to "" (charset.NewReader would report EOF for it).
func decodeBody(body []byte, contentType string) (string, error) {
	if len(body) == 0 {
		return "", nil
	}
	r, err := charset.NewReader(bytes.NewReader(body), contentType)
	if err != nil {
		return "", err
	}
	decoded, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// titleFromURL derives a passthrough result's title from the URL path
// basename ("https://host/dir/file.md" → "file.md"). Cosmetic only: the MCP
// layer does not expose Title. Falls back to the host when the path has no
// basename (no path, or just "/").
func titleFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if base := path.Base(u.Path); base != "/" && base != "." {
		return base
	}
	return u.Host
}
