package fetch

import (
	"sync"
	"time"
)

// DefaultCacheTTL is how long a cached result stays fresh. One hour per the
// spec — short enough that content churn on most pages is respected, long
// enough to skip redundant re-fetches within a working session.
const DefaultCacheTTL = time.Hour

// cacheEntry is one cached Result with its expiry time.
type cacheEntry struct {
	result  *Result
	expires time.Time
}

// cache is an in-memory TTL cache of fetch results, keyed by (url,
// return_format). It is safe for concurrent use. There is no persistence: a
// process restart clears it. Eviction is lazy (expired entries are overwritten
// on next write); no background sweeper, keeping the maintenance surface at
// zero per the spec.
//
// The cache holds no reference to any fetch strategy: the caller supplies the
// miss handler at get-time, so the same cache serves any client/timeout
// configuration.
type cache struct {
	ttl     time.Duration
	mu      sync.Mutex
	entries map[string]cacheEntry
}

// newCache builds an empty in-memory cache with the given TTL.
func newCache(ttl time.Duration) *cache {
	return &cache{
		ttl:     ttl,
		entries: make(map[string]cacheEntry),
	}
}

// get returns the cached Result for params, calling onMiss on a miss or when
// forceFresh is true. forceFresh bypasses a valid cached entry, forcing a
// fresh fetch (and refreshing the entry). onMiss is invoked outside the lock.
func (c *cache) get(params Params, forceFresh bool, onMiss func(Params) (*Result, error)) (*Result, error) {
	if !forceFresh {
		key := cacheKey(params)
		c.mu.Lock()
		if entry, ok := c.entries[key]; ok && time.Now().Before(entry.expires) {
			c.mu.Unlock()
			return entry.result, nil
		}
		c.mu.Unlock()
	}

	// Miss, expired, or forced: fetch outside the lock so concurrent gets for
	// other keys aren't serialized.
	result, err := onMiss(params)
	if err != nil {
		// Don't cache errors; a transient failure shouldn't shadow the next try.
		return nil, err
	}
	key := cacheKey(params)
	c.mu.Lock()
	c.entries[key] = cacheEntry{result: result, expires: time.Now().Add(c.ttl)}
	c.mu.Unlock()
	return result, nil
}

// cacheKey is the composite key distinguishing url and return_format. Same URL
// with a different format is a separate entry.
func cacheKey(params Params) string {
	format := params.ReturnFormat
	if format == "" {
		format = "markdown"
	}
	return format + "\x00" + params.URL
}
