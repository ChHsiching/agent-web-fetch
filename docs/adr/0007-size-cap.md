# Size cap: reject, never truncate (2 MB default, 8 MB ceiling)

The fetch path enforces a size cap on every response body, on both the
Extraction and Passthrough paths (ADR-0006). When Content-Length is present
and over the cap, the response is rejected before a single body byte is read;
otherwise a bounded read (cap plus one byte) rejects bodies that turn out to
be larger. An over-limit response returns a structured error stating the
document's actual size and the active limit, so the agent can decide whether
to raise the cap or narrow the fetch.

The default cap is 2 MB. The optional `max_mb` parameter raises it for
large-context consumers, with a hard ceiling of 8; values above the ceiling
are rejected with an error naming it. `max_mb` participates in the cache key
alongside url and return_format, so a document admitted under a raised cap is
never served to a default-cap fetch of the same URL.

The decision is rejection, never truncation. A truncated body lies to the
agent: it looks like the whole file, so the agent reasons over partial data
without knowing. A rejection with the actual size and the limit tells the
truth and hands the agent the two real options: raise `max_mb`, or narrow the
fetch.

The values come from current model context sizes. A 200k-token context cannot
usefully consume more than roughly 2 MB of text, so a larger default only buys
context flooding. A 1M-token research context needs about 4 MB, so the opt-out
exists, and 8 MB covers it with headroom; beyond that the right move is a
narrower fetch, not a bigger body.
