package fetch

import (
	"strings"
	"testing"
)

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

const sampleListPageHTML = `<!DOCTYPE html>
<html><head><title>Items — Index</title></head>
<body>
<ul>
<li><a href="/a">Item A</a></li>
<li><a href="/b">Item B</a></li>
<li><a href="/c">Item C</a></li>
</ul>
</body></html>`

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

const sampleEmptyBodyHTML = `<!DOCTYPE html>
<html><head><style>body{}</style><script>track();</script></head><body></body></html>`

func TestExtract_ArticleReturnsTitleAndBody(t *testing.T) {
	title, body, err := extract(sampleArticleHTML, "https://example.com/article", formatMarkdown)

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

func TestExtract_FallsBackToFullDocument(t *testing.T) {
	_, body, err := extract(sampleListPageHTML, "https://example.com/items", formatMarkdown)

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

func TestExtract_BlogPost(t *testing.T) {
	title, body, err := extract(sampleBlogHTML, "https://dev.example.com/caching", formatMarkdown)

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

func TestExtract_DocPage(t *testing.T) {
	title, body, err := extract(sampleDocHTML, "https://docs.example.com/install", formatMarkdown)

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

func TestExtract_EmptyBodyReturnsPlaceholder(t *testing.T) {
	_, body, err := extract(sampleEmptyBodyHTML, "https://example.com/blank", formatMarkdown)

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
