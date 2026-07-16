# Anonymous + stealth identity — no login reuse

The reader runs an anonymous browser session hardened against bot detection
(spoofed UA, realistic headers, stealth plugin). It does **not** attach to the
user's daily browser, does **not** reuse their login cookies, and does **not**
ship a cookie-import subsystem in v1. Pages behind a login wall are
honestly reported as unfetchable.

This fits the reader's actual workload. A web reader takes URLs the agent
already holds and retrieves their content — and the overwhelming majority of
those URLs are public (docs, blogs, wikis, news, public repos). Building an
identity-reuse system around a tool that rarely needs authentication would be
over-engineering, and binding to the user's daily browser (model A) would
penalize non-Chrome users for a benefit most fetches don't need. Login-walled
content is the exception, not the main case; if it becomes common later, a
lightweight optional cookie-injection hook can be added without re-architecting.

Stealth here is not for bypassing authorization — it's for reaching *public*
content reliably, since many sites return 403 or a challenge page to a default
headless client even when the content is openly accessible.
