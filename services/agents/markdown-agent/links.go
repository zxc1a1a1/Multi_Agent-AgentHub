package markdownagent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	ErrBlockedLinkTarget = errors.New("link target is not allowed")
	ErrInvalidLinkScheme = errors.New("link scheme is not allowed")
	markdownLinkPattern  = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)`)
)

// LinkIssue is one link-check finding.
type LinkIssue struct {
	URL     string `json:"url"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type linkRef struct {
	url     string
	line    int
	column  int
}

// ExtractLinks returns Markdown inline link targets with 1-based positions.
func ExtractLinks(markdown string) []linkRef {
	refs := make([]linkRef, 0)
	lines := strings.Split(markdown, "\n")
	inFence := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		for _, match := range markdownLinkPattern.FindAllStringSubmatchIndex(line, -1) {
			if len(match) < 4 {
				continue
			}
			target := line[match[2]:match[3]]
			refs = append(refs, linkRef{url: target, line: i + 1, column: match[2] + 1})
		}
	}
	return refs
}

// ValidateLinkURL rejects non-http(s), userinfo and literal private/loopback
// targets. Hostname DNS resolution is checked again at dial time.
func ValidateLinkURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("%w: invalid URL", ErrBlockedLinkTarget)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidLinkScheme
	}
	if parsed.User != nil {
		return fmt.Errorf("%w: credentials in URL", ErrBlockedLinkTarget)
	}
	host := strings.Trim(strings.ToLower(parsed.Hostname()), "[]")
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return fmt.Errorf("%w: localhost", ErrBlockedLinkTarget)
	}
	if ip := net.ParseIP(host); ip != nil && blockedIP(ip) {
		return fmt.Errorf("%w: private address", ErrBlockedLinkTarget)
	}
	return nil
}

func blockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() ||
		ip.Equal(net.ParseIP("169.254.169.254"))
}

// SafeHTTPClient returns a client that refuses private targets at connect time
// (including DNS rebinding) and validates every redirect target.
func SafeHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	dialer := &net.Dialer{Timeout: timeout}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			for _, candidate := range ips {
				if blockedIP(candidate.IP) {
					return nil, fmt.Errorf("%w: resolved private address", ErrBlockedLinkTarget)
				}
			}
			if len(ips) == 0 {
				return nil, fmt.Errorf("%w: no address", ErrBlockedLinkTarget)
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			return ValidateLinkURL(req.URL.String())
		},
	}
}

// CheckLinks extracts links and validates them without making network calls for
// private targets. Public links are checked with HEAD under the safe client.
func CheckLinks(ctx context.Context, markdown string) []LinkIssue {
	issues := make([]LinkIssue, 0)
	client := SafeHTTPClient(3 * time.Second)
	for _, ref := range ExtractLinks(markdown) {
		if err := ValidateLinkURL(ref.url); err != nil {
			code := "LINK_BLOCKED"
			if errors.Is(err, ErrInvalidLinkScheme) {
				code = "LINK_BAD_SCHEME"
			}
			issues = append(issues, LinkIssue{
				URL: ref.url, Line: ref.line, Column: ref.column, Code: code, Message: err.Error(),
			})
			continue
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodHead, ref.url, nil)
		if err != nil {
			issues = append(issues, LinkIssue{
				URL: ref.url, Line: ref.line, Column: ref.column, Code: "LINK_INVALID", Message: "invalid request",
			})
			continue
		}
		response, err := client.Do(request)
		if err != nil {
			issues = append(issues, LinkIssue{
				URL: ref.url, Line: ref.line, Column: ref.column, Code: "LINK_UNREACHABLE", Message: "link could not be checked",
			})
			continue
		}
		response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 400 {
			issues = append(issues, LinkIssue{
				URL: ref.url, Line: ref.line, Column: ref.column, Code: "LINK_STATUS",
				Message: fmt.Sprintf("unexpected status %d", response.StatusCode),
			})
		}
	}
	return issues
}
