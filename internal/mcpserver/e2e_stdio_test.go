//go:build e2e

package mcpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var e2eBinary string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "awf-e2e-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	// ".exe" unconditionally: it is just a filename, and exec.Command runs it
	// on every platform. go test runs with cwd = this package dir.
	e2eBinary = filepath.Join(tmp, "agent-web-fetch-e2e.exe")
	build := exec.Command("go", "build", "-o", e2eBinary, "../../cmd/agent-web-fetch")
	if out, err := build.CombinedOutput(); err != nil {
		panic("e2e: go build failed: " + string(out))
	}
	os.Exit(m.Run())
}

// e2eSession starts the built binary as a subprocess over stdio and connects a
// real MCP client to it. Closing the session closes stdin, which is the
// server's normal shutdown path (see Run).
func e2eSession(t *testing.T) *mcp.ClientSession {
	t.Helper()
	transport := &mcp.CommandTransport{Command: exec.Command(e2eBinary)}
	client := mcp.NewClient(&mcp.Implementation{Name: "e2e-client", Version: "v0.0.0"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("e2e: connect to subprocess server: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func e2eCall(t *testing.T, session *mcp.ClientSession, args map[string]any) (text string, isError bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "fetch", Arguments: args})
	if err != nil {
		t.Fatalf("e2e: CallTool(%v) protocol error: %v", args, err)
	}
	if len(res.Content) == 0 {
		t.Fatalf("e2e: CallTool(%v) returned no content", args)
	}
	tc, _ := res.Content[0].(*mcp.TextContent)
	if tc == nil {
		t.Fatalf("e2e: CallTool(%v) returned non-text content: %v", args, res.Content)
	}
	return tc.Text, res.IsError
}

func newE2EServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return ts
}

// TestE2E_RawTextPassedThroughVerbatim is the heart of issue #11: a text/plain
// document (e.g. raw.githubusercontent.com markdown) must come back byte-for-
// byte — no code fences, no prefix, no extraction.
func TestE2E_RawTextPassedThroughVerbatim(t *testing.T) {
	const body = "# Title\n\nSome *markdown* with `code`.\n\n- a\n- b\n"
	ts := newE2EServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}))
	session := e2eSession(t)

	got, isError := e2eCall(t, session, map[string]any{"url": ts.URL + "/readme.md"})
	if isError {
		t.Fatalf("raw text fetch failed: %s", got)
	}
	if got != body {
		t.Errorf("raw text not verbatim:\n got: %q\nwant: %q", got, body)
	}
	if strings.Contains(got, "```") {
		t.Errorf("raw text was wrapped in code fences: %q", got)
	}

	gotText, isError := e2eCall(t, session, map[string]any{"url": ts.URL + "/readme.md?fmt=text", "return_format": "text"})
	if isError {
		t.Fatalf("raw text fetch with return_format=text failed: %s", gotText)
	}
	if gotText != body {
		t.Errorf("return_format=text differs for raw file:\n got: %q\nwant: %q", gotText, body)
	}
}

func TestE2E_JSONPassedThroughVerbatim(t *testing.T) {
	const body = `{"name": "agent-web-fetch", "issues": [11, 12, 13, 14]}` + "\n"
	ts := newE2EServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	session := e2eSession(t)

	got, isError := e2eCall(t, session, map[string]any{"url": ts.URL + "/data.json"})
	if isError {
		t.Fatalf("json fetch failed: %s", got)
	}
	if got != body {
		t.Errorf("json not verbatim:\n got: %q\nwant: %q", got, body)
	}
}

func TestE2E_CharsetDecodedToUTF8(t *testing.T) {
	ts := newE2EServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=iso-8859-1")
		_, _ = w.Write([]byte("caf\xe9 menu\r\nsecond line\r\n"))
	}))
	session := e2eSession(t)

	got, isError := e2eCall(t, session, map[string]any{"url": ts.URL + "/menu.txt"})
	if isError {
		t.Fatalf("charset fetch failed: %s", got)
	}
	const want = "café menu\r\nsecond line\r\n"
	if got != want {
		t.Errorf("charset not decoded to UTF-8 verbatim:\n got: %q\nwant: %q", got, want)
	}
}

func TestE2E_HTMLStillExtracted(t *testing.T) {
	ts := newE2EServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><head><title>Old</title></head><body><article><h1>Real Article</h1><p>The article body is long enough to clear the readability extraction threshold used by the fetch module, so extraction wins over any fallback path.</p></article></body></html>`))
	}))
	session := e2eSession(t)

	got, isError := e2eCall(t, session, map[string]any{"url": ts.URL + "/page.html"})
	if isError {
		t.Fatalf("html fetch failed: %s", got)
	}
	if !strings.Contains(got, "Real Article") {
		t.Errorf("extracted markdown missing article text: %q", got)
	}
	if strings.Contains(got, "<html>") {
		t.Errorf("html came back verbatim instead of extracted: %q", got)
	}
}

func TestE2E_HTMLTextModeReturnsPlainText(t *testing.T) {
	const page = `<html><head><title>Plain Mode</title></head><body><article><h1>Plain Mode Heading</h1><p>The article body is long enough to clear the readability extraction threshold used by the fetch module, and it carries <strong>bold words</strong> plus <a href="https://example.com/x">a reference link</a> so markdown markers would show if any leaked.</p></article></body></html>`
	ts := newE2EServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
	}))
	session := e2eSession(t)

	gotText, isError := e2eCall(t, session, map[string]any{"url": ts.URL + "/mode.html", "return_format": "text"})
	if isError {
		t.Fatalf("html text-mode fetch failed: %s", gotText)
	}
	if !strings.Contains(gotText, "long enough to clear the readability") {
		t.Errorf("text mode missing article substance: %q", gotText)
	}
	for _, marker := range []string{"# ", "**", "](https://example.com/x"} {
		if strings.Contains(gotText, marker) {
			t.Errorf("text mode must not contain markdown marker %q: %q", marker, gotText)
		}
	}

	gotMarkdown, isError := e2eCall(t, session, map[string]any{"url": ts.URL + "/mode.html"})
	if isError {
		t.Fatalf("html default-format fetch failed: %s", gotMarkdown)
	}
	if !strings.Contains(gotMarkdown, "**bold words**") && !strings.Contains(gotMarkdown, "# Plain Mode Heading") {
		t.Errorf("default format should be markdown with markup; got: %q", gotMarkdown)
	}
}

func TestE2E_UnsupportedContentTypeStillRejected(t *testing.T) {
	ts := newE2EServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 'P', 'N', 'G'})
	}))
	session := e2eSession(t)

	got, isError := e2eCall(t, session, map[string]any{"url": ts.URL + "/img.png"})
	if !isError {
		t.Errorf("expected IsError result for image/png, got content: %q", got)
	}
	if !strings.Contains(got, "unsupported content type") {
		t.Errorf("error text missing cause, got: %q", got)
	}
}

// TestE2E_OversizedBodyRejectedThenAdmittedWithMaxMB covers the size cap
// (ADR-0007) over the real transport. The body is ~2.1 MB served chunked (a
// body this size never gets a Content-Length from httptest, so the bounded
// read is what enforces the cap): the default 2 MB cap must reject it as an
// IsError result naming the cap, and max_mb=3 must admit the same URL whole.
func TestE2E_OversizedBodyRejectedThenAdmittedWithMaxMB(t *testing.T) {
	body := strings.Repeat("e2e", 700*1024+100)
	ts := newE2EServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}))
	session := e2eSession(t)

	got, isError := e2eCall(t, session, map[string]any{"url": ts.URL + "/huge.txt"})
	if !isError {
		t.Fatalf("expected IsError for a 2.1 MB body under the 2 MB default cap, got content of %d bytes", len(got))
	}
	if !strings.Contains(got, "size cap exceeded") || !strings.Contains(got, "max_mb") {
		t.Errorf("error should state the cap and the max_mb escape hatch; got: %q", got)
	}

	gotRaised, isError := e2eCall(t, session, map[string]any{"url": ts.URL + "/huge.txt", "max_mb": 3})
	if isError {
		t.Fatalf("expected max_mb=3 to admit the 2.1 MB body, got: %s", gotRaised)
	}
	if gotRaised != body {
		t.Errorf("raised-cap fetch must return the body whole; got %d bytes, want %d", len(gotRaised), len(body))
	}

	gotCeiling, isError := e2eCall(t, session, map[string]any{"url": ts.URL + "/huge.txt", "max_mb": 9})
	if !isError {
		t.Fatalf("expected IsError for max_mb above the ceiling, got content of %d bytes", len(gotCeiling))
	}
	if !strings.Contains(gotCeiling, "between 1 and 8") {
		t.Errorf("ceiling error should name the accepted range 1 to 8; got: %q", gotCeiling)
	}
}

func TestE2E_ToolDescriptionSurfacesPassthrough(t *testing.T) {
	session := e2eSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools.Tools) != 1 || tools.Tools[0].Name != "fetch" {
		t.Fatalf("expected exactly the fetch tool, got: %v", tools.Tools)
	}
	desc := strings.ToLower(tools.Tools[0].Description)
	if !strings.Contains(desc, "verbatim") || !strings.Contains(desc, "raw text") {
		t.Errorf("tool description does not surface the raw-text passthrough: %q", tools.Tools[0].Description)
	}
}

func TestE2E_RealRawGithubUsercontent(t *testing.T) {
	if os.Getenv("AWF_E2E_REAL") != "1" {
		t.Skip("set AWF_E2E_REAL=1 to run the live raw.githubusercontent.com check")
	}
	session := e2eSession(t)
	got, isError := e2eCall(t, session, map[string]any{
		"url": "https://raw.githubusercontent.com/ChHsiching/agent-web-fetch/main/README.md",
	})
	if isError {
		t.Fatalf("live raw fetch failed: %s", got)
	}
	if !strings.Contains(got, "agent-web-fetch") {
		t.Errorf("live raw fetch missing project name, first 200 bytes: %.200s", got)
	}
	if strings.Contains(got[:min(200, len(got))], "error:") {
		t.Errorf("live raw fetch looks like an error result: %.200s", got)
	}
}
