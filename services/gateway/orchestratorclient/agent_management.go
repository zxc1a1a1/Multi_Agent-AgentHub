package orchestratorclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ProxyAgentManagement forwards an agent management HTTP request to the remote
// Orchestrator. pathSuffix is the suffix after /api/agents (e.g., "", "/foo",
// "/foo/check") and is appended to the fixed /internal/orchestrator/agents
// prefix. rawQuery is the undecoded query string from the original request
// (may be ""). This method reuses the existing baseURL, httpClient, and
// internalToken from OrchestratorRunService — no second HTTP client or URL
// config is created.
//
// The upstream URL is constructed safely via url.Parse:
//
//	baseURL / "/internal/orchestrator/agents" + pathSuffix + "?" + rawQuery
//
// pathSuffix and rawQuery are never used to set scheme or host.
func (s *OrchestratorRunService) ProxyAgentManagement(
	ctx context.Context, method, pathSuffix, rawQuery string,
	body io.Reader, contentType string,
) (*http.Response, error) {
	if s == nil {
		return nil, errors.New("orchestrator run service is nil")
	}

	target, err := url.Parse(s.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse orchestrator base URL: %w", err)
	}
	target.Path = strings.TrimRight(target.Path, "/") + "/internal/orchestrator/agents" + pathSuffix
	target.RawQuery = rawQuery

	req, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create proxy request: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if s.internalToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.internalToken)
	}
	return s.httpClient.Do(req)
}
