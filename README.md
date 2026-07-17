# agent-web-fetch

**English** | [简体中文](./README.zh-CN.md)

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

Grab the right file from the latest release:

| Platform          | File                                  |
| ----------------- | ------------------------------------- |
| Windows           | `agent-web-fetch-windows-amd64.exe`   |
| macOS (Apple Silicon) | `agent-web-fetch-darwin-arm64`    |
| macOS (Intel)     | `agent-web-fetch-darwin-amd64`        |
| Linux             | `agent-web-fetch-linux-amd64`         |

No installer, no runtime to install (no Node, Python, or Go required).

**Where to put the file:** anywhere you like — there is no required location and
no need to add it to your `PATH` (MCP launches it via the absolute path you put
in the config). The only constraints are that it stays where you put it (the
config references that path) and that your user has read/execute permission on
it. So don't drop it in another user's directory or a system folder that needs
admin rights. Any folder you own — a documents folder, a dedicated tools
folder, your home directory — works fine.



### 2. Register it with your MCP client

This is a standard **stdio MCP server**: it has no arguments and no environment
requirements. In every MCP client the config entry is the same idea — point
`command` at the binary's absolute path, leave `args` empty:

```json
"chhsich-web-fetch": {
  "type": "stdio",
  "command": "/absolute/path/to/agent-web-fetch",
  "args": []
}
```

What differs between clients is only **where** this entry goes and the exact
key names. Concrete examples for the common ones:

**ZCode** — add the entry to its MCP servers config (a flat object keyed by
server name, no outer wrapper):

```json
{
  "chhsich-web-fetch": {
    "type": "stdio",
    "command": "C:/Users/you/bin/agent-web-fetch.exe",
    "args": []
  }
}
```

**Claude Code** — `~/.claude.json` (or `%USERPROFILE%\.claude.json` on Windows),
where servers live under a `mcpServers` key:

```json
{
  "mcpServers": {
    "chhsich-web-fetch": {
      "type": "stdio",
      "command": "C:/Users/you/bin/agent-web-fetch.exe",
      "args": []
    }
  }
}
```

Or via the CLI (does the same thing): `claude mcp add chhsich-web-fetch "C:/Users/you/bin/agent-web-fetch.exe"`

**Any other stdio MCP client** — find where it keeps its MCP server list (a
JSON/YAML config, a settings UI, etc.) and add one entry: type `stdio`,
`command` = absolute path to the binary, `args` = `[]`. That's the whole
contract — there are no other parameters to set.

> **Naming:** the key (`chhsich-web-fetch` above) is your client-side label
> for the server — call it whatever you want. The tool it exposes is named
> `fetch`, so the model calls `fetch(...)`. Two servers exposing a tool both
> named `fetch` don't conflict as long as the server keys differ (the server
> name is the namespace).

> **Path tip (Windows):** use the full absolute path including `.exe`.
> Forward slashes work in JSON and avoid backslash escaping
> (`"C:/Users/you/bin/agent-web-fetch.exe"`).

> **Windows SmartScreen note:** the release binary is unsigned, so Windows may
> show a "Windows protected your PC" prompt the first time it runs. Click
> **More info → Run anyway**. This is expected for unsigned binaries and only
> happens once.

Restart your client after editing the config. The `fetch` tool now appears
alongside the built-in tools and the model can call it like any other.

### 3. Verify it works

After restarting your client, ask the model to fetch any public page, e.g.:

> Use the fetch tool to read https://example.com

You should get back the page's content as markdown. If nothing comes back or
the tool is missing, set `WEB_FETCH_LOG` (below) and check the log file.

### 4. (Optional) Enable file logging

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
