package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/dispatcher"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/registry"
)

func TestExpandGroupChatRoundsRoundRobin(t *testing.T) {
	p := &plan.OrchestrationPlan{
		ExecutionPath: "group_chat", Strategy: plan.StrategyOrderedParallel,
		Participants: []plan.PlanParticipant{
			{AgentName: "code-agent", Selected: true},
			{AgentName: "review-agent", Selected: true},
			{AgentName: "test-agent", Selected: true},
		},
		Tasks: []plan.TaskPlan{{TaskID: "original", AgentName: "code-agent", TaskContent: "x"}},
	}
	expanded := expandGroupChatRounds(p, 3, "design a cache")
	if len(expanded.Tasks) != 9 {
		t.Fatalf("tasks=%d, want 9", len(expanded.Tasks))
	}
	if expanded.Strategy != plan.StrategySequential {
		t.Fatalf("strategy=%s, want sequential", expanded.Strategy)
	}
	for i, task := range expanded.Tasks {
		if i > 0 {
			if strings.EqualFold(task.AgentName, expanded.Tasks[i-1].AgentName) {
				t.Fatalf("consecutive speaker repeated at %d: %#v", i, task)
			}
			if len(task.DependsOn) != 1 || task.DependsOn[0] != expanded.Tasks[i-1].TaskID {
				t.Fatalf("task %d missing dependency chain: %#v", i, task)
			}
			if !strings.Contains(task.TaskContent, "{{deps."+expanded.Tasks[i-1].TaskID+".output}}") {
				t.Fatalf("task %d missing previous output injection", i)
			}
		}
	}
	if !strings.Contains(expanded.Tasks[0].TaskContent, "design a cache") {
		t.Fatalf("user request missing: %s", expanded.Tasks[0].TaskContent)
	}
	if len(p.Tasks) != 1 {
		t.Fatal("original plan must not be mutated")
	}
}

func TestExpandGroupChatRoundsDisabledOrSingleAgent(t *testing.T) {
	p := &plan.OrchestrationPlan{
		ExecutionPath: "group_chat", Participants: []plan.PlanParticipant{{AgentName: "code-agent", Selected: true}},
		Tasks: []plan.TaskPlan{{TaskID: "t1"}},
	}
	if got := expandGroupChatRounds(p, 1, "x"); got != p {
		t.Fatal("rounds=1 must keep the plan unchanged")
	}
	if got := expandGroupChatRounds(p, 3, "x"); got != p {
		t.Fatal("single agent must keep the plan unchanged")
	}
}

func TestGroupChatRoundsDispatchRoundRobin(t *testing.T) {
	t.Setenv("GROUP_CHAT_MAX_ROUNDS", "2")
	t.Setenv("ORCHESTRATOR_MAX_TASKS", "10")
	var messages []string
	var mu sync.Mutex
	child := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Params struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		messages = append(messages, body.Params.Message.Content)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"taskId":"task-1","status":"completed","events":[{"author":"agent","role":"assistant","parts":[{"type":"text","text":"ok"}],"final":true}]}`))
	}))
	defer child.Close()
	reg, err := registry.NewStaticAgentRegistry([]registry.AgentEndpoint{
		{Name: "code-agent", URL: child.URL},
		{Name: "review-agent", URL: child.URL},
	})
	if err != nil {
		t.Fatal(err)
	}
	p := makeTestPlan("run-rounds", plan.StrategyOrderedParallel, []plan.TaskPlan{makeTaskPlan("t1", "code-agent", "seed")})
	p.ExecutionPath = "group_chat"
	p.Participants = []plan.PlanParticipant{
		{AgentName: "code-agent", Selected: true},
		{AgentName: "review-agent", Selected: true},
	}
	p.CandidateParticipants = p.Participants
	p.DefaultSelectedParticipants = p.Participants
	p.AllowedAgents = []string{"code-agent", "review-agent"}
	p.PlanOwner = nil
	p.Strategy = plan.StrategyOrderedParallel
	srv := NewServer(
		WithRegistry(reg),
		WithDispatcher(dispatcher.NewA2ADispatcher()),
		WithMainAgentPlanner(&FakeMainAgent{PlanToReturn: p}),
	)
	body := `{"runId":"run-rounds","conversationId":"conv","messages":[{"role":"user","text":"design cache"}],"requestedPath":"group_chat","selectedAgentNames":["code-agent","review-agent"]}`
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(body)))

	mu.Lock()
	defer mu.Unlock()
	if len(messages) != 4 {
		t.Fatalf("dispatches=%d, want 4: body=%s", len(messages), rec.Body.String())
	}
	wantAgents := []string{"code-agent", "review-agent", "code-agent", "review-agent"}
	for i, message := range messages {
		if !strings.Contains(message, "speaker "+wantAgents[i]) {
			t.Fatalf("dispatch %d speaker wrong: %s", i, message)
		}
	}
	if !strings.Contains(messages[0], "Round 1") || !strings.Contains(messages[3], "Round 2") {
		t.Fatalf("round labels missing: %#v", messages)
	}
}
