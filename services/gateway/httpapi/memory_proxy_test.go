package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

type stubMemoryProxy struct {
	method string
	path   string
	body   string
	status int
	resp   string
}

func (s *stubMemoryProxy) DoMemory(_ context.Context, method, path string, body []byte) (int, []byte, error) {
	s.method = method
	s.path = path
	s.body = string(body)
	return s.status, []byte(s.resp), nil
}

func TestMemoryProxyForwards(t *testing.T) {
	stub := &stubMemoryProxy{status: http.StatusOK, resp: `{"entries":[]}`}
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{}, WithMemoryProxy(stub))
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/memory/propose?x=1", strings.NewReader(`{"scope":{}}`))
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "entries") {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
	if stub.method != http.MethodPost || stub.path != "/internal/memory/propose?x=1" {
		t.Fatalf("proxy path/method wrong: %s %s", stub.method, stub.path)
	}
	if stub.body != `{"scope":{}}` {
		t.Fatalf("body not forwarded: %s", stub.body)
	}
}

func TestMemoryProxyUnavailable(t *testing.T) {
	srv, err := NewServer(store.NewMemoryStore(), &mockRunService{})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/memory/policy", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}
