package orchestratorclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ProxyInternal proxies a constrained API request from Gateway to Orchestrator.
// Gateway remains protocol passthrough; it does not validate or mutate workspace
// files and does not call child agents.
func (s *OrchestratorRunService) ProxyInternal(ctx context.Context, method string, internalPath string, body io.Reader, contentType string) (*http.Response, error) {
	if s == nil {
		return nil, fmt.Errorf("orchestrator service is nil")
	}
	method = strings.TrimSpace(method)
	internalPath = strings.TrimSpace(internalPath)
	if method == "" || internalPath == "" || !strings.HasPrefix(internalPath, "/internal/orchestrator/") {
		return nil, fmt.Errorf("invalid proxy request")
	}
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+internalPath, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if s.internalToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.internalToken)
	}
	return s.httpClient.Do(req)
}
