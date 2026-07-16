# Launch reliability is non-negotiable

The recurring MCP `connect failed` / initialization failures the user has
experienced with other tools are the project's top acceptance criterion —
named unprompted four times across the grilling. This ADR consolidates the
defenses already established by ADR-0001 through ADR-0004 and the
problem-8/9 decisions into one checklist. Every known cause of connection
failure has an explicit mitigation:

| # | Failure source | Mitigation (already decided) |
| - | -------------- | ---------------------------- |
| 1 | `npx` cold-start package fetch fails/times out | ADR-0001: binary distribution, never `npx`. Zero network at launch. |
| 2 | Target machine missing a runtime (node/python) | ADR-0004: Go statically-linked true binary, no runtime dependency. |
| 3 | Bun/Windows runtime edge cases | ADR-0004: rejected TS/Bun in favor of Go (first-class Windows support). |
| 4 | Launch-time network call (telemetry/update check/download) | ADR-0004: "No network at launch" — the binary does nothing online at startup. |
| 5 | Slow init trips the client's handshake timeout | ADR-0004: millisecond MCP handshake, nothing to download or warm up. |
| 6 | Runtime panic crashes the process | Problem 8: `recover` converts panics to tool-errors; the process never dies. A crash-restart is itself a connection-failure risk, so crashes are designed out. |
| 7 | Slow target site hangs the server | Problem 8: bounded by `timeout` (default 30s); always returns, never hangs. |
| 8 | stdout polluted by logs corrupts the stdio protocol | Problem 9: silent by default; logs never touch stdout (file only). |

This is an invariant of the project, not a feature. Any future change that
weakens any row above is a regression against the user's core requirement and
must be rejected.
