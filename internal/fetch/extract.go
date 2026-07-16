package fetch

import (
	"strings"
	"net/url"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	readability "codeberg.org/readeck/go-readability"
)

// minReadableLen is the floor below which a Readability extraction is treated
// as "empty/low-quality" and the full-document fallback kicks in. Picked to
// reject nav/footer bleed-through while accepting any genuine short article.
const minReadableLen = 50

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
// a Readability-style extraction to isolate the main article, converts the
// result to markdown, and — when extraction yields little or nothing (e.g. a
// list page, homepage, or dashboard with no single main article) — falls back
// to full-document markdown so the reader never returns empty output.
//
// baseURL lets relative links/images in the source resolve to absolute URLs.
// Returns the extracted title (possibly empty on fallback) and the markdown.
func extract(htmlDoc, baseURL string) (title, markdown string, err error) {
	pageURL, _ := url.Parse(baseURL)
	article, extractErr := readability.FromReader(strings.NewReader(htmlDoc), pageURL)
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

// emptyContentPlaceholder is returned when a document has no extractable text
// at all (e.g. body is only script/style), so the reader honours its
// "never empty" contract.
const emptyContentPlaceholder = "(no extractable content)"
