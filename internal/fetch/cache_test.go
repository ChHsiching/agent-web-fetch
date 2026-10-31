package fetch

import (
	"testing"
	"time"
)

func mkParams(url string) Params {
	return Params{URL: url, ReturnFormat: "markdown"}
}

func countingMiss(calls *int) func(Params) (*Result, error) {
	return func(p Params) (*Result, error) {
		*calls++
		return &Result{Title: "T-" + p.URL, Content: "C-" + p.URL + "-" + p.ReturnFormat}, nil
	}
}

func TestCache_HitAvoidsSecondCall(t *testing.T) {
	calls := 0
	cache := newCache(time.Minute)

	r1, err := cache.get(mkParams("https://example.com/a"), false, countingMiss(&calls))
	if err != nil {
		t.Fatalf("first get: %v", err)
	}
	r2, err := cache.get(mkParams("https://example.com/a"), false, countingMiss(&calls))
	if err != nil {
		t.Fatalf("second get: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected miss handler called once, got %d", calls)
	}
	if r1.Content != r2.Content {
		t.Errorf("second call should return cached result %q, got %q", r1.Content, r2.Content)
	}
}

func TestCache_NoCacheForcesFresh(t *testing.T) {
	calls := 0
	cache := newCache(time.Minute)

	_, _ = cache.get(mkParams("https://example.com/a"), false, countingMiss(&calls))
	_, _ = cache.get(mkParams("https://example.com/a"), true, countingMiss(&calls))

	if calls != 2 {
		t.Errorf("expected 2 fetches with forceFresh bypass, got %d", calls)
	}
}

func TestCache_ExpiredEntryRefetches(t *testing.T) {
	calls := 0
	cache := newCache(5 * time.Millisecond)

	_, _ = cache.get(mkParams("https://example.com/a"), false, countingMiss(&calls))
	time.Sleep(20 * time.Millisecond)
	_, _ = cache.get(mkParams("https://example.com/a"), false, countingMiss(&calls))

	if calls != 2 {
		t.Errorf("expected 2 fetches after TTL expiry, got %d", calls)
	}
}

func TestCache_KeyDistinguishesFormat(t *testing.T) {
	calls := 0
	cache := newCache(time.Minute)

	pMD := mkParams("https://example.com/a")
	pTXT := mkParams("https://example.com/a")
	pTXT.ReturnFormat = "text"
	rMD, _ := cache.get(pMD, false, countingMiss(&calls))
	rTXT, _ := cache.get(pTXT, false, countingMiss(&calls))

	if calls != 2 {
		t.Errorf("expected 2 fetches for different formats, got %d", calls)
	}
	if rMD.Content == rTXT.Content {
		t.Errorf("markdown and text entries should differ; both = %q", rMD.Content)
	}
}
