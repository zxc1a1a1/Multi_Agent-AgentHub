package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

var (
	ErrBlockedTarget = errors.New("network target is blocked")
	ErrSearchOffline = errors.New("web search provider is not configured")
)

// Resolver is injectable so SSRF tests do not depend on external DNS.
type Resolver interface {
	LookupIP(ctx context.Context, host string) ([]net.IP, error)
}

type netResolver struct{}

func (netResolver) LookupIP(ctx context.Context, host string) ([]net.IP, error) {
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		ips = append(ips, addr.IP)
	}
	return ips, nil
}

type WebPolicy struct {
	Resolver Resolver
}

func NewWebPolicy(resolver Resolver) *WebPolicy {
	if resolver == nil {
		resolver = netResolver{}
	}
	return &WebPolicy{Resolver: resolver}
}

// ValidateURL rejects non-HTTP(S), userinfo, literal/private/loopback/
// link-local/metadata targets. Every resolved address must be public.
func (p *WebPolicy) ValidateURL(ctx context.Context, raw string) (*url.URL, []net.IP, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, nil, fmt.Errorf("%w: scheme %q", ErrBlockedTarget, u.Scheme)
	}
	if u.User != nil {
		return nil, nil, fmt.Errorf("%w: userinfo is not allowed", ErrBlockedTarget)
	}
	host := u.Hostname()
	if host == "" {
		return nil, nil, fmt.Errorf("%w: missing host", ErrBlockedTarget)
	}
	var ips []net.IP
	if literal := net.ParseIP(host); literal != nil {
		ips = []net.IP{literal}
	} else {
		var err error
		ips, err = p.Resolver.LookupIP(ctx, host)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve host: %w", err)
		}
	}
	if len(ips) == 0 {
		return nil, nil, fmt.Errorf("%w: no addresses", ErrBlockedTarget)
	}
	for _, ip := range ips {
		if blockedIP(ip) {
			return nil, nil, fmt.Errorf("%w: %s", ErrBlockedTarget, ip.String())
		}
	}
	return u, ips, nil
}

func blockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		// Cloud metadata endpoints.
		if v4[0] == 169 && v4[1] == 254 {
			return true
		}
	}
	return false
}

type Fetcher struct {
	policy *WebPolicy
	client *http.Client
}

func NewFetcher(policy *WebPolicy) *Fetcher {
	if policy == nil {
		policy = NewWebPolicy(nil)
	}
	f := &Fetcher{policy: policy}
	f.client = &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			_, _, err := f.policy.ValidateURL(req.Context(), req.URL.String())
			return err
		},
	}
	return f
}

// WithHTTPClient is for tests and controlled deployments; the redirect policy
// remains enforced.
func (f *Fetcher) WithHTTPClient(client *http.Client) *Fetcher {
	if client != nil {
		f.client = client
	}
	return f
}

func (f *Fetcher) Fetch(ctx context.Context, raw string, maxBytes int) (string, string, error) {
	if _, _, err := f.policy.ValidateURL(ctx, raw); err != nil {
		return "", "", err
	}
	if maxBytes <= 0 || maxBytes > 1024*1024 {
		maxBytes = 256 * 1024
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "AgentHub/2.0 (+bounded fetch)")
	resp, err := f.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("upstream status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes)+1))
	if err != nil {
		return "", "", err
	}
	truncated := len(body) > maxBytes
	if truncated {
		body = body[:maxBytes]
	}
	contentType := resp.Header.Get("Content-Type")
	if truncated {
		contentType += "; truncated"
	}
	return string(body), contentType, nil
}

type SearchProvider interface {
	Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error)
}

type SearchResult struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func (w *Workspace) WebFetchTool(fetcher *Fetcher) *workspaceTool {
	schema := json.RawMessage(`{"type":"object","properties":{"url":{"type":"string"},"maxBytes":{"type":"integer","minimum":1,"maximum":1048576}},"required":["url"],"additionalProperties":false}`)
	d := Descriptor{
		Name: "web.fetch", Version: "1", Description: "Fetch a public HTTP(S) document with SSRF protections.",
		InputSchema: schema, ReadOnly: true, Approval: ApprovalConfirm,
		Timeout: 20 * time.Second, MaxOutputBytes: 256 * 1024, Provider: "native",
	}
	return &workspaceTool{name: d.Name, description: d.Description, schema: schema, descriptor: &d,
		run: func(ctx context.Context, raw json.RawMessage) (*adk.ToolResult, error) {
			if fetcher == nil {
				return nil, fmt.Errorf("web fetch is not configured")
			}
			var args struct {
				URL      string `json:"url"`
				MaxBytes int    `json:"maxBytes"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return nil, err
			}
			body, contentType, err := fetcher.Fetch(ctx, args.URL, args.MaxBytes)
			if err != nil {
				return nil, err
			}
			return jsonResult(map[string]any{"url": args.URL, "contentType": contentType, "content": body})
		}}
}

func WebSearchTool(provider SearchProvider) *workspaceTool {
	schema := json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"},"maxResults":{"type":"integer","minimum":1,"maximum":20}},"required":["query"],"additionalProperties":false}`)
	d := Descriptor{
		Name: "web.search", Version: "1", Description: "Search the web through a configured provider.",
		InputSchema: schema, ReadOnly: true, Approval: ApprovalConfirm,
		Timeout: 20 * time.Second, MaxOutputBytes: 64 * 1024, Provider: "native",
	}
	return &workspaceTool{name: d.Name, description: d.Description, schema: schema, descriptor: &d,
		run: func(ctx context.Context, raw json.RawMessage) (*adk.ToolResult, error) {
			if provider == nil {
				return nil, ErrSearchOffline
			}
			var args struct {
				Query      string `json:"query"`
				MaxResults int    `json:"maxResults"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return nil, err
			}
			if strings.TrimSpace(args.Query) == "" {
				return nil, fmt.Errorf("query is required")
			}
			results, err := provider.Search(ctx, args.Query, args.MaxResults)
			if err != nil {
				return nil, err
			}
			return jsonResult(map[string]any{"query": args.Query, "results": results})
		}}
}
