package fetch

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// These tests guard the fetch pipeline's handling of non-UTF-8 HTML — a real
// concern for Chinese government / school / forum sites that still ship GBK.
// They are regression tests: the pipeline decodes GBK correctly today (the
// heavy lifting is done by the HTML parser's charset detection inside the
// readability / html-to-markdown chain, which receives the raw bytes via
// strings.NewReader and reads <meta charset> / HTTP Content-Type / sniffs).
// These tests lock that behaviour down so a future change to how the body is
// read (e.g. an over-eager UTF-8 conversion before extraction) doesn't
// silently reintroduce mojibake.

const gbkArticleBody = "这是一篇用 GBK 编码的中文文章正文内容，长度足以超过可读性提取的阈值。"

func gbkArticleHTML(charset string) string {
	return `<!DOCTYPE html><html><head><meta charset="` + charset + `"><title>中文标题</title></head><body><article><h1>中文标题</h1><p>` +
		gbkArticleBody + `</p><p>补充段落让正文长度稳妥超过可读性阈值，走主提取路径而非全文回退。</p></article></body></html>`
}

func encodeGBK(s string) ([]byte, error) {
	var buf bytes.Buffer
	w := transform.NewWriter(&buf, simplifiedchinese.GBK.NewEncoder())
	if _, err := io.WriteString(w, s); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type gbkClient struct {
	body []byte
	ct   string
}

func (c *gbkClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": {c.ct}},
		Body:       io.NopCloser(bytes.NewReader(c.body)),
	}, nil
}

func TestFetch_DecodesGBK_WithMetaCharset(t *testing.T) {
	resetCache()
	body, err := encodeGBK(gbkArticleHTML("gbk"))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	res, err := Fetch(context.Background(), Params{URL: "https://example.cn/a"},
		&gbkClient{body: body, ct: "text/html; charset=gbk"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if !strings.Contains(res.Content, gbkArticleBody) {
		t.Errorf("expected decoded Chinese in content; got:\n%s", res.Content)
	}
}

func TestFetch_DecodesGBK_OnlyHTTPHeader(t *testing.T) {
	resetCache()
	htmlNoMeta := `<!DOCTYPE html><html><head><title>中文标题</title></head><body><article><h1>中文标题</h1><p>` +
		gbkArticleBody + `</p><p>补充段落让正文长度稳妥超过可读性阈值。</p></article></body></html>`
	body, err := encodeGBK(htmlNoMeta)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	res, err := Fetch(context.Background(), Params{URL: "https://example.cn/b"},
		&gbkClient{body: body, ct: "text/html; charset=gbk"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if !strings.Contains(res.Content, gbkArticleBody) {
		t.Errorf("expected decoded Chinese (HTTP-header-only charset); got:\n%s", res.Content)
	}
}

func TestFetch_DecodesGBK_NoCharsetDeclared(t *testing.T) {
	resetCache()
	htmlNoMeta := `<!DOCTYPE html><html><head><title>中文标题</title></head><body><article><h1>中文标题</h1><p>` +
		gbkArticleBody + `</p><p>补充段落让正文长度稳妥超过可读性阈值。</p></article></body></html>`
	body, err := encodeGBK(htmlNoMeta)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	res, err := Fetch(context.Background(), Params{URL: "https://example.cn/c"},
		&gbkClient{body: body, ct: "text/html"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if !strings.Contains(res.Content, gbkArticleBody) {
		t.Errorf("expected decoded Chinese (sniffed, no declaration); got:\n%s", res.Content)
	}
}
