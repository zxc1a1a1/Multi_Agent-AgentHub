package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/execution"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

type handoffDispatch struct {
	Content  string
	Envelope *adk.RunIdentity
}

func handoffTestPlanJSON(t *testing.T) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"runId": "run-handoff", "conversationId": "conv-handoff",
		"executionPath": "group_chat", "strategy": "sequential",
		"allowedAgents": []string{"code-agent", "review-agent"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func handoffServerSetup(t *testing.T, store *fakeExecutionStore) (*Server, *[]handoffDispatch, *sync.Mutex, func()) {
	t.Helper()
	var mu sync.Mutex
	dispatches := make([]handoffDispatch, 0)
	child := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Params struct {
				Envelope *adk.RunIdentity `json:"envelope"`
				Message  struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		dispatches = append(dispatches, handoffDispatch{Content: body.Params.Message.Content, Envelope: body.Params.Envelope})
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"taskId":"remote-1","status":"completed","events":[{"author":"agent","role":"assistant","parts":[{"type":"text","text":"review ok"}],"final":true}]}`))
	}))
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: child.URL},
		{Name: "review-agent", URL: child.URL},
	})
	if err != nil {
		child.Close()
		t.Fatal(err)
	}
	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithExecutionStore(store),
		WithInstanceID("test-instance"),
	)
	return srv, &dispatches, &mu, child.Close
}

func handoffBody(overrides map[string]any) string {
	body := map[string]any{
		"runId": "run-handoff", "taskId": "task-1", "targetAgent": "review-agent",
		"message": "review the patch", "reason": "needs review",
		"fencingVersion": 2, "principalId": "user-1", "traceId": "trace-1",
		"delegateDepth": 0,
	}
	for key, value := range overrides {
		body[key] = value
	}
	raw, _ := json.Marshal(body)
	return string(raw)
}

func TestHandoffEndpointDispatchesAllowlistedTarget(t *testing.T) {
	store := &fakeExecutionStore{resumeState: &execution.ResumeState{PlanJSON: handoffTestPlanJSON(t)}}
	srv, dispatches, mu, closeChild := handoffServerSetup(t, store)
	defer closeChild()
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/tools/handoff", strings.NewReader(handoffBody(nil))))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Status      string `json:"status"`
		TaskID      string `json:"taskId"`
		TargetAgent string `json:"targetAgent"`
		Text        string `json:"text"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != "completed" || payload.TargetAgent != "review-agent" || payload.Text != "review ok" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*dispatches) != 1 {
		t.Fatalf("dispatches=%d, want 1", len(*dispatches))
	}
	dispatch := (*dispatches)[0]
	if dispatch.Content != "review the patch" {
		t.Fatalf("message not forwarded: %q", dispatch.Content)
	}
	if dispatch.Envelope == nil || dispatch.Envelope.RunID != "run-handoff" ||
		dispatch.Envelope.DelegateDepth != 1 || dispatch.Envelope.FencingVersion != 2 ||
		dispatch.Envelope.PrincipalID != "user-1" {
		t.Fatalf("delegate identity not forwarded: %#v", dispatch.Envelope)
	}
	if !strings.HasPrefix(dispatch.Envelope.TaskID, "task-1_handoff_review_agent_1_") {
		t.Fatalf("unexpected handoff task id: %s", dispatch.Envelope.TaskID)
	}
}

func TestHandoffEndpointRejectsOutsideBoundary(t *testing.T) {
	store := &fakeExecutionStore{resumeState: &execution.ResumeState{PlanJSON: handoffTestPlanJSON(t)}}
	srv, dispatches, mu, closeChild := handoffServerSetup(t, store)
	defer closeChild()
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/tools/handoff",
		strings.NewReader(handoffBody(map[string]any{"targetAgent": "web-agent"}))))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*dispatches) != 0 {
		t.Fatalf("out-of-boundary target dispatched: %d", len(*dispatches))
	}
}

func TestHandoffEndpointRejectsDepthAndContextOverflow(t *testing.T) {
	store := &fakeExecutionStore{resumeState: &execution.ResumeState{PlanJSON: handoffTestPlanJSON(t)}}
	srv, dispatches, mu, closeChild := handoffServerSetup(t, store)
	defer closeChild()

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/tools/handoff",
		strings.NewReader(handoffBody(map[string]any{"delegateDepth": 1}))))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("depth status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/tools/handoff",
		strings.NewReader(handoffBody(map[string]any{"message": strings.Repeat("m", handoffMaxMessageBytes+1)}))))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("context status=%d body=%s", rec.Code, rec.Body.String())
	}

	mu.Lock()
	defer mu.Unlock()
	if len(*dispatches) != 0 {
		t.Fatalf("rejected handoffs dispatched: %d", len(*dispatches))
	}
}

func TestHandoffEndpointRequiresRunPlan(t *testing.T) {
	store := &fakeExecutionStore{}
	srv, _, _, closeChild := handoffServerSetup(t, store)
	defer closeChild()
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/tools/handoff", strings.NewReader(handoffBody(nil))))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
