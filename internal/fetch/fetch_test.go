package fetch

import (
	"context"
	"net/http"
	"testing"
)

// fakeHTTPClient is a stand-in for a real HTTP client so tests never reach the
// network. It records calls so tests can prove Fetch did not issue a request.
// Fetch does not call it during URL validation yet; the fetch path arrives in
// the next ticket, where this fake will be fleshed out.
type fakeHTTPClient struct {
	calls int
}

func (f *fakeHTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	f.calls++
	return &http.Response{StatusCode: 200}, nil
}

// TestFetch_RejectsMalformedURL asserts the core T1 behaviour: a malformed URL
// is rejected with a structured "invalid url" error AND no HTTP request is
// issued. The fake client's call count is the proof that no request went out.
func TestFetch_RejectsMalformedURL(t *testing.T) {
	client := &fakeHTTPClient{}
	params := Params{URL: "not a valid url"}

	result, err := Fetch(context.Background(), params, client)

	if err == nil {
		t.Fatalf("expected an error for malformed URL, got nil (result=%v)", result)
	}
	if !IsInvalidURLError(err) {
		t.Errorf("expected an InvalidURLError, got %v", err)
	}
	if client.calls != 0 {
		t.Errorf("expected no HTTP request for malformed URL, got %d call(s)", client.calls)
	}
}

// TestFetch_RejectsNonAbsoluteURL asserts that a URL missing a scheme or host
// is also rejected: a Web Reader needs an absolute target, not a bare path.
func TestFetch_RejectsNonAbsoluteURL(t *testing.T) {
	cases := []string{"example.com", "/some/path", "ftp://files.example.com/x"}
	for _, u := range cases {
		client := &fakeHTTPClient{}
		_, err := Fetch(context.Background(), Params{URL: u}, client)
		if !IsInvalidURLError(err) {
			t.Errorf("URL %q: expected InvalidURLError, got %v", u, err)
		}
		if client.calls != 0 {
			t.Errorf("URL %q: expected no request, got %d call(s)", u, client.calls)
		}
	}
}

// TestFetch_AcceptsValidURL asserts that a well-formed absolute URL passes
// validation and does not error. Retrieval itself is the next ticket's job;
// here we only assert the URL is accepted (no error) and the result is non-nil.
func TestFetch_AcceptsValidURL(t *testing.T) {
	client := &fakeHTTPClient{}
	params := Params{URL: "https://example.com/page"}

	result, err := Fetch(context.Background(), params, client)

	if err != nil {
		t.Fatalf("expected no error for valid URL %q, got %v", params.URL, err)
	}
	if result == nil {
		t.Fatal("expected a non-nil result for a valid URL")
	}
}
