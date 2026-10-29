package tools

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeResolver map[string][]string

func (f fakeResolver) LookupIP(_ context.Context, host string) ([]net.IP, error) {
	raw, ok := f[host]
	if !ok {
		return nil, errors.New("no such host")
	}
	out := make([]net.IP, len(raw))
	for i, s := range raw {
		out[i] = net.ParseIP(s)
	}
	return out, nil
}

func TestWebPolicyBlocksPrivateAndMetadataTargets(t *testing.T) {
	policy := NewWebPolicy(fakeResolver{
		"localhost":        {"127.0.0.1"},
		"metadata.example": {"169.254.169.254"},
		"private.example":  {"10.1.2.3"},
		"public.example":   {"93.184.216.34"},
	})
	for _, raw := range []string{
		"file:///etc/passwd", "ftp://public.example/x", "http://user:pass@public.example/",
		"http://localhost/", "http://metadata.example/latest", "http://private.example/",
		"http://127.0.0.1/", "http://[::1]/",
	} {
		if _, _, err := policy.ValidateURL(context.Background(), raw); !errors.Is(err, ErrBlockedTarget) {
			t.Fatalf("%s must be blocked, got %v", raw, err)
		}
	}
	if _, ips, err := policy.ValidateURL(context.Background(), "https://public.example/doc"); err != nil || len(ips) != 1 {
		t.Fatalf("public URL rejected: %v %#v", err, ips)
	}
}

func TestFetchFollowsOnlySafeRedirects(t *testing.T) {
	resolver := fakeResolver{
		"public.example":   {"93.184.216.34"},
		"internal.example": {"10.0.0.5"},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start":
			http.Redirect(w, r, "http://internal.example/secret", http.StatusFound)
		case "/doc":
			_, _ = w.Write([]byte("0123456789"))
		case "/secret":
			_, _ = w.Write([]byte("should not be reached"))
		}
	}))
	defer ts.Close()
	transport := &http.Transport{DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "tcp", ts.Listener.Addr().String())
	}}
	fetcher := NewFetcher(NewWebPolicy(resolver))
	fetcher.WithHTTPClient(&http.Client{Transport: transport, CheckRedirect: fetcher.client.CheckRedirect})

	if _, _, err := fetcher.Fetch(context.Background(), "http://public.example/start", 1024); !errors.Is(err, ErrBlockedTarget) {
		t.Fatalf("redirect to private IP must be blocked, got %v", err)
	}
	body, contentType, err := fetcher.Fetch(context.Background(), "http://public.example/doc", 4)
	if err != nil {
		t.Fatalf("fetch doc: %v", err)
	}
	if body != "0123" || !strings.Contains(contentType, "truncated") {
		t.Fatalf("unexpected fetch result: %q %q", body, contentType)
	}
}

func TestWebToolsUseConfiguredProviderOnly(t *testing.T) {
	w, _ := newTestWorkspace(t)
	tool := w.WebFetchTool(NewFetcher(NewWebPolicy(fakeResolver{"public.example": {"93.184.216.34"}})))
	if tool.Descriptor().Approval != ApprovalConfirm || !tool.Descriptor().ReadOnly {
		t.Fatalf("web fetch descriptor wrong: %#v", tool.Descriptor())
	}
	search := WebSearchTool(nil)
	res, _ := search.Execute(context.Background(), json.RawMessage(`{"query":"x"}`))
	if !res.IsError || !strings.Contains(res.Content, "not configured") {
		t.Fatalf("search without provider must be unavailable: %#v", res)
	}
}
