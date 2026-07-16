# Static HTTP + mature extraction library — no headless browser

The fetch path is plain HTTP plus a mature, stable content-extraction library
(Mozilla Readability — the same engine Firefox's Reader Mode uses). No
headless browser, no stealth, no JS rendering. Pages that require JavaScript
to render their content are honestly reported as unfetchable.

This supersedes the earlier lean toward a headless-browser route (which had
been chosen in problem 2 for JS coverage). It was reversed once "near-zero
maintenance / almost never needs updating" was elevated to a hard constraint:
a browser stack is the single largest source of ongoing maintenance
(Chromium ships every ~4 weeks, Playwright tracks it, anti-detection is an
arms race), so it is structurally incompatible with that constraint. Plain
HTTP and Readability have no such treadmill — protocols and a mature
extraction library barely change, so the tool can run for years untouched.

JS-rendered pages are the cost. That cost is accepted because the reader's
real workload — public docs, blogs, wikis, news, public repos — overwhelmingly
serves its article body in static HTML, where extraction just works.

**v1 scope, not a permanent exclusion.** JS rendering is deferred to a later
iteration, not ruled out forever. This ADR locks the v1 decision (static only)
for the sake of near-zero maintenance; if real-world usage shows JS-rendered
pages are a frequent gap, a future ADR may add an **optional** headless-browser
fallback — opt-in (e.g. user installs Playwright to enable it), off the default
path, so the zero-maintenance default stays intact. Do not read this ADR as
"JS will never be supported"; read it as "JS is out of scope for v1."
