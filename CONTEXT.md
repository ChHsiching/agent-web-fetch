# agent-web-fetch

A web reader / fetch tool for MCP-capable agents (Claude Code, ZCode, …) —
takes a URL the agent already knows and returns model-friendly content. A
drop-in replacement for paid built-in fetch tools.

## Language

**Web Reader**:
A tool that takes a URL and returns the content at that URL, parsed into
model-friendly form. The agent already has the link — the reader's job is to
fetch and extract, not to discover.
_Avoid_: Web Search

**Web Search**:
A tool that takes a query and returns a list of results/links. Discovery, not
retrieval. Out of scope for this project.
_Avoid_: Web Reader (when you mean discovery)

**Extraction**:
The step that turns fetched HTML into model-friendly content (markdown/text)
by stripping boilerplate and isolating the main article — e.g. via a
Readability-style algorithm. Part of the reader's job.
_Avoid_: Parsing (too low-level), scraping

**Processing**:
Any transformation *beyond* fetch + extract — summaries, translations, image
descriptions, link digests, reformatting. Explicitly **out of scope**: the
reader returns retrieved content, it does not generate or embellish. This
boundary exists because any processing would pull in a model dependency (cost)
or extra deployment surface (maintenance), violating the free / near-zero
maintenance principles.
_Avoid_: Enhancement, enrichment, summarization
