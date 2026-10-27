# Raw-text passthrough — fetch returns text documents verbatim

The fetch path classifies responses by Content-Type. HTML (and XHTML) still
goes through Readability extraction as before. Responses that are already
text — all `text/*` media types, `application/json` and any `+json` suffix,
`application/xml`/`text/xml` and any `+xml` suffix, `application/yaml` and any
`+yaml` suffix, `application/javascript` — are decoded to UTF-8 and returned
verbatim: no extraction, no wrapping, no reformatting. Everything else (PDF,
images, audio/video, generic binaries, empty Content-Type) is still rejected
as unsupported.

This partially reverses the v1 stance "the reader only extracts HTML," which
silently blocked raw files (.md/.txt/.json on raw.githubusercontent.com and
similar hosts) that agents legitimately need to read (issue #11). A raw text
document is already model-friendly; extraction cannot apply (Readability is
HTML-only), and any reformatting would cross the Processing boundary defined
in CONTEXT.md. Decoding to UTF-8 and returning the bytes verbatim is
retrieval, not processing — the new **Passthrough** term in CONTEXT.md names
this path.

Binary types stay rejected on purpose: PDF parsing or image captioning would
drag in the dependency treadmill ADR-0003 exists to avoid. Known follow-ups
deliberately left out of this decision: an HTML sniff for servers that label
HTML as `text/plain`, and a response-size cap (tracked separately).

Charset decoding uses `golang.org/x/net/html/charset` (HTTP header charset →
BOM → sniff → windows-1252 fallback — the browser behavior); the modules were
already in the dependency graph as indirect requires. Both `return_format`
values (markdown/text) return the same verbatim body for raw files — the
document already is its own model-friendly form.
