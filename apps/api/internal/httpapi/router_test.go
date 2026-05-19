package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/your-org/multi-agent-framework/apps/api/internal/config"
)

func TestCreateAgentRun(t *testing.T) {
	handler := NewRouter(config.Config{WebOrigin: "http://localhost:5173"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-runs", bytes.NewBufferString(`{"message":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
}

func TestCreateAgentRunValidatesMessage(t *testing.T) {
	handler := NewRouter(config.Config{WebOrigin: "http://localhost:5173"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-runs", bytes.NewBufferString(`{"message":""}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", res.Code, res.Body.String())
	}
}
