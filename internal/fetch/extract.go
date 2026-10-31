package fetch

import (
	"net/url"
	"strings"

	readability "codeberg.org/readeck/go-readability"
	"codeberg.org/readeck/go-readability/render"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"golang.org/x/net/html"
)

// minReadableLen is the floor below which a Readability extraction is treated
// as "empty/low-quality" and the full-document fallback kicks in. Picked to
// reject nav/footer bleed-through while accepting any genuine short article.
const minReadableLen = 50

// The two return_format values the reader accepts for HTML pages (CONTEXT.md:
// Extraction produces "model-friendly content (markdown/text)"). The empty
// string means the markdown default, the same convention cacheKey uses;
// anything else is rejected by Fetch before it reaches here.
const (
	formatMarkdown = "markdown"
	formatText     = "text"
)

// mdConverter converts HTML fragments to GitHub-flavored markdown (base +
// commonmark + table), preserving links and images so the model can reference
// them. Configured once; safe for concurrent use.
var mdConverter = converter.NewConverter(
	converter.WithPlugins(
		base.NewBasePlugin(),
		commonmark.NewCommonmarkPlugin(),
		table.NewTablePlugin(),
	),
)

// extract turns a fetched HTML document into model-friendly content. It runs
// a Readability-style extraction to isolate the main article, renders the
// result as markdown (formatMarkdown) or plain text (formatText), and — when
// extraction yields little or nothing (e.g. a list page, homepage, or
// dashboard with no single main article) — falls back to the whole document
// in the same format so the reader never returns empty output.
//
// baseURL lets relative links/images in the source resolve to absolute URLs.
// Returns the extracted title (possibly empty on fallback) and the content.
// The format must already be validated and normalized by Fetch.
func extract(htmlDoc, baseURL, format string) (title, content string, err error) {
	pageURL, _ := url.Parse(baseURL)
	article, extractErr := readability.FromReader(strings.NewReader(htmlDoc), pageURL)
	if format == formatText {
		return extractPlainText(htmlDoc, article, extractErr)
	}
	if extractErr == nil && len(strings.TrimSpace(article.TextContent)) >= minReadableLen {
		md, convErr := mdConverter.ConvertString(article.Content)
		if convErr == nil && strings.TrimSpace(md) != "" {
			return article.Title, md, nil
		}
	}
	// Fallback: convert the whole document to markdown. The reader never
	// returns empty output — even a document with no markdownable text yields
	// a placeholder rather than an empty string.
	md, fallbackErr := mdConverter.ConvertString(htmlDoc)
	if fallbackErr != nil {
		return "", "", fallbackErr
	}
	if strings.TrimSpace(md) == "" {
		return "", emptyContentPlaceholder, nil
	}
	return "", md, nil
}

// extractPlainText is the return_format: text arm of Extraction (spec #16):
// the readability article's own plain text — the same TextContent the
// markdown arm renders from, with no markdown syntax emitted. When the page
// has no article, the fallback returns the whole document's text via
// render.InnerText, the block-aware renderer the article's TextContent is
// itself built from, so the fallback reads like the article arm minus
// extraction. Never empty: a document with no text at all yields the
// placeholder, mirroring the markdown fallback's contract.
func extractPlainText(htmlDoc string, article readability.Article, extractErr error) (title, text string, err error) {
	if extractErr == nil && len(strings.TrimSpace(article.TextContent)) >= minReadableLen {
		return article.Title, strings.TrimSpace(article.TextContent), nil
	}
	doc, parseErr := html.Parse(strings.NewReader(htmlDoc))
	if parseErr != nil {
		return "", "", parseErr
	}
	if text := strings.TrimSpace(render.InnerText(doc)); text != "" {
		return "", text, nil
	}
	return "", emptyContentPlaceholder, nil
}

// emptyContentPlaceholder is returned when a document has no extractable text
// at all (e.g. body is only script/style), so the reader honours its
// "never empty" contract.
const emptyContentPlaceholder = "(no extractable content)"
