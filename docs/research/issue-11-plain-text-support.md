# Research: plain-text / raw-file support (issue #11)

Date: 2026-10-27. Read-only research note; no code changed.

## Issue summary

Issue #11 (author ceeeeeeeeeeeeeeeb, OPEN): fetching raw text files — `.md`,
`.txt`, `.json` on `raw.githubusercontent.com` — fails with:

```
error: unsupported content type: text/plain; charset=utf-8
```

The error string is exactly our `UnsupportedContentError.Error()` format
(`internal/fetch/errors.go:70-72`) prefixed with `"error: "` by the MCP layer
(`internal/mcpserver/server.go:64`), so the repro path is confirmed as our
server, not the client.

## Root cause

- `internal/fetch/fetch.go:124-127`: after the 2xx check, `fetchUncached`
  accepts only `contentTypeIsHTML(ct)` and returns `UnsupportedContentError`
  otherwise.
- `internal/fetch/errors.go:110-116`: `contentTypeIsHTML` matches only
  `text/html` and `application/xhtml` (substring, case-insensitive, charset
  suffix tolerated).
- `internal/fetch/errors.go:63-65`: the doc comment states this is deliberate:
  "The reader only extracts HTML; other types are reported clearly rather than
  producing garbage" — aimed at PDF/image/binary, but it also blocks harmless
  plain text.
- The extraction pipeline (`internal/fetch/extract.go:38-58`, Readability +
  html-to-markdown) is HTML-only; `extract` is called unconditionally at
  `fetch.go:134`.

## Findings

### 1. What these endpoints actually serve (verified live, 2026-10-27)

All checks were `curl -s -D -` (GET, header dump):

| URL | Content-Type |
| --- | --- |
| `https://raw.githubusercontent.com/golang/go/master/README.md` | `text/plain; charset=utf-8` |
| `https://raw.githubusercontent.com/golang/go/master/api/go1.txt` | `text/plain; charset=utf-8` |
| `https://raw.githubusercontent.com/npm/cli/latest/package.json` | `text/plain; charset=utf-8` |
| `https://raw.githubusercontent.com/github/gitignore/main/Unity.gitignore` (extensionless `Make.dist` likewise) | `text/plain; charset=utf-8` |
| `https://gist.githubusercontent.com/octocat/6cad...86d8/hello_world.rb` | `text/plain; charset=utf-8` |
| `https://api.github.com/repos/golang/go` | `application/json; charset=utf-8` |
| `https://httpbin.org/xml` | `application/xml` |
| `https://raw.githubusercontent.com/git/git/master/git-gui/lib/git-gui.ico` (binary control) | `image/vnd.microsoft.icon` |

Two consequences: (a) raw GitHub serves **every** text file — any extension,
including `.json` and extensionless — as `text/plain; charset=utf-8`, so a
`text/*` rule alone fixes the reported repro; (b) raw GitHub serves correct
binary MIME types (`image/*`, and `application/octet-stream` for generic
binaries), so a content-type allowlist can reliably separate text from binary
on these hosts. JSON APIs additionally need `application/json`.

### 2. Fix location and constraints — solvable purely in our code

- **Branch point**: in `fetchUncached`, replace the binary check at
  `fetch.go:124-127` with a three-way classification (HTML → extraction as
  today; text → passthrough; anything else → `UnsupportedContentError` as
  today). The text branch bypasses `extract` (`fetch.go:134`) and returns
  `&Result{Title: ..., Content: decodedBody}` directly, mirroring
  `fetch.go:138`.
- **Result semantics**: `Result` is `{Title, Content}`
  (`fetch.go:37-40`). For plain text, `Content` = the decoded body verbatim;
  `Title` can be derived from the URL path basename (`path.Base` of the parsed
  URL, extension kept or stripped — cosmetic, see next point). Low-stakes
  because the MCP layer **never sends Title to the client**: the tool result is
  built from `result.Content` only (`internal/mcpserver/server.go:68-70`).
- **MCP layer constraints**: none. The tool input schema
  (`server.go:26-31`) is unchanged; output is a single `TextContent`
  (`server.go:68-70`) — any string fits. The only text worth touching is the
  tool description ("...return its main content as model-friendly markdown",
  `server.go:45`), which should gain a clause about raw-text passthrough.
- **Cache constraints**: none. The cache stores any `*Result` keyed by
  `(return_format, url)` (`internal/fetch/cache.go:45-68, 72-78`); errors are
  never cached (`cache.go:59-62`). Note both `markdown` and `text` formats of
  the same URL will cache identical copies under separate keys — already true
  for HTML today, so no new behavior.
- **Verdict**: **solvable**. No dependency or API limitation blocks it; the
  reference MCP fetch server does the same thing in Python (see §4).

### 3. Charset handling — where it happens today, and what a text branch needs

- Today's HTML charset decoding is **not** in our code (`fetch.go:134` does a
  plain `string(body)`) and **not** in go-readability proper
  (`codeberg.org/readeck/go-readability` `readability.go:22-26` →
  `parser-parse.go:17-20`). It happens in `github.com/go-shiori/dom`'s
  `Parse` (`parser.go:28-55` in the module cache), which go-readability calls:
  it reads all bytes, runs `chardet.NewHtmlDetector().DetectBest`, resolves the
  name via `golang.org/x/net/html/charset.Lookup`, wraps the reader in the
  x/text decoder, NFC-normalizes, then `html.Parse`.
- `internal/fetch/encoding_test.go:15-23` documents exactly this reliance and
  locks it in with three GBK regression tests (`encoding_test.go:62-118`).
  Precision note: its comment says the parser "reads `<meta charset>` / HTTP
  Content-Type / sniffs" — the HTTP header is in fact **not** consulted
  (fetch.go never passes it downstream); the header-only test passes because
  chardet's byte sniffing detects GBK anyway. A fix must not "correct" that
  comment away — it is accurate in spirit.
- **For a text/plain branch we need our own decode step**, and everything
  required is already in the module graph:
  - `golang.org/x/net/html/charset.DetermineEncoding(content, contentType)`
    (`golang.org/x/net@v0.55.0/html/charset/charset.go:52-99`): BOM →
    Content-Type `charset` parameter → HTML meta prescan (no-op for plain
    text) → UTF-8 validity → `windows-1252` fallback. There is also
    `charset.NewReader(r, contentType)` (`charset.go:105-121`) which wraps the
    whole dance into one reader — the natural fit for the branch, since it
    actually uses the HTTP header (better than chardet for text/plain).
  - `golang.org/x/text/encoding` provides the decoders (used by x/net internally).
  - `github.com/gogs/chardet` also has `NewTextDetector()`
    (`detector.go:72-75`) as an alternative sniffer — not needed if x/net's
    `NewReader` is used.
- **go.mod status (precise)**: `github.com/gogs/chardet`
  (`go.mod:16`), `golang.org/x/net v0.55.0` (`go.mod:21`) and
  `golang.org/x/text v0.37.0` (`go.mod:24`) are all present as **indirect**
  dependencies today. Importing x/net (and x/text if used directly) in
  `internal/fetch` would move them to the **direct** `require` block via
  `go mod tidy` — **no new modules, no new versions, no downloads**. Strictly
  speaking "no new dependencies" means "no new modules"; three existing
  indirect modules get promoted.

### 4. Scope benchmark — what comparable tools accept (primary sources)

- **Reference MCP fetch server** (`modelcontextprotocol/servers`,
  `src/fetch/src/mcp_server_fetch/server.py:362-375`, verified 2026-10-27 —
  https://github.com/modelcontextprotocol/servers/blob/main/src/fetch/src/mcp_server_fetch/server.py):
  it classifies a response as HTML only when `"<html"` appears in the first
  100 characters, or `text/html` is in the Content-Type, or the Content-Type
  is empty. Everything else — **any** content type — is returned as raw text
  with the prefix: *"Content type {content_type} cannot be simplified to
  markdown, but here is the raw content:"*. It rejects nothing.
- **Jina Reader** (`https://jina.ai/reader/`, FAQ): converts any URL to
  markdown; PDFs are natively supported; images are captioned; nothing is
  rejected except non-public URLs. That breadth (PDF extraction, image
  captioning) would be **Processing** under our CONTEXT.md vocabulary and is
  not the benchmark to copy — but it confirms no comparable tool rejects
  `text/*`.
- Positioning: our repo's static-HTTP stance (ADR-0003) puts us between the
  two: pass text through verbatim (free), skip binary processing (would drag
  in PDF/OCR stacks that violate near-zero maintenance).

### 5. Consistency with CONTEXT.md and the ADRs

- **Web Reader** (CONTEXT.md:9-13): "takes a URL ... and returns the content
  at that URL, parsed into model-friendly form." Returning a raw text file
  verbatim **is** returning the content in model-friendly form — a `.md`/`.txt`
  body is already model-friendly. Fits the definition; no vocabulary change
  strictly required.
- **Extraction** (CONTEXT.md:20-24) is explicitly HTML-specific ("turns
  fetched HTML into ..."). A plain-text response simply **skips** extraction —
  the vocabulary describes the HTML path, it does not forbid other paths.
- **Processing** (CONTEXT.md:26-33) — "any transformation *beyond* fetch +
  extract" — is not violated: decode-to-UTF-8 + verbatim return is retrieval,
  not generation/embellishment. (This is also why we should NOT re-wrap or
  reformat raw files.)
- **ADR-0003**: unaffected. Still plain HTTP, still Readability for HTML, no
  JS/browser added.
- **ADR-0005**: unaffected. The new branch lives inside `fetchUncached`, which
  is already inside `Fetch`'s panic recovery (`fetch.go:79-83`).
- However, `errors.go:63-65` records a *deliberate architecture decision*
  ("The reader only extracts HTML"), and the fix reverses part of it. Per the
  repo's ADR conventions (`docs/adr/0001..0005`: `NNNN-kebab-title.md`, a
  short imperative title plus rationale prose, no status/date headers), this
  warrants a small new ADR — recommend `docs/adr/0006-raw-text-passthrough.md`
  — rather than silently editing the old comment. Amending the
  `UnsupportedContentError` doc comment is part of the fix either way.

### 6. ReturnFormat semantics for plain text

- Current reality: `Params.ReturnFormat` (`fetch.go:26-31`) is exposed in the
  tool schema (`server.go:29`: "Output format: markdown or text. Defaults to
  markdown."), listed in the README table (`README.md:40`), and used in the
  cache key (`cache.go:72-78`) — but **no code in the fetch path ever reads
  it**. `fetchUncached`/`extract` always produce markdown; `grep` across
  non-test code confirms the only consumers are the schema, the param mapping
  (`server.go:80`), and `cacheKey`. So "text" today is an alias of "markdown"
  that merely double-caches; there is no established behavior to stay
  consistent with, and the plain-text branch is the place to define it.
- Recommendation: **both `markdown` and `text` return the body verbatim.**
  Rationale: (a) reformatting or boilerplate-stripping a raw file would be
  Processing (CONTEXT.md:26-33); (b) a `.md` file *is* already markdown —
  converting it changes nothing and wrapping it in fences would damage
  fidelity; (c) the reference MCP fetch server likewise returns non-HTML
  bodies raw for both modes (server.py:370-375). Optionally document in the
  schema/README that for raw files the two formats coincide.

## Recommended fix sketch

Content-type decision table (parse media type from the header, lowercase,
ignore parameters for the match):

| Classification | Types | Action |
| --- | --- | --- |
| HTML | `text/html`, `application/xhtml+xml` (existing `contentTypeIsHTML`) | Readability + markdown, as today |
| **Pass through** | all `text/*` (covers `text/plain`, `text/csv`, `text/markdown`, `text/xml`, `text/javascript`, ...); `application/json` and any `+json` suffix; `application/xml`, `text/xml`, any `+xml` suffix; `application/yaml`, `application/x-yaml`, any `+yaml` suffix; `application/javascript` | decode to UTF-8, return verbatim |
| Reject | everything else: `application/pdf`, `image/*`, `video/*`, `audio/*`, `application/octet-stream`, `application/zip`, empty Content-Type (conservative; see open questions) | `UnsupportedContentError` (unchanged error type) |

- **Where**: `fetchUncached` (`fetch.go:124-127`) grows the three-way
  classification; the text branch skips `extract` and returns
  `&Result{Title: titleFromURL(params.URL), Content: decoded}`.
- **Charset**: wrap the body with `charset.NewReader(bytes.NewReader(body), ct)`
  from `golang.org/x/net/html/charset` (already in the module graph, see §3),
  then read; it honors the `charset` parameter, falls back to BOM/UTF-8
  sniffing, and defaults to windows-1252 like browsers do. No chardet call
  needed.
- **Title/Content semantics**: `Content` verbatim; `Title` from
  `path.Base(parsedURL.Path)` (cosmetic — the MCP layer currently discards
  Title, `server.go:68-70`).
- **Cache/format**: no changes. Both formats return the same string; entries
  land under separate cache keys exactly as HTML does today.
- **Tests**: the existing PDF rejection test (`internal/fetch/errors_test.go:76-90`)
  stays green (PDF remains rejected). New tests needed: `text/plain; charset=utf-8`
  passthrough (the issue repro), `application/json`, `+json` suffix, GBK
  `text/plain` decode, and a still-rejected binary. Update the
  `UnsupportedContentError` doc comment (`errors.go:63-65`) and optionally the
  tool description (`server.go:45`) + README parameter table.
- **ADR**: add short `docs/adr/0006-raw-text-passthrough.md` recording the
  decision and its Processing-boundary rationale (§5).

## Open questions / risks

- **Unbounded body size**: `fetch.go:129` uses bare `io.ReadAll` with no cap —
  pre-existing for HTML, but raw files make multi-megabyte CSV/JSON more
  likely to hit the model's context. Consider a max-bytes guard with a
  truncation marker as a follow-up (not required to fix the issue).
- **Lying servers**: some endpoints serve HTML as `text/plain`. The reference
  server sniffs `"<html"` in the first 100 bytes (server.py:362-365) and
  extracts anyway. We could adopt the same cheap sniff for pass-through
  bodies; recommend as an optional refinement, not part of the minimal fix.
- **Empty Content-Type**: currently rejected by `contentTypeIsHTML("")` ==
  false. The reference server treats it as HTML. Recommend keeping rejection
  for now (no evidence of real-world need) and revisiting only if reported.
- **Markdown-as-instructions exposure**: returning a raw `.md` verbatim means
  prompt-style content in the file reaches the model unquoted — identical to
  the reference server's raw mode and to any `cat`, so accepted; noted for
  completeness.
- **Promoted indirect deps**: x/net (and possibly x/text) moving from indirect
  to direct in `go.mod` is expected churn for reviewers; chardet can stay
  indirect.

## Verdict

**Solvable in-repo, with no new module dependencies** (three existing indirect
modules — `golang.org/x/net`, `golang.org/x/text`, optionally `gogs/chardet` —
get promoted to direct imports; `go mod tidy` handles it with zero new
versions). No MCP-schema, cache, or ADR-0003/0005 constraint blocks it; the
only documentation debt is one small new ADR plus comment/README touch-ups.
The pass-through list recommended above (`text/*`, `application/json` +
`+json`, `application/xml`/`text/xml` + `+xml`, `application/yaml` + `+yaml`,
`application/javascript`) fixes the exact repro (`text/plain; charset=utf-8`)
and matches or subset-matches the reference MCP fetch server's behavior.
