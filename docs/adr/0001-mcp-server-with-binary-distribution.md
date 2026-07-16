# Ship as an MCP server, distributed as a prebuilt binary — never npx

We decided the tool is an MCP (Model Context Protocol) server, so it plugs
into Claude Code / ZCode / other MCP-capable agents as a native tool — the
same surface as their built-in fetch. Distribution is a single prebuilt
binary referenced by absolute local path in `mcp.json`, **not** `npx -y ...`.

The npx route was rejected because it is the root cause of the
`connect failed` / initialization-timeout failures the user has hit with
other MCP tools: cold start fetches from npm (network flakiness), node/npx
path issues on Windows, version drift, and slow startup that makes the agent
time out. A local binary has no network dependency at launch, carries its
own runtime, and is ready the instant the process spawns — which is what
makes the tool feel as dependable as a built-in.

Codex's MCP support is not yet confirmed; we ship for Claude Code + ZCode
first and let Codex follow when its MCP story is stable.
