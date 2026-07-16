# Implementation language: Go

The server is written in Go and distributed as a true native binary per ADR-0001.

Go wins on the project's hard constraints. It compiles to a single
statically-linked binary with no runtime, no VM, no interpreter — the MCP
server process is ready the instant it spawns, which is what makes
"initialization never fails / never connect-failed" achievable rather than
aspirational. There is no node/python environment to be missing on the target
machine, no dependency tree to drift, no `npx` cold start. The toolchain and
all libraries are open source (free, uncapped), and cross-compiling for
Windows/Linux/macOS is a one-liner, so the ADR-0001 binary-distribution model
is honored cleanly.

TypeScript + Bun (`--compile`) was the considered alternative. It would have
given us Mozilla's original Readability.js (the canonical implementation,
versus a Go community port) and the official MCP TypeScript SDK. That route
was rejected because launch-time stability — the user's top-weighted
requirement, named three times — is asymmetrically at risk there: Bun's Windows
support is less mature than Linux/macOS, and bundling a ~90MB runtime into a
resident MCP server introduces a new failure surface precisely on the
"never connect-failed" axis the user cares about most. That weakness has no
fallback — a packaging or runtime edge shows up as the exact failure mode the
user is trying to eliminate.

Go's weakness, by contrast, is containable: a community Readability port may
extract slightly worse than the original on some pages, but the problem-6
fallback (full-document markdown when extraction fails) degrades gracefully
rather than breaking the server. In other words: Go's weak axis has a fallback;
TS/Bun's weak axis doesn't, and it's on the axis the user weights highest.
The bounded one-time cost of Go's thinner extraction ecosystem is accepted —
it does not violate the "near-zero maintenance" constraint.

## Non-negotiable: launch-time stability

The recurring `connect failed` / initialization failures seen with other MCP
tools are treated as the top acceptance criterion. This means, concretely:

- **No network at launch.** The binary does nothing online at startup — no
  package fetch (we're not `npx`), no telemetry call home, no update check.
- **No external runtime.** Statically linked; the only thing it needs from the
  host is the OS.
- **Fast, deterministic init.** The MCP handshake completes in milliseconds,
  well under any client timeout, because there is nothing to download or warm
  up.
- **Fail safe, never hang.** If something is unavailable at runtime (a target
  site is down), that failure must surface as a tool-error returned to the
  agent, never as a hung server or a broken initialization.
