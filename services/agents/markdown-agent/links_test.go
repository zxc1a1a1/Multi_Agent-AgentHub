package markdownagent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestValidateLinkURLRejectsPrivateTargets(t *testing.T) {
	blocked := []string{
		"http://127.0.0.1/x",
		"http://10.0.0.1/x",
		"http://192.168.1.10/x",
		"http://169.254.169.254/latest/meta-data",
		"http://[::1]/x",
		"http://localhost:8080/x",
		"file:///etc/passwd",
		"http://user:pass@example.com/x",
	}
	for _, target := range blocked {
		if err := ValidateLinkURL(target); err == nil {
			t.Fatalf("target %q must be rejected", target)
		}
	}
	if err := ValidateLinkURL("https://example.com/docs"); err != nil {
		t.Fatalf("public URL rejected: %v", err)
	}
}

func TestCheckLinksBlocksPrivateWithoutNetworkCall(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	markdown := "see [local](" + server.URL + "/secret)\n"
	issues := CheckLinks(context.Background(), markdown)
	if len(issues) != 1 || issues[0].Code != "LINK_BLOCKED" {
		t.Fatalf("unexpected issues: %#v", issues)
	}
	if atomic.LoadInt32(&hits) != 0 {
		t.Fatal("private link must be rejected before any request")
	}
}

func TestSafeHTTPClientRejectsPrivateRedirect(t *testing.T) {
	client := SafeHTTPClient(time.Second)
	target, err := url.Parse("http://127.0.0.1/internal")
	if err != nil {
		t.Fatal(err)
	}
	request := &http.Request{URL: target}
	if err := client.CheckRedirect(request, nil); err == nil {
		t.Fatal("redirect to loopback must be rejected")
	}
}

func TestExtractLinksPositions(t *testing.T) {
	markdown := "line one\nsee [docs](https://example.com) and [x](https://example.com/y)\n```\n[ignored](http://127.0.0.1)\n```\n"
	refs := ExtractLinks(markdown)
	if len(refs) != 2 {
		t.Fatalf("expected 2 links, got %#v", refs)
	}
	if refs[0].line != 2 || refs[0].column != 12 || refs[1].url != "https://example.com/y" {
		t.Fatalf("unexpected link refs: %#v", refs)
	}
}

func TestCheckLinksToolReturnsJSON(t *testing.T) {
	result, err := (CheckLinksTool{}).Execute(context.Background(), []byte(`{"markdown":"[bad](http://127.0.0.1/)"}`))
	if err != nil || result == nil || result.IsError {
		t.Fatalf("tool failed: %#v err=%v", result, err)
	}
	if !strings.Contains(result.Content, "LINK_BLOCKED") {
		t.Fatalf("expected blocked finding: %s", result.Content)
	}
}
