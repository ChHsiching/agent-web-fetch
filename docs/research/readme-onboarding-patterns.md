# Research: README onboarding patterns (引导) for agent-web-fetch

Date: 2026-10-31. Read-only research note; no code changed (this file is the
only addition). Purpose: study how high-star, comparable tools write the
onboarding/guidance sections of their READMEs, and recommend a section skeleton
for our README's install / MCP-config / usage onboarding.

## Method and sources

Primary sources only: the actual README files, fetched 2026-10-31 via
`raw.githubusercontent.com`. Star counts from the GitHub REST API, same date.
Repos below 1k stars were considered and excluded per the task constraint
(e.g. `zcaceres/fetch-mcp` 835★, `dondai1234/master-fetch` 898★); vendor MCP
front-ends for paid services were not profiled beyond the firecrawl one.

| Repo | Stars (2026-10-31) | README studied |
| --- | --- | --- |
| modelcontextprotocol/servers | 91,706 | root + `src/fetch/` |
| junegunn/fzf (Go CLI contrast) | 83,876 | root |
| firecrawl/firecrawl-mcp-server | 7,729 | root |
| AgentDeskAI/browser-tools-mcp | 7,341 | root |
| KnockOutEZ/wigolo | 5,508 | root |
| goreleaser/goreleaser (Go CLI contrast) | 16,127 | root |
| jae-jae/fetcher-mcp | 1,087 | root |

## Per-repo findings

### 1. modelcontextprotocol/servers — root README (91,706★)

https://github.com/modelcontextprotocol/servers/blob/main/README.md

Skeleton: title → intro → `[!IMPORTANT]` + `[!WARNING]` GitHub alerts → SDK
list → **🌟 Reference Servers** (one-line-per-server bullet list, each linking
to its subdirectory README) → Archived → **🚀 Getting Started** (Using MCP
Servers in this Repository / Using an MCP Client / Transports: stdio and
Streamable HTTP) → 🛠️ Creating Your Own Server → 📚 Learn More → 🤝
Contributing → 📦 Releasing → 🔒 Security → 📜 License → 💬 Community → ⭐
Support.

Onboarding mechanics:

- Quick start is a copy-paste `npx`/`uvx` command per server type
  (`npx -y @modelcontextprotocol/server-memory`, `uvx mcp-server-git`),
  followed immediately by one Claude Desktop `mcpServers` JSON block and the
  Windows `cmd /c` wrapper variant.
- A **Transports explainer** tells stdio users the server "appears to hang.
  That is expected ... You do not normally start a stdio server yourself; your
  MCP client starts it from its configuration" (their words). This is the
  highest-star precedent for our "You never launch it yourself" note.
- MCP Inspector promoted as the try-it-without-a-client tool.
- Per-server documentation lives in subdirectory READMEs; the root gives each
  server exactly one line plus a link.

Formatting: emoji prefix on every H2; sentence-ish case; `sh` and `json`
fences; no badges, no GIF, no screenshots, no comparison table, no FAQ.

### 2. modelcontextprotocol/servers — `src/fetch/` README (the fetch server)

https://github.com/modelcontextprotocol/servers/blob/main/src/fetch/README.md

Skeleton: title → `<!-- mcp-name: io.github.modelcontextprotocol/server-fetch -->`
registry marker → one-sentence description → Source link → SDK requirement →
`[!CAUTION]` SSRF callout → chunked-reading note → ### Available Tools
(bullet list: tool + typed params) → ### Prompts → ### Errors → ### Protocol
versions → ## Installation (Using uv (recommended) / Using PIP) → ##
Configuration (Configure for Claude.app / Configure for VS Code / four
"Customization - X" subsections) → ## Windows Configuration → ## Debugging →
## Development → ## Contributing → ## License.

Onboarding mechanics:

- **`<details>` collapsibles per install method** — the Claude.app section
  offers three collapsed variants (uvx / docker / pip), each a complete
  `claude_desktop_config.json` block. This keeps one config shape canonical
  while covering alternatives.
- **VS Code one-click install badges** (shields.io buttons hitting
  `insiders.vscode.dev/redirect/mcp/install?...`) plus manual
  `mcp.json` instructions for VS Code.
- Windows gets its own H2 with a dedicated encoding-env-var fix and two more
  collapsed JSON blocks.
- Debugging = MCP Inspector command. License = prose paragraph, MIT.

Deliberately left out: badges for stars/CI, demo GIF or screenshot, "why this
tool" positioning (unnecessary — it is the reference implementation), FAQ,
comparison table. 318 lines.

### 3. firecrawl/firecrawl-mcp-server (7,729★)

https://github.com/firecrawl/firecrawl-mcp-server/blob/main/README.md

Skeleton: centered logo `<div>` → title → one-paragraph description → thanks
callout → ## Features (bullets) → ## When to Use This Server → ## Installation
(### Hosted MCP (keyless free tier) / Running with npx / Manual Installation /
Running on Cursor / Running on Windsurf / Running with Streamable HTTP Local
Mode / Running with Docker / Running on VS Code) → ## Configuration
(Environment Variables / Configuration Examples / Usage with Claude Desktop) →
## How to Choose a Tool (+ ### Quick Reference Table, ### Format Selection
Guide) → ## Available Tools (16 numbered ### per tool) → ## Programmatic use →
## Logging System → ## Error Handling → ## Development → Contributing →
License → appendix sections (after the license — a defect, not a pattern to
copy).

Onboarding mechanics:

- Installation leads with the **"Hosted MCP (keyless free tier)"** — the free
  entry point is placed first, before anything needing an API key.
- **Per-client H3 subsections** with exact JSON; Cursor gets numbered UI steps
  pinned to editor versions ("To configure Firecrawl MCP in Cursor v0.48.6:
  1. Open Cursor Settings ..."), plus the Windows `cmd /c "set ... && npx"`
  escape hatch.
- Claude Desktop config lives under Configuration, not Installation.
- "How to Choose a Tool" = if/then bullets + a **Quick Reference Table**
  (Tool | Best for | Returns). The table is about *their own* tools, not a
  competitor comparison.

Formatting: Title Case H2s; `bash`/`json` fences; zero badges; no GIF; no
FAQ. 1,424 lines — dominated by the per-tool catalog; the onboarding portion
(lines 1–400) is the relevant part for us.

### 4. AgentDeskAI/browser-tools-mcp (7,341★)

https://github.com/AgentDeskAI/browser-tools-mcp/blob/main/README.md

Skeleton: title → bold version-urgency paragraph → one-paragraph value prop
naming concrete clients ("Cursor, Claude Code, Windsurf, Cline, Zed, Gemini
CLI") → blockquote version note → **## Why this instead of a CDP-based
server** → ## Install (### 1. Point your MCP client at the server / ### 2.
Load the Chrome extension / ### 3. Use it) → ## Tools (table) + subsections →
## Privacy and security → ## Configuration (flags table) → ## Known limits →
## Development → ## License ("MIT", one line).

Onboarding mechanics — the tightest of the set (288 lines):

- **"Why this instead of X"**: a named-competitor positioning section right
  after the pitch ("Tools like Chrome DevTools MCP and Playwright MCP drive a
  *fresh, automated* browser. That is the right choice for writing tests. It
  is the wrong choice for ..."). This is the direct precedent for our
  "free replacement for paid web-readers" stance.
- Install is exactly **three numbered steps**; step 1 is ONE generic
  `mcpServers` JSON block (not per-client walls), plus the Windows `cmd /c`
  npx wrapper and a stated runtime version ("Requires **Node 22.19 or
  newer**. Check with `node --version`").
- **"### 3. Use it" gives example prompts** to type into the agent ("check
  the console for errors") — onboarding ends with what to *say*, not just
  what to install.
- Troubleshooting is embedded: "Not working? Run `npx ... --doctor`, which
  reports exactly which piece is missing", and a `--verbose` sample output in
  a code fence so users can see what success looks like.
- "Known limits" is an honest H2 near the end (analogous to our
  "What it does *not* do").

Formatting: sentence case headings, zero emoji, zero badges, no GIF, tables
for tools and flags. License: one word + link.

### 5. KnockOutEZ/wigolo (5,508★) — closest positioning comparable

https://github.com/KnockOutEZ/wigolo/blob/main/README.md

Tagline: "Local-first web intelligence for AI agents — **no keys, no cloud,
no metered bill.**" Same AGPL-3.0 + commercial-license model as ours.

Skeleton: centered banner → tagline → works-with client list → **badge row
(10 shields: npm, downloads, GitHub stars, CI, node, MCP, license AGPL,
status, X, Discord)** → Trendshift/sponsor badges → in-page anchor nav →
pitch paragraph → **demo GIF** → ## Quickstart → ### Recommended — a free key
... → ## What your agent gets back (annotated `jsonc` example) → ## Tools
(emoji table) → ## Why it's different → ## Benchmark (GIF + **### How it
compares** table) → ## Beyond your editor (REST/SDKs/Docker/skills) → ## Star
history → ## Architecture → ## Configuration → ## Docs & examples → ## Beta &
feedback → ## Troubleshooting → ## FAQ (`<details>` collapsibles) → ##
Available on → ## Sponsors → ## Contributing → ## License.

Onboarding mechanics:

- Quickstart is **two copy-paste commands** (`npx wigolo init`, optionally
  `--agents=claude-code,cursor` to auto-wire clients) + `npx wigolo doctor`
  health check + a clean uninstall command. Per-client config is delegated to
  a docs/installation.md ("has the exact config block for every client").
- **Competitor comparison table**: columns wigolo / Firecrawl / Exa / Tavily;
  rows are capabilities plus the two positioning rows — "API key / account:
  none | required | required | required" and "Cost per query: $0 | metered |
  metered | metered" — with a dated footnote ("Feature standing as of July
  2026 — check each vendor's docs for current state"). This is exactly the
  shape our "free/uncapped vs paid web-reader" note wants.
- **FAQ in `<details>` collapsibles**, including "Free? What's the catch?"
  and "Can I use this at work?" (commercial-license answer) — the two
  questions our AGPL + free positioning will get.
- "What your agent gets back" shows a real (abridged) tool response in
  `jsonc` before any tool catalog — evidence-first onboarding.

Formatting: mixed Title Case H2s; emoji in the tools table; heavy HTML
centering; 445 lines. The most marketing-complete README in the set (also the
only one with a GIF, star history, sponsors).

### 6. jae-jae/fetcher-mcp (1,087★)

https://github.com/jae-jae/fetcher-mcp/blob/main/README.md

Skeleton: centered icon → **8-language i18n link row** → title → one-liner →
## Advantages (bold-label bullets) → ## Quick Start (npx + one-time
`playwright install chromium`) → HTTP/SSE transport → Debug Mode → ##
Configuration MCP (macOS/Windows `claude_desktop_config.json` paths + one
JSON block) → ## Docker Deployment (compose file) → ## Features (tools with
per-param bullets) → ## Tips → ## Development → ## Debugging (MCP Inspector)
→ ## Related Projects → ## License (MIT link).

Notable for two ideas: (a) the **Tips section maps natural-language prompts
to parameters** ("Please wait for the page to fully load" → sets
`waitForNavigation: true`) — onboarding aimed at the *human talking to the
agent*, not the human editing JSON; (b) i18n links at the top justify our
English | 简体中文 switcher. No badges, no GIF, no FAQ, no comparison.

### 7. Go CLI contrast (2 examples only)

**junegunn/fzf** (83,876★,
https://github.com/junegunn/fzf/blob/master/README.md): centered logo +
badge row (build, version, license, contributors, sponsors, stars) →
screenshot (static PNG, not GIF) → "Highlights" with the **`**Label** //
description`** bullet style — first highlight: "Portable // Distributed as a
single binary for easy installation" → generated Table of Contents → ##
Installation with **one H3 per install channel** (Homebrew / Mise / Linux
packages / Windows packages / git / Binary releases) → Upgrading → Building →
usage manual. The single-binary selling point and per-channel install
subsections are the transferable parts.

**goreleaser/goreleaser** (16,127★,
https://github.com/goreleaser/goreleaser/blob/main/README.md): the minimal
pole — logo, one-liner, "## Get GoReleaser" as two links out to the docs
site, Documentation/Community sections, and **all 12 badges at the bottom**
of the file. Install is fully delegated to a website. A deliberate pattern
(README as landing page), but wrong for us: we have no docs site, and our
README must carry the config JSON itself.

## Cross-cutting conventions

- **Config JSON**: always complete, copy-pasteable `mcpServers` blocks fenced
  as ```json; the server key name is arbitrary and every repo treats it as
  such. Nobody ships config for more than ~4 clients inline; extras go to
  `<details>` (official fetch README) or a separate installation doc (wigolo).
- **Windows gotcha**: every npx-based repo carries the `cmd /c` wrapper note
  (servers root, firecrawl, browser-tools); the official fetch server has a
  whole Windows H2. Our SmartScreen + forward-slash notes are the binary
  equivalent — keep them inline where the path first appears.
- **Code-fence languages**: `json` for configs, `bash`/`sh` for commands,
  `yaml` for compose, `jsonc` for annotated responses (wigolo), unlabeled for
  sample output.
- **Headings**: sentence case dominates the tightest READMEs (official fetch
  server, browser-tools, fetcher-mcp); Title Case appears in firecrawl/wigolo;
  emoji headings only in the servers root README (institutional style) and
  wigolo's tables. Badges: 4 of 8 repos (wigolo, fzf, goreleaser, plus the
  fetch server's functional VS Code install buttons). GIFs: only wigolo.
  FAQ: only wigolo. Competitor comparison table: only wigolo.
- **License**: always last or near-last, always one line/paragraph + link,
  never the full text.
- **Length discipline**: the best onboarding-to-noise ratios are the shortest
  (browser-tools 288 lines, official fetch 318); 1,000+ line READMEs are
  tool-catalog-driven (firecrawl, fzf), not onboarding-driven.

## What high-star READMEs deliberately leave out

- A demo GIF or screenshot (6 of 8 have none; fzf uses one static PNG).
- Star badges on small/young projects (wigolo earns one at 5.5k★; fzf at
  83k★; browser-tools at 7.3k★ has none).
- Full license text, changelogs, per-client config for every conceivable
  client, and FAQ (except wigolo).
- The goreleaser lesson: do not delegate install to an external site unless
  one exists.

## Recommended skeleton for our README

Opinionated recommendation, grafting the best-fitting patterns onto what our
README already has (it is strong; this is restructuring plus three additions,
not a rewrite). Pattern provenance in brackets.

1. `# agent-web-fetch` + English | 简体中文 switcher (keep; fetcher-mcp shows
   i18n rows are idiomatic).
2. One-paragraph pitch **naming concrete client names**, with free/uncapped
   stated once, here (browser-tools opens this way).
3. Optional small badge row — release tag, platforms, license AGPL, CI; 3–4
   shields max, no star badge yet (fzf pattern; wigolo shows when a star
   badge becomes tasteful).
4. `## Why this instead of a paid web-reader` — our existing three bullets
   (free/uncapped, reliable launch, near-zero maintenance), retitled as an
   explicit instead-of section (browser-tools §"Why this instead of a
   CDP-based server"). This is where the "free replacement" positioning
   lives — not scattered.
5. `## Quick start` — three numbered steps (browser-tools pattern):
   1. **Install** — release-binary table first (zero prerequisites, our
      differentiator), then `go install
      github.com/ChHsiching/agent-web-fetch/cmd/agent-web-fetch@latest`
      (state the Go ≥ 1.26 requirement, browser-tools states its Node
      minimum), then build-from-source. One H3 per channel (fzf pattern).
   2. **Register** — ONE canonical stdio JSON block; the per-client variants
      (ZCode / Claude Code / Zed / opencode) collapse into `<details>`
      (official fetch README pattern) so the section reads in one screen.
      Keep the "you never launch it yourself" note as a one-liner — the
      servers root README's Transports section is the 91k-star precedent.
   3. **Use it** — the verify commands (status view, `claude mcp list`) plus
      a ready-to-paste prompt naming the server (browser-tools "### 3. Use
      it"; we already have this text — promote it into the numbered flow).
6. `## Usage` (new section) — example `fetch()` calls in fenced blocks:
   an HTML page → markdown; a `raw.githubusercontent.com/.../*.md` and a
   `.../package.json` fetch demonstrating verbatim passthrough; one showing
   `timeout` / `no_cache`. Show one real (abridged) response the way wigolo's
   "What your agent gets back" does. This is where the issue-11 raw-text
   feature becomes visible to a README skimmer.
7. `## Comparison` (new, small) — a 3–4 column table vs paid web-reader MCP
   tools with rows "API key: none/required", "Cost: $0/metered",
   "Runtime: single binary / npx + Node"; add the dated footnote wigolo uses.
   Keep to verifiable, stable facts (price claims rot; the footnote is the
   insurance).
8. `## What it does *not* do` (keep; browser-tools "Known limits" analog).
9. `## Troubleshooting` — fold the WEB_FETCH_LOG note, the
   built-in-fetch-tool conflict advice, and 2–3 FAQ entries in `<details>`
   form, including "Free? What's the catch?" and "Can I use this at work?"
   (wigolo FAQ pattern; our AGPL dual-license answer already exists in the
   License section — link it).
10. `## How it works` (keep).
11. `## License` + Contribution Terms last (universal convention; universal
    one-liner-plus-link shape — our extra commercial-license prose is
    justified by the dual licensing, as wigolo's is).

Estimated effect: install+config+first-call visible within the first screen
and a half; the per-client wall stops pushing Usage below the fold.

## Alternative patterns observed (if a different shape is wanted)

- **Pattern A — reference-minimal** (modelcontextprotocol/servers
  `src/fetch/` README, 91.7k★): no positioning, no badges, no images;
  Installation (one recommended method + one alternative) → Configuration
  with per-method `<details>` and per-client H3s → Customization → Windows →
  Debugging via MCP Inspector. Choose this if we want maximum sobriety and
  the smallest maintenance surface; it under-sells the free/uncapped angle.
- **Pattern B — marketing-complete** (KnockOutEZ/wigolo, 5.5k★): badge row,
  demo GIF, comparison table, benchmark section, FAQ, star history, sponsors.
  Appropriate once the project has traction assets (a real GIF, a benchmark);
  today it would be borrowed clothes — cherry-pick only the comparison table
  and FAQ entries.
- **Pattern C — tight narrative** (AgentDeskAI/browser-tools-mcp, 7.3k★):
  "Why this instead of X" → 3-step numbered install → Tools table → security
  → Known limits, zero badges, 288 lines. The closest match to our situation
  (single-purpose server with an opinionated stance), and the basis of the
  recommendation above.

## Verdict

Adopt Pattern C as the spine, borrowing three devices: `<details>`
collapsibles for per-client config from the official fetch server README, the
competitor comparison table (with dated footnote) and FAQ from wigolo, and
per-channel install H3s from fzf. The three genuinely missing pieces in our
current README are the `go install` path (with a stated Go version
requirement), a Usage section showing example `fetch()` calls including
raw-text passthrough of `.md`/`.json` URLs, and a compact comparison table
carrying the free/uncapped positioning.
