# agent-web-fetch

**English** | [简体中文](./README.zh-CN.md)

[![GitHub release](https://img.shields.io/github/v/tag/ChHsiching/agent-web-fetch?include_prereleases&sort=semver)](https://github.com/ChHsiching/agent-web-fetch/releases/latest)
[![License](https://img.shields.io/github/license/ChHsiching/agent-web-fetch)](https://github.com/ChHsiching/agent-web-fetch/blob/main/LICENSE)

A **free, uncapped web reader** for any agent client that speaks the Model
Context Protocol, including Claude Code and ZCode. It takes a URL the agent
already holds and returns the page's main content as clean, model-friendly
markdown. It is a drop-in replacement for the paid, rate-limited fetch tools
built into many agent clients.

## Why this instead of a paid web-reader

- **Free and uncapped.** Runs on your machine over plain HTTP. No API key, no
  quota, no monthly limit, no 429s.
- **Reliable launch.** Ships as a single statically-linked binary. No `npx`,
  no runtime to install, no network at startup. It connects the first time,
  every time.
- **Near-zero maintenance.** Static HTTP fetch plus the Readability extraction
  algorithm, the same engine Firefox Reader View uses. No headless browser
  treadmill, no anti-detection arms race to keep up with.

Hosted web-reader services move your URLs and page content through a vendor
cloud and meter every request. This tool does the same job on your machine,
in a process your MCP client already manages, for $0 per page.

## What it does

Give it a URL, get back the page's main article as markdown:

```
fetch({ "url": "https://example.com" })
  → "# Example Domain\n\nThis domain is for use in documentation examples..."
```

It runs Readability-style extraction to isolate the main article and strip
boilerplate (nav, footer, scripts). When a page has no clear main article
(list pages, dashboards), it falls back to the full document so you never get
empty output. Image URLs are preserved in the markdown.

URLs that serve raw text documents skip extraction entirely. This covers all
`text/*` types (markdown, plain text, CSV, ...), JSON (including `+json`
vendor types), XML, YAML, and JavaScript. The body is decoded to UTF-8 and
returned verbatim, with no wrapping or reformatting. For these raw files both
`return_format` values return identical content. The document already is its
own model-friendly form. Binary and other non-text types (PDF, images,
audio/video) are still rejected with a clear error.

### Tool parameters

| Parameter       | Required | Default    | Description                                                      |
| --------------- | -------- | ---------- | ---------------------------------------------------------------- |
| `url`           | yes      | none       | The URL to fetch. Must be an absolute `http`/`https` URL.        |
| `timeout`       | no       | `30s`      | Per-request timeout, as a Go duration string (e.g. `45s`).       |
| `return_format` | no       | `markdown` | `markdown` or `text`.                                            |
| `no_cache`      | no       | `false`    | Bypass the in-memory cache and force a fresh fetch.              |
| `max_mb`        | no       | `2`        | Response size cap in MB, from 1 to 8. Oversized responses are rejected with an error naming the size and the limit; never truncated. |

### What it does *not* do

- **No web search / discovery.** It fetches URLs you give it; it doesn't find
  pages.
- **No JavaScript rendering.** Pages that need JS to produce content return
  sparse results. This is a deliberate v1 trade-off for near-zero maintenance.
- **No summarization / translation / image description.** It returns content,
  it doesn't process it. No paid model calls.
- **No authenticated content.** It fetches anonymously; login-walled pages
  are out of scope.

## Quick start

### 1. Install the binary for your platform

Grab the right file from the latest release:

| Platform          | File                                  |
| ----------------- | ------------------------------------- |
| Windows           | `agent-web-fetch-windows-amd64.exe`   |
| macOS (Apple Silicon) | `agent-web-fetch-darwin-arm64`    |
| macOS (Intel)     | `agent-web-fetch-darwin-amd64`        |
| Linux             | `agent-web-fetch-linux-amd64`         |

No installer and no runtime to install. You do not need Node, Python, or Go.

**Where to put the file:** a recommended per-user location exists on each
platform. It needs no admin rights and is the conventional spot for
user-installed programs:

| Platform | Recommended location |
| -------- | -------------------- |
| Windows | `%LOCALAPPDATA%\Programs\agent-web-fetch\agent-web-fetch.exe` |
| macOS / Linux | `~/.local/bin/agent-web-fetch` |

That said, MCP doesn't actually care where the file lives. It launches the
binary via the absolute path in your config, so you can put it anywhere you
have read/execute permission. You do not need to add it to your `PATH`. Just
don't drop it into other users' directories or system folders that need admin
rights. The examples below use the recommended locations; replace the path if
you put it elsewhere.

<details>
<summary>Prefer <code>go install</code> (requires Go 1.26+)</summary>

```bash
go install github.com/ChHsiching/agent-web-fetch/cmd/agent-web-fetch@latest
```

The binary lands in `$GOBIN`, or `$GOPATH/bin` if `$GOBIN` is unset. On
Windows that is typically `%USERPROFILE%\go\bin`. Point your MCP config at
that absolute path.
</details>

<details>
<summary>Build from source</summary>

Requires Go 1.26+ (matches `go.mod`).

```bash
# Build all four platform binaries into dist/
./build.sh          # any bash (Git Bash, WSL, Linux, macOS)
# or, on unix with make:
make release
```

Each binary is statically linked (`CGO_ENABLED=0`) with no external runtime
dependency.
</details>

### 2. Register it with your MCP client

This is a standard **stdio MCP server**: it has no arguments and no
environment requirements. **You never launch it yourself.** Your MCP client
starts and manages the process based on the config entry below. Running the
binary by hand does nothing for the agent: it just sits there waiting for MCP
messages on stdin. This is how every stdio MCP server works; see the [official
transports note](https://github.com/modelcontextprotocol/servers#transports).

In every MCP client the config entry is the same idea. Point `command` at
the binary's absolute path and leave `args` empty:

```json
"chhsich-web-fetch": {
  "type": "stdio",
  "command": "/absolute/path/to/agent-web-fetch",
  "args": []
}
```

What differs between clients is **where** this entry goes, the exact key
names, and occasionally the shape of the entry. Concrete examples for the
common ones:

<details>
<summary><strong>ZCode:</strong> flat object keyed by server name, no outer wrapper</summary>

```json
{
  "chhsich-web-fetch": {
    "type": "stdio",
    "command": "C:/Users/<username>/AppData/Local/Programs/agent-web-fetch/agent-web-fetch.exe",
    "args": []
  }
}
```
</details>

<details>
<summary><strong>Claude Code:</strong> <code>~/.claude.json</code> under <code>mcpServers</code>; on Windows <code>%USERPROFILE%\.claude.json</code></summary>

```json
{
  "mcpServers": {
    "chhsich-web-fetch": {
      "type": "stdio",
      "command": "C:/Users/<username>/AppData/Local/Programs/agent-web-fetch/agent-web-fetch.exe",
      "args": []
    }
  }
}
```

Or via the CLI (does the same thing): `claude mcp add chhsich-web-fetch "C:/Users/<username>/AppData/Local/Programs/agent-web-fetch/agent-web-fetch.exe"`
</details>

<details>
<summary><strong>Zed:</strong> <code>context_servers</code> key, no <code>type</code> field</summary>

Easiest through the UI: **Settings → AI → MCP Servers → Add Server → Add Local Server**, pointing it at the exe. Or edit the settings file. Open it via the command palette with `zed: open settings file`; on Windows it lives at `C:\Users\<username>\AppData\Roaming\Zed\settings.json`.

```json
{
  "context_servers": {
    "chhsich-web-fetch": {
      "command": "C:/Users/<username>/AppData/Local/Programs/agent-web-fetch/agent-web-fetch.exe",
      "args": []
    }
  }
}
```
</details>

<details>
<summary><strong>opencode:</strong> <code>mcp</code> key, <code>"type": "local"</code>, and <code>command</code> is an array</summary>

Config lives in `opencode.json` in the project root, or `~/.config/opencode/opencode.json` globally. On Windows that is `%USERPROFILE%\.config\opencode\opencode.json`.

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "chhsich-web-fetch": {
      "type": "local",
      "command": ["C:/Users/<username>/AppData/Local/Programs/agent-web-fetch/agent-web-fetch.exe"]
    }
  }
}
```
</details>

**Any other stdio MCP client.** Find where it keeps its MCP server list,
whether a JSON/YAML config or a settings UI, and add one entry: type `stdio`,
`command` = absolute path to the binary, `args` = `[]`. That's the whole
contract. There are no other parameters to set.

> **Replace the path:** the examples above use the recommended install location
> with `<username>` as a placeholder. Swap in your actual username, or use
> `%LOCALAPPDATA%` if your client expands env vars. Adjust the path if you put
> the binary somewhere else.

> **Naming:** the key, `chhsich-web-fetch` above, is your client-side label
> for the server. Call it whatever you want. The tool it exposes is named
> `fetch`, so the model calls `fetch(...)`. Two servers exposing a tool both
> named `fetch` don't conflict as long as the server keys differ. The server
> name is the namespace.

> **Path tip (Windows):** use the full absolute path including `.exe`.
> Forward slashes work in JSON and avoid backslash escaping.

> **Windows SmartScreen note:** the release binary is unsigned, so Windows may
> show a "Windows protected your PC" prompt the first time it runs. Click
> **More info → Run anyway**. This is expected for unsigned binaries and only
> happens once.

Restart your client after editing the config. The `fetch` tool now appears
alongside the built-in tools and the model can call it like any other.

### 3. Use it

First confirm the server actually registered. Every client has a status view:

- **Zed**: Settings → AI → MCP Servers; the dot next to `chhsich-web-fetch`
  should be green ("Server is active").
- **opencode**: run `opencode mcp list` in a terminal.
- **Claude Code**: run `claude mcp list`.

Then ask the model to fetch any public page, naming the server so it picks
this tool over any built-in one:

> Use the chhsich-web-fetch fetch tool to read https://example.com

You should get back the page's content as markdown.

**If the agent keeps using its own built-in fetch:** Zed and opencode both
ship their own web-fetch tool, and registering this server does *not* disable
it. The model may simply keep picking the built-in one, which looks exactly
like "the tool isn't working" while the registration is in fact fine. Two
ways to make it use this one:

- Name the server in the prompt (as above), or add a line to your
  `AGENTS.md` / rules file: *"When you need to read a web page, use the
  `chhsich-web-fetch` fetch tool."*
- Or turn off the built-in tool. In Zed, create an [agent
  profile](https://zed.dev/docs/ai/agent-profiles) where the built-in
  `fetch` is off and only this server's tools are on. In opencode, disable
  it via the `tools` key in `opencode.json`, e.g. `"tools": { "webfetch": false }`.

If the server never shows up or calls fail, see [Troubleshooting](#troubleshooting).

## Usage

Raw text files come back byte-for-byte. This is the tool reading its own
README from `raw.githubusercontent.com`:

```
fetch({ "url": "https://raw.githubusercontent.com/ChHsiching/agent-web-fetch/main/README.md" })
  → "# agent-web-fetch\n\n**English** | [简体中文](./README.zh-CN.md)\n\nA **free, uncapped web reader*..."
```

The full markdown source comes back, with no code fences added and no
extraction. JSON works the same way. `package.json`, API responses, GeoJSON
all pass through verbatim.

Regular pages go through extraction. A long article comes back as its main
body in markdown with nav, footer, and scripts stripped. Parameters when you
need them:

```
fetch({ "url": "https://example.com", "timeout": "45s", "no_cache": true })
```

Phrases in your prompt map to parameters, so you rarely need to spell them
out:

- *"use chhsich-web-fetch to read ..."* → the model picks this server's tool
  over any built-in one.
- *"re-fetch it, don't use cache"* → the model sets `no_cache: true`.
- *"the site is slow"* → a larger `timeout`.

## Comparison

| | agent-web-fetch | hosted web-reader MCPs |
| --- | --- | --- |
| API key / account | none | required (free tiers exist) |
| Cost per page | $0, uncapped | metered credits |
| Where content is processed | your machine | vendor cloud |
| Runtime | one static binary, no deps | `npx` + Node, or a remote URL |
| JavaScript rendering | no | some offer it |
| License | AGPL-3.0, with commercial option | varies (AGPL to closed) |

Feature standing as of October 2026. Check each service's current pricing.
"Hosted web-reader MCPs" means tools of the Firecrawl / Jina Reader shape.

## Troubleshooting

**The server never appears, or calls fail.** By default the server is silent
(stdout is the MCP transport and must stay clean). Set the `WEB_FETCH_LOG`
environment variable to a file path in the server's config entry, e.g.
`"env": { "WEB_FETCH_LOG": "C:/Users/<username>/awf.log" }` for clients that
support per-server env. Restart the client, reproduce the call, and read the
log.

<details>
<summary>Free? What's the catch?</summary>

No catch in the pricing sense: there is no server to pay for because the
binary runs on your machine. The trade-offs are the ones listed under
[What it does *not* do](#what-it-does-not-do): no JavaScript rendering, no
authenticated pages. The license is AGPL-3.0, which is free for use,
modification, and distribution under its terms.
</details>

<details>
<summary>Can I use this at work?</summary>

Yes. Internal commercial use is free under AGPL-3.0. Two obligations kick in
only if you distribute the binary or offer it as a network service: those
derivative works must be open-sourced under AGPL-3.0. Closed-source
commercial distribution needs a separate license; see
[License](#license).
</details>

## How it works

```
URL → validate (http/https, absolute) → HTTP GET (realistic browser headers)
   → Readability extraction → markdown → in-memory cache (1h TTL)
                                     ↘ full-document fallback (never empty)
   ↘ raw text (text/*, JSON, XML, YAML, JS) → verbatim passthrough (UTF-8)
```

The fetch pipeline is a single deep module (`internal/fetch`), exposed over
stdio as one MCP tool (`internal/mcpserver`). Every failure (timeout, non-2xx,
bad content type, even a recovered panic) comes back as a structured error the
model can read. The server process never crashes.

See `CONTEXT.md` for the project glossary and `docs/adr/` for the architectural
decisions (MCP + binary distribution, anonymous fetching, static extraction,
Go, launch reliability).

## Contributing

Issues and PRs are welcome on
[GitHub](https://github.com/ChHsiching/agent-web-fetch). Bug reports are most
useful with the URL you fetched, what you expected, and what you got back.

Local development:

```bash
go test ./...     # test suite
go vet ./...      # static checks
./build.sh        # build all platform binaries into dist/
```

If you're an AI agent (or a human) orienting in the repo: `CONTEXT.md` holds
the project glossary, `docs/adr/` the numbered architectural decisions, and
`AGENTS.md` the conventions for working on issues in this repository.

By submitting a PR you agree to the [contribution terms](#contribution-terms)
below.

## License

AGPL-3.0-only · Copyright (c) 2026 ChHsiching. See [LICENSE](LICENSE).

- Use (including internal commercial use), modification, and distribution are free. Distributing it or offering it as a network service requires derivative works to be open-sourced under AGPL-3.0.
- Closed-source commercial use requires a separate commercial license: hsichingchang@gmail.com

### Contribution Terms

By submitting a PR, you agree to license your contribution under AGPL-3.0 and grant the maintainer the right to offer separate commercial licenses. Your contribution remains available to everyone under AGPL.
