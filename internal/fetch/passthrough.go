package fetch

import (
	"bytes"
	"io"
	"net/url"
	"path"
	"strings"

	"golang.org/x/net/html/charset"
)

func isRawTextContentType(ct string) bool {
	media := mediaType(ct)
	if media == "" {
		return false
	}
	if contentTypeIsHTML(ct) {
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

func mediaType(ct string) string {
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return strings.ToLower(strings.TrimSpace(ct))
}

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
