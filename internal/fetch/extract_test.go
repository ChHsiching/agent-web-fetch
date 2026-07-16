package fetch

import (
	"strings"
	"testing"
)

// sampleArticleHTML is a pre-recorded, representative article page: a clear
// main article with boilerplate (nav, footer, script) around it. Extraction
// should keep the article body and title and drop the boilerplate.
const sampleArticleHTML = `<!DOCTYPE html>
<html><head><title>Real Article Title — Site</title></head>
<body>
<nav><a href="/">Home</a> <a href="/about">About</a></nav>
<article>
<h1>Real Article Title</h1>
<p>This is the first paragraph of the article. It contains the substance the reader wants.</p>
<p>A second paragraph follows, with <a href="https://example.com/ref">a reference link</a> and an image: <img src="https://example.com/img.png" alt="diagram"/>.</p>
</article>
<footer>© 2026 Site. <a href="/privacy">Privacy</a></footer>
<script>tracker();</script>
</body></html>`

// sampleListPageHTML is a page with no single main article — a list/index.
// Readability should extract little/nothing here, triggering the full-document
// fallback so the reader still returns useful content (never empty).
const sampleListPageHTML = `<!DOCTYPE html>
<html><head><title>Items — Index</title></head>
<body>
<ul>
<li><a href="/a">Item A</a></li>
<li><a href="/b">Item B</a></li>
<li><a href="/c">Item C</a></li>
</ul>
</body></html>`

// sampleBlogHTML represents a blog post: dated, authored prose with headings.
// Readability should keep the post body and drop the sidebar.
const sampleBlogHTML = `<!DOCTYPE html>
<html><head><title>Notes on Caching — Dev Blog</title></head>
<body>
<aside><h3>Subscribe</h3><p>Get updates by email.</p></aside>
<article>
<h2>Notes on Caching</h2>
<p><em>By Ada, 2026-07-10.</em></p>
<p>A short blog post about why simple caches beat clever ones. The first lesson is that expiry matters more than eviction policy.</p>
<p>The second lesson is that an LRU keyed on the full request shape is usually enough.</p>
</article>
</body></html>`

// sampleDocHTML represents a documentation page: a titled section with prose,
// a code block, and a list of steps.
const sampleDocHTML = `<!DOCTYPE html>
<html><head><title>Installation — Acme Docs</title></head>
<body>
<nav>Acme Docs › Installation</nav>
<main>
<h1>Installation</h1>
<p>To install Acme, run the following in your terminal.</p>
<pre><code>acme init my-project</code></pre>
<ol>
<li>Create a project with the command above.</li>
<li>Change into the new directory.</li>
<li>Run acme serve to start the dev server.</li>
</ol>
</main>
</body></html>`

// sampleEmptyBodyHTML is a document with no markdownable text — only script
// and style. Extraction must still return non-empty output (the never-empty
// contract), via the placeholder.
const sampleEmptyBodyHTML = `<!DOCTYPE html>
<html><head><style>body{}</style><script>track();</script></head><body></body></html>`

// TestExtract_ArticleReturnsTitleAndBody asserts the core T3 behaviour: given
// an article-type HTML page, extract returns the article title and a markdown
// body that contains the article's substance and preserves image URLs.
func TestExtract_ArticleReturnsTitleAndBody(t *testing.T) {
	title, body, err := extract(sampleArticleHTML, "https://example.com/article")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(title, "Real Article Title") {
		t.Errorf("title = %q, want it to contain \"Real Article Title\"", title)
	}
	if !strings.Contains(body, "first paragraph of the article") {
		t.Errorf("body missing article substance; got:\n%s", body)
	}
	if !strings.Contains(body, "https://example.com/ref") {
		t.Errorf("body missing reference link; got:\n%s", body)
	}
	if !strings.Contains(body, "https://example.com/img.png") {
		t.Errorf("body missing image URL; got:\n%s", body)
	}
	if strings.Contains(body, "tracker()") {
		t.Errorf("body should drop script; got:\n%s", body)
	}
}

// TestExtract_FallsBackToFullDocument asserts that when extraction yields
// little/no content (a list page with no main article), the full-document
// fallback kicks in and the reader still returns non-empty useful content.
func TestExtract_FallsBackToFullDocument(t *testing.T) {
	_, body, err := extract(sampleListPageHTML, "https://example.com/items")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if body == "" {
		t.Fatal("expected non-empty body via fallback, got empty string")
	}
	if !strings.Contains(body, "Item A") {
		t.Errorf("fallback body should contain list items; got:\n%s", body)
	}
}

// TestExtract_BlogPost keeps the post body and drops the sidebar, covering the
// "blog" representative page type.
func TestExtract_BlogPost(t *testing.T) {
	title, body, err := extract(sampleBlogHTML, "https://dev.example.com/caching")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(title, "Notes on Caching") {
		t.Errorf("title = %q, want \"Notes on Caching\"", title)
	}
	if !strings.Contains(body, "expiry matters more than eviction policy") {
		t.Errorf("body missing post substance; got:\n%s", body)
	}
	if strings.Contains(strings.ToLower(body), "subscribe") {
		t.Errorf("body should drop sidebar; got:\n%s", body)
	}
}

// TestExtract_DocPage keeps the install steps and code block, covering the
// "doc" representative page type.
func TestExtract_DocPage(t *testing.T) {
	title, body, err := extract(sampleDocHTML, "https://docs.example.com/install")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(title, "Installation") {
		t.Errorf("title = %q, want \"Installation\"", title)
	}
	if !strings.Contains(body, "acme init my-project") {
		t.Errorf("body missing code block; got:\n%s", body)
	}
	if !strings.Contains(body, "Run acme serve") {
		t.Errorf("body missing step content; got:\n%s", body)
	}
}

// TestExtract_EmptyBodyReturnsPlaceholder asserts the never-empty contract: a
// document with no markdownable text still returns non-empty output.
func TestExtract_EmptyBodyReturnsPlaceholder(t *testing.T) {
	_, body, err := extract(sampleEmptyBodyHTML, "https://example.com/blank")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if strings.TrimSpace(body) == "" {
		t.Fatal("expected non-empty output for script/style-only document, got empty")
	}
	if strings.Contains(body, "track()") {
		t.Errorf("body should drop script; got:\n%s", body)
	}
}
