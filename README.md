# agent-web-fetch

A **free, uncapped web reader** for MCP-capable agents (Claude Code, ZCode, and
any client that speaks the Model Context Protocol). It takes a URL the agent
already holds and returns the page's main content as clean, model-friendly
markdown — a drop-in replacement for the paid, rate-limited fetch tools built
into many agent clients.

- **Free and uncapped** — runs on your machine over plain HTTP. No API key, no
  quota, no monthly limit, no 429s.
- **Reliable launch** — ships as a single statically-linked binary. No `npx`,
  no runtime to install, no network at startup. It connects the first time,
  every time.
- **Near-zero maintenance** — static HTTP fetch + the Readability extraction
  algorithm (the same engine Firefox Reader View uses). No headless browser
  treadmill, no anti-detection arms race to keep up with.

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

### Tool parameters

| Parameter       | Required | Default   | Description                                                      |
| --------------- | -------- | --------- | ---------------------------------------------------------------- |
| `url`           | yes      | —         | The URL to fetch. Must be an absolute `http`/`https` URL.        |
| `timeout`       | no       | `30s`     | Per-request timeout, as a Go duration string (e.g. `45s`).       |
| `return_format` | no       | `markdown`| `markdown` or `text`.                                            |
| `no_cache`      | no       | `false`   | Bypass the in-memory cache and force a fresh fetch.              |

### What it does *not* do

- **No web search / discovery** — it fetches URLs you give it; it doesn't find
  pages.
- **No JavaScript rendering** — pages that need JS to produce content return
  sparse results. (This is a deliberate v1 trade-off for near-zero maintenance.)
- **No summarization / translation / image description** — it returns content,
  it doesn't process it. No paid model calls.
- **No authenticated content** — it fetches anonymously; login-walled pages are
  out of scope.

## Install

### 1. Download the binary for your platform

Grab the right file from the latest release and put it anywhere on your machine
(for example `~/bin/` or `C:\Users\you\bin\`):

| Platform          | File                                  |
| ----------------- | ------------------------------------- |
| Windows           | `agent-web-fetch-windows-amd64.exe`   |
| macOS (Apple Silicon) | `agent-web-fetch-darwin-arm64`    |
| macOS (Intel)     | `agent-web-fetch-darwin-amd64`        |
| Linux             | `agent-web-fetch-linux-amd64`         |

No installer, no runtime to install (no Node, Python, or Go required).

### 2. Register it with your MCP client

Add an entry to your client's MCP config pointing at the binary's **absolute
path**.

**Claude Code** (`mcp.json`):

```json
{
  "mcpServers": {
    "web-fetch": {
      "command": "/absolute/path/to/agent-web-fetch"
    }
  }
}
```

**ZCode** and other MCP clients use the same shape — set `command` to the
absolute path of the binary.

On Windows, use the full path including the `.exe` extension and escaped
backslashes (or forward slashes, which also work):

```json
{
  "mcpServers": {
    "web-fetch": {
      "command": "C:\\Users\\you\\bin\\agent-web-fetch.exe"
    }
  }
}
```

Restart your client. The `fetch` tool now appears alongside the built-in tools
and the model can call it like any other.

### 3. (Optional) Enable file logging

By default the server is silent (stdout is the MCP transport and must stay
clean). To write diagnostics to a file for troubleshooting:

Set the `WEB_FETCH_LOG` environment variable to a file path. Logs append there
and never touch stdout.

## Build from source

Requires Go 1.22+.

```bash
# Build all four platform binaries into dist/
./build.sh          # any bash (Git Bash, WSL, Linux, macOS)
# or, on unix with make:
make release
```

Each binary is statically linked (`CGO_ENABLED=0`) with no external runtime
dependency. Run the test suite with `go test ./...`.

## How it works

```
URL → validate (http/https, absolute) → HTTP GET (realistic browser headers)
   → Readability extraction → markdown → in-memory cache (1h TTL)
                                     ↘ full-document fallback (never empty)
```

The fetch pipeline is a single deep module (`internal/fetch`), exposed over
stdio as one MCP tool (`internal/mcpserver`). Every failure (timeout, non-2xx,
bad content type, even a recovered panic) comes back as a structured error the
model can read — the server process never crashes.

See `CONTEXT.md` for the project glossary and `docs/adr/` for the architectural
decisions (MCP + binary distribution, anonymous fetching, static extraction,
Go, launch reliability).
