package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NormalizeAgentURL validates and normalizes an agent base URL.
// It only allows http and https schemes, rejects empty/missing hosts,
// strips trailing slashes, and rejects dangerous schemes (file, javascript, etc.).
func NormalizeAgentURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("agent URL must not be empty")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse agent URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("agent URL scheme must be http or https, got %q", u.Scheme)
	}

	if u.Host == "" {
		return "", fmt.Errorf("agent URL must have a host")
	}

	u.Path = strings.TrimSuffix(u.Path, "/")
	u.RawPath = strings.TrimSuffix(u.RawPath, "/")

	return u.String(), nil
}

// FetchAgentCard fetches an agent's /.well-known/agent.json from the given base URL.
// It normalizes the URL, applies a default 5-second timeout (respecting the caller's
// shorter deadline if set), decodes the response into an AgentCard, and validates it.
func FetchAgentCard(ctx context.Context, baseURL string) (*AgentCard, error) {
	normalized, err := NormalizeAgentURL(baseURL)
	if err != nil {
		return nil, fmt.Errorf("normalize agent URL: %w", err)
	}

	// Apply default 5-second timeout, respecting any shorter caller deadline.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	wellKnownURL := normalized + "/.well-known/agent.json"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnownURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %s: %w", wellKnownURL, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch agent card from %s: %w", wellKnownURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch agent card from %s: HTTP %d", wellKnownURL, resp.StatusCode)
	}

	var card AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, fmt.Errorf("decode agent card from %s: %w", wellKnownURL, err)
	}

	errs := ValidateAgentCard(&card)
	if len(errs) > 0 {
		return nil, fmt.Errorf("validate agent card from %s: %d error(s): %v", wellKnownURL, len(errs), errs)
	}

	return &card, nil
}
