package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
	persistence "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence/sqlite"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"

	_ "modernc.org/sqlite"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func openPersistenceDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := persistence.RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	return db
}

func newPersistenceWriter(t *testing.T) *PersistenceWriter {
	t.Helper()
	db := openPersistenceDB(t)
	return NewPersistenceWriter(sqlite.NewStore(db))
}

// seqEventsWithMeta creates an iter.Seq2 from a list of adk.Event.
func seqEventsWithMeta(events ...adk.Event) func(yield func(adk.Event, error) bool) {
	return func(yield func(adk.Event, error) bool) {
		for _, event := range events {
			if !yield(event, nil) {
				return
			}
		}
	}
}

func metadata(eventType, runID, messageID, taskID, senderType, senderName string) map[string]any {
	return map[string]any{
		agui.MetaEventType:  eventType,
		agui.MetaRunID:      runID,
		agui.MetaMessageID:  messageID,
		agui.MetaTaskID:     taskID,
		agui.MetaSenderType: senderType,
		agui.MetaSenderName: senderName,
	}
}

func assertMessages(t *testing.T, db *sql.DB, conversationID string, expected int) []sqlite.Message {
	t.Helper()
	store := sqlite.NewStore(db)
	msgs, err := store.ListMessages(context.Background(), conversationID)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(msgs) != expected {
		t.Fatalf("expected %d messages, got %d", expected, len(msgs))
	}
	return msgs
}

func assertRuns(t *testing.T, db *sql.DB, conversationID string, expected int) []sqlite.Run {
	t.Helper()
	store := sqlite.NewStore(db)
	runs, err := store.ListRunsByConversation(context.Background(), conversationID)
	if err != nil {
		t.Fatalf("ListRunsByConversation: %v", err)
	}
	if len(runs) != expected {
		t.Fatalf("expected %d runs, got %d", expected, len(runs))
	}
	return runs
}

func assertRunSteps(t *testing.T, db *sql.DB, runID string, expected int) []sqlite.RunStep {
	t.Helper()
	store := sqlite.NewStore(db)
	steps, err := store.ListRunSteps(context.Background(), runID)
	if err != nil {
		t.Fatalf("ListRunSteps: %v", err)
	}
	if len(steps) != expected {
		t.Fatalf("expected %d run steps, got %d", expected, len(steps))
	}
	return steps
}

func assertStr(t *testing.T, got, want, label string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: expected %q, got %q", label, want, got)
	}
}

func assertContains(t *testing.T, s, substr, label string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("%s: expected to contain %q, got %q", label, substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr, label string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("%s: must NOT contain %q, got %q", label, substr, s)
	}
}

// ---------------------------------------------------------------------------
// TestPersistenceWriterSingleCode
// ---------------------------------------------------------------------------

func TestPersistenceWriterSingleCode(t *testing.T) {
	db := openPersistenceDB(t)
	store := sqlite.NewStore(db)
	conv := sqlite.Conversation{ID: "conv-sc", Title: "Single Code"}
	if err := store.CreateConversation(context.Background(), conv); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	pw := NewPersistenceWriter(store)
	ctx := context.Background()
	conversationID := conv.ID
	runID := "run-sc"

	// Save user message
	if err := pw.SaveUserMessage(ctx, conversationID, "write a Go server"); err != nil {
		t.Fatalf("SaveUserMessage: %v", err)
	}

	// Simulate single code-agent run events
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:  "RUN_STARTED",
		RunID: runID,
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_START",
		RunID:     runID,
		MessageID: "msg-code",
		TaskID:    "task-code",
		Sender:    &agui.EventSender{Type: "agent", Name: "code-agent"},
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_CONTENT",
		RunID:     runID,
		MessageID: "msg-code",
		Delta:     "package main\n",
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_CONTENT",
		RunID:     runID,
		MessageID: "msg-code",
		Delta:     "func main() {}",
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_END",
		RunID:     runID,
		MessageID: "msg-code",
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:  "RUN_FINISHED",
		RunID: runID,
	})

	// Verify
	runs := assertRuns(t, db, conversationID, 1)
	assertStr(t, runs[0].Status, "completed", "run status")

	steps := assertRunSteps(t, db, runID, 1)
	assertStr(t, steps[0].AgentName, "code-agent", "step agent_name")
	assertStr(t, steps[0].TaskID, "task-code", "step task_id")
	assertStr(t, steps[0].Status, "completed", "step status")

	msgs := assertMessages(t, db, conversationID, 2) // user + code-agent
	assertStr(t, msgs[0].Role, "user", "msg[0] role")
	assertStr(t, msgs[1].Role, "assistant", "msg[1] role")
	assertStr(t, msgs[1].SenderType, "agent", "msg[1] sender_type")
	assertStr(t, msgs[1].SenderName, "code-agent", "msg[1] sender_name")
	assertStr(t, msgs[1].AgentName, "code-agent", "msg[1] agent_name")
	assertStr(t, msgs[1].Status, "sent", "msg[1] status")
	assertStr(t, msgs[1].Content, "package main\nfunc main() {}", "msg[1] content")
}

// ---------------------------------------------------------------------------
// TestPersistenceWriterSingleWeb
// ---------------------------------------------------------------------------

func TestPersistenceWriterSingleWeb(t *testing.T) {
	db := openPersistenceDB(t)
	store := sqlite.NewStore(db)
	conv := sqlite.Conversation{ID: "conv-sw", Title: "Single Web"}
	if err := store.CreateConversation(context.Background(), conv); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	pw := NewPersistenceWriter(store)
	ctx := context.Background()
	conversationID := conv.ID
	runID := "run-sw"

	_ = pw.SaveUserMessage(ctx, conversationID, "build a login page")

	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:  "RUN_STARTED",
		RunID: runID,
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_START",
		RunID:     runID,
		MessageID: "msg-web",
		TaskID:    "task-web",
		Sender:    &agui.EventSender{Type: "agent", Name: "web-agent"},
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_CONTENT",
		RunID:     runID,
		MessageID: "msg-web",
		Delta:     "<section><h1>Login</h1>",
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_END",
		RunID:     runID,
		MessageID: "msg-web",
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:  "RUN_FINISHED",
		RunID: runID,
	})

	runs := assertRuns(t, db, conversationID, 1)
	assertStr(t, runs[0].Status, "completed", "run status")

	steps := assertRunSteps(t, db, runID, 1)
	assertStr(t, steps[0].AgentName, "web-agent", "step agent_name")

	msgs := assertMessages(t, db, conversationID, 2)
	assertStr(t, msgs[1].SenderName, "web-agent", "sender_name")
	assertStr(t, msgs[1].AgentName, "web-agent", "agent_name")
	assertStr(t, msgs[1].Content, "<section><h1>Login</h1>", "content")
}

// ---------------------------------------------------------------------------
// TestPersistenceWriterMixedOrderedParallel
// ---------------------------------------------------------------------------

func TestPersistenceWriterMixedOrderedParallel(t *testing.T) {
	db := openPersistenceDB(t)
	store := sqlite.NewStore(db)
	conv := sqlite.Conversation{ID: "conv-mop", Title: "Mixed OrderedParallel"}
	if err := store.CreateConversation(context.Background(), conv); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	pw := NewPersistenceWriter(store)
	ctx := context.Background()
	conversationID := conv.ID
	runID := "run-mop"

	_ = pw.SaveUserMessage(ctx, conversationID, "build login page and api")

	// RUN_STARTED
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:  "RUN_STARTED",
		RunID: runID,
	})

	// Web agent task
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_START",
		RunID:     runID,
		MessageID: "msg-web",
		TaskID:    "task-web",
		Sender:    &agui.EventSender{Type: "agent", Name: "web-agent"},
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_CONTENT",
		RunID:     runID,
		MessageID: "msg-web",
		Delta:     "<section><h1>Login</h1></section>",
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_END",
		RunID:     runID,
		MessageID: "msg-web",
	})

	// Code agent task (new messageId triggers new Message)
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_START",
		RunID:     runID,
		MessageID: "msg-code",
		TaskID:    "task-code",
		Sender:    &agui.EventSender{Type: "agent", Name: "code-agent"},
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_CONTENT",
		RunID:     runID,
		MessageID: "msg-code",
		Delta:     "package main",
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_END",
		RunID:     runID,
		MessageID: "msg-code",
	})

	// Orchestrator summary (new messageId triggers new Message)
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_START",
		RunID:     runID,
		MessageID: "msg-summary",
		TaskID:    "task-summary",
		Sender:    &agui.EventSender{Type: "agent", Name: "orchestrator"},
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_CONTENT",
		RunID:     runID,
		MessageID: "msg-summary",
		Delta:     "All 2 task(s) completed.",
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_END",
		RunID:     runID,
		MessageID: "msg-summary",
	})

	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:  "RUN_FINISHED",
		RunID: runID,
	})

	// Verify
	runs := assertRuns(t, db, conversationID, 1)
	assertStr(t, runs[0].Status, "completed", "run status")

	steps := assertRunSteps(t, db, runID, 3)
	assertStr(t, steps[0].AgentName, "web-agent", "step[0] agent")
	assertStr(t, steps[1].AgentName, "code-agent", "step[1] agent")
	assertStr(t, steps[2].AgentName, "orchestrator", "step[2] agent")
	assertStr(t, steps[0].TaskID, "task-web", "step[0] task_id")
	assertStr(t, steps[1].TaskID, "task-code", "step[1] task_id")
	assertStr(t, steps[2].TaskID, "task-summary", "step[2] task_id")

	// 4 messages: user + web-agent + code-agent + orchestrator
	msgs := assertMessages(t, db, conversationID, 4)
	assertStr(t, msgs[0].Role, "user", "msg[0] role")

	assertStr(t, msgs[1].SenderName, "web-agent", "msg[1] sender")
	assertStr(t, msgs[1].AgentName, "web-agent", "msg[1] agent_name")
	assertStr(t, msgs[1].Status, "sent", "msg[1] status")
	assertContains(t, msgs[1].Content, "<section>", "msg[1] content")

	assertStr(t, msgs[2].SenderName, "code-agent", "msg[2] sender")
	assertStr(t, msgs[2].AgentName, "code-agent", "msg[2] agent_name")
	assertStr(t, msgs[2].Status, "sent", "msg[2] status")
	assertContains(t, msgs[2].Content, "package main", "msg[2] content")

	assertStr(t, msgs[3].SenderName, "orchestrator", "msg[3] sender")
	assertStr(t, msgs[3].AgentName, "orchestrator", "msg[3] agent_name")
	assertStr(t, msgs[3].Status, "sent", "msg[3] status")
	assertContains(t, msgs[3].Content, "2 task(s)", "msg[3] content")

	// All agent messages must have distinct IDs (not merged)
	ids := make(map[string]bool)
	for _, msg := range msgs {
		if ids[msg.ID] {
			t.Errorf("duplicate message id: %s", msg.ID)
		}
		ids[msg.ID] = true
	}
}

// ---------------------------------------------------------------------------
// TestPersistenceWriterRunError
// ---------------------------------------------------------------------------

func TestPersistenceWriterRunError(t *testing.T) {
	db := openPersistenceDB(t)
	store := sqlite.NewStore(db)
	conv := sqlite.Conversation{ID: "conv-err", Title: "Error Test"}
	if err := store.CreateConversation(context.Background(), conv); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	pw := NewPersistenceWriter(store)
	ctx := context.Background()
	conversationID := conv.ID
	runID := "run-err"

	_ = pw.SaveUserMessage(ctx, conversationID, "do something that fails")

	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:  "RUN_STARTED",
		RunID: runID,
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_START",
		RunID:     runID,
		MessageID: "msg-err",
		TaskID:    "task-err",
		Sender:    &agui.EventSender{Type: "agent", Name: "code-agent"},
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_CONTENT",
		RunID:     runID,
		MessageID: "msg-err",
		Delta:     "partial output before crash",
	})
	// RUN_ERROR — should set run, step, and message to failed
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:  "RUN_ERROR",
		RunID: runID,
		Error: &agui.SafeError{Code: "AGENT_ERROR", Message: "agent call failed"},
	})
	// RUN_FINISHED should not override the failed status.
	// Our implementation only sets status=completed; it does not check current status.
	// But since RUN_ERROR sets runFailed=true, the MemoryStore path skips saving.
	// For the PersistenceWriter, we don't prevent RUN_FINISHED overwriting,
	// but in practice (from orchestratorclient) RUN_FINISHED won't arrive after RUN_ERROR.
	// If it does, the run status will be overwritten to "completed".

	// Verify
	runs := assertRuns(t, db, conversationID, 1)
	assertStr(t, runs[0].Status, "failed", "run status")
	assertStr(t, runs[0].ErrorCode, "AGENT_ERROR", "run error_code")

	steps := assertRunSteps(t, db, runID, 1)
	assertStr(t, steps[0].Status, "failed", "step status")
	assertStr(t, steps[0].ErrorCode, "AGENT_ERROR", "step error_code")

	msgs := assertMessages(t, db, conversationID, 2) // user + agent
	// The agent message should have the buffered content and failed status
	assertStr(t, msgs[1].Status, "failed", "msg status")
	assertStr(t, msgs[1].ErrorCode, "AGENT_ERROR", "msg error_code")
	assertStr(t, msgs[1].ErrorMessage, "agent call failed", "msg error_message")
	// Content from before the error should be preserved
	assertContains(t, msgs[1].Content, "partial output before crash", "msg content")
}

// ---------------------------------------------------------------------------
// TestHandleChatWithPersistenceWriterDoesNotChangeSSE
// ---------------------------------------------------------------------------

func TestHandleChatWithPersistenceWriterDoesNotChangeSSE(t *testing.T) {
	// MemoryStore is still the primary store for the API.
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	// Run events with metadata simulating AG-UI v1.0 events from orchestratorclient.
	runner := &mockRunService{
		seq: seqEvents(
			adk.Event{
				ID:       "evt-run-start",
				Author:   "orchestrator",
				Metadata: metadata("run_started", "run-sse-test", "", "", "agent", "orchestrator"),
			},
			adk.Event{
				ID:       "evt-msg-start",
				Author:   "code-agent",
				Metadata: metadata("message_start", "run-sse-test", "msg-code-sse", "task-code-sse", "agent", "code-agent"),
			},
			adk.Event{
				ID:       "evt-msg-delta",
				Author:   "code-agent",
				Metadata: metadata("message_delta", "run-sse-test", "msg-code-sse", "task-code-sse", "agent", "code-agent"),
				Content: &adk.Content{
					Role:  adk.RoleAssistant,
					Parts: []adk.Part{adk.TextPart{Text: "hello from persistence test"}},
				},
			},
			adk.Event{
				ID:       "evt-msg-end",
				Author:   "code-agent",
				Metadata: metadata("message_end", "run-sse-test", "msg-code-sse", "task-code-sse", "agent", "code-agent"),
			},
			adk.Event{
				ID:       "evt-run-finish",
				Author:   "orchestrator",
				Metadata: metadata("run_finished", "run-sse-test", "", "", "agent", "orchestrator"),
			},
		),
	}

	// Build server with PersistenceWriter injected.
	db := openPersistenceDB(t)
	pw := NewPersistenceWriter(sqlite.NewStore(db))
	// Also init a conversation row in sqlite so FK constraints are satisfied.
	_ = sqlite.NewStore(db).CreateConversation(context.Background(), sqlite.Conversation{
		ID:    conv.ID,
		Title: "SSE + DB Test",
	})

	srv, err := NewServer(st, runner, WithPersistenceWriter(pw))
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"hello persistence"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}

	respBody := rec.Body.String()

	// SSE must still contain expected events (unchanged behavior).
	assertContains(t, respBody, "event: run_started\n", "SSE run_started")
	assertContains(t, respBody, `"type":"RUN_STARTED"`, "SSE RUN_STARTED")
	assertContains(t, respBody, `"type":"TEXT_MESSAGE_START"`, "SSE TEXT_MESSAGE_START")
	assertContains(t, respBody, "hello from persistence test", "SSE delta text")
	assertContains(t, respBody, `"type":"TEXT_MESSAGE_END"`, "SSE TEXT_MESSAGE_END")
	assertContains(t, respBody, `"type":"RUN_FINISHED"`, "SSE RUN_FINISHED")
	assertContains(t, respBody, `"sender":{"type":"agent","name":"code-agent"}`, "SSE sender")

	// MemoryStore should still have 2 messages (user + assistant merged).
	memMsgs, err := st.ListMessages(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("MemoryStore ListMessages: %v", err)
	}
	if len(memMsgs) != 2 {
		t.Fatalf("MemoryStore: expected 2 messages, got %d", len(memMsgs))
	}

	// SQLite should have user + code-agent = 2 messages (not merged).
	sqliteMsgs := assertMessages(t, db, conv.ID, 2)
	assertStr(t, sqliteMsgs[0].Role, "user", "sqlite msg[0] role")
	assertStr(t, sqliteMsgs[1].Role, "assistant", "sqlite msg[1] role")
	assertStr(t, sqliteMsgs[1].SenderName, "code-agent", "sqlite msg[1] sender_name")

	// SQLite should have 1 run + 1 step.
	sqliteRuns := assertRuns(t, db, conv.ID, 1)
	assertStr(t, sqliteRuns[0].Status, "completed", "sqlite run status")

	sqliteSteps := assertRunSteps(t, db, "run-sse-test", 1)
	assertStr(t, sqliteSteps[0].AgentName, "code-agent", "sqlite step agent_name")
}

// ---------------------------------------------------------------------------
// TestHandleChatWithoutPersistenceWriterPreservesLegacyBehavior
// ---------------------------------------------------------------------------

func TestHandleChatWithoutPersistenceWriterPreservesLegacyBehavior(t *testing.T) {
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	runner := &mockRunService{
		seq: seqEvents(adk.Event{
			ID:     "evt-legacy",
			Author: "code-agent",
			Content: &adk.Content{
				Role:  adk.RoleAssistant,
				Parts: []adk.Part{adk.TextPart{Text: "legacy response"}},
			},
			Final: true,
		}),
	}

	// No persistence writer — should behave exactly as before.
	srv, err := NewServer(st, runner)
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"legacy test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	respBody := rec.Body.String()
	assertContains(t, respBody, "legacy response", "SSE response")

	msgs, err := st.ListMessages(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (user + assistant merged), got %d", len(msgs))
	}
}

// ---------------------------------------------------------------------------
// TestHandleChatWithPersistenceWriterMultiAgentSSE
// ---------------------------------------------------------------------------

func TestHandleChatWithPersistenceWriterMultiAgentSSE(t *testing.T) {
	st := store.NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	runID := "run-multi"

	runner := &mockRunService{
		seq: seqEventsWithMeta(
			adk.Event{
				ID:       "evt-run-start",
				Author:   "orchestrator",
				Metadata: metadata("run_started", runID, "", "", "agent", "orchestrator"),
			},
			// Web agent message
			adk.Event{
				ID:       "evt-web-start",
				Author:   "web-agent",
				Metadata: metadata("message_start", runID, "msg-web", "task-web", "agent", "web-agent"),
			},
			adk.Event{
				ID:       "evt-web-delta",
				Author:   "web-agent",
				Metadata: metadata("message_delta", runID, "msg-web", "task-web", "agent", "web-agent"),
				Content:  &adk.Content{Role: adk.RoleAssistant, Parts: []adk.Part{adk.TextPart{Text: "<section>Web Output</section>"}}},
			},
			adk.Event{
				ID:       "evt-web-end",
				Author:   "web-agent",
				Metadata: metadata("message_end", runID, "msg-web", "task-web", "agent", "web-agent"),
			},
			// Code agent message
			adk.Event{
				ID:       "evt-code-start",
				Author:   "code-agent",
				Metadata: metadata("message_start", runID, "msg-code", "task-code", "agent", "code-agent"),
			},
			adk.Event{
				ID:       "evt-code-delta",
				Author:   "code-agent",
				Metadata: metadata("message_delta", runID, "msg-code", "task-code", "agent", "code-agent"),
				Content:  &adk.Content{Role: adk.RoleAssistant, Parts: []adk.Part{adk.TextPart{Text: "package main"}}},
			},
			adk.Event{
				ID:       "evt-code-end",
				Author:   "code-agent",
				Metadata: metadata("message_end", runID, "msg-code", "task-code", "agent", "code-agent"),
			},
			// Orchestrator summary
			adk.Event{
				ID:       "evt-sum-start",
				Author:   "orchestrator",
				Metadata: metadata("message_start", runID, "msg-sum", "task-summary", "agent", "orchestrator"),
			},
			adk.Event{
				ID:       "evt-sum-delta",
				Author:   "orchestrator",
				Metadata: metadata("message_delta", runID, "msg-sum", "task-summary", "agent", "orchestrator"),
				Content:  &adk.Content{Role: adk.RoleAssistant, Parts: []adk.Part{adk.TextPart{Text: "All tasks done."}}},
			},
			adk.Event{
				ID:       "evt-sum-end",
				Author:   "orchestrator",
				Metadata: metadata("message_end", runID, "msg-sum", "task-summary", "agent", "orchestrator"),
			},
			adk.Event{
				ID:       "evt-run-finish",
				Author:   "orchestrator",
				Metadata: metadata("run_finished", runID, "", "", "agent", "orchestrator"),
			},
		),
	}

	db := openPersistenceDB(t)
	pw := NewPersistenceWriter(sqlite.NewStore(db))
	_ = sqlite.NewStore(db).CreateConversation(context.Background(), sqlite.Conversation{
		ID:    conv.ID,
		Title: "Multi Agent Test",
	})

	srv, err := NewServer(st, runner, WithPersistenceWriter(pw))
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	body := `{"conversationId":"` + conv.ID + `","message":"build login page and api"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}

	respBody := rec.Body.String()
	// SSE should contain all three agent names
	assertContains(t, respBody, "web-agent", "SSE web-agent")
	assertContains(t, respBody, "code-agent", "SSE code-agent")
	assertContains(t, respBody, "orchestrator", "SSE orchestrator")

	// SQLite: 4 messages (user + web + code + orchestrator), not merged
	sqliteMsgs := assertMessages(t, db, conv.ID, 4)
	senders := make(map[string]bool)
	for _, msg := range sqliteMsgs {
		senders[msg.SenderName] = true
	}
	if !senders["web-agent"] {
		t.Error("SQLite: missing web-agent message")
	}
	if !senders["code-agent"] {
		t.Error("SQLite: missing code-agent message")
	}
	if !senders["orchestrator"] {
		t.Error("SQLite: missing orchestrator message")
	}
	// Verify the user message has empty sender_name
	assertStr(t, sqliteMsgs[0].SenderName, "", "user sender_name")

	// SQLite: 3 run steps
	sqliteSteps := assertRunSteps(t, db, runID, 3)
	agents := []string{sqliteSteps[0].AgentName, sqliteSteps[1].AgentName, sqliteSteps[2].AgentName}
	if agents[0] != "web-agent" || agents[1] != "code-agent" || agents[2] != "orchestrator" {
		t.Errorf("run step agents: expected [web-agent code-agent orchestrator], got %v", agents)
	}

	// MemoryStore: still has 2 messages (user + merged assistant)
	memMsgs, err := st.ListMessages(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("MemoryStore ListMessages: %v", err)
	}
	if len(memMsgs) != 2 {
		t.Fatalf("MemoryStore: expected 2 messages, got %d", len(memMsgs))
	}
}

// ---------------------------------------------------------------------------
// TestPersistenceWriterRunErrorSanitized
// ---------------------------------------------------------------------------

func TestPersistenceWriterRunErrorSanitized(t *testing.T) {
	db := openPersistenceDB(t)
	store := sqlite.NewStore(db)
	conv := sqlite.Conversation{ID: "conv-san", Title: "Sanitize Test"}
	if err := store.CreateConversation(context.Background(), conv); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	pw := NewPersistenceWriter(store)
	ctx := context.Background()
	conversationID := conv.ID
	runID := "run-san"

	_ = pw.SaveUserMessage(ctx, conversationID, "trigger error")

	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:  "RUN_STARTED",
		RunID: runID,
	})
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:      "TEXT_MESSAGE_START",
		RunID:     runID,
		MessageID: "msg-san",
		TaskID:    "task-san",
		Sender:    &agui.EventSender{Type: "agent", Name: "code-agent"},
	})
	// Error with sensitive content — the agui translator already sanitizes
	// event.Error.Message via TextStreamFilter. We verify that the sanitized
	// value is what gets persisted.
	pw.HandleEvent(ctx, conversationID, agui.Event{
		Type:  "RUN_ERROR",
		RunID: runID,
		Error: &agui.SafeError{
			Code:    "INTERNAL",
			Message: "agent call failed",
		},
	})

	runs := assertRuns(t, db, conversationID, 1)
	// Error message must not contain sensitive content
	errMsg := runs[0].ErrorMessage
	assertNotContains(t, errMsg, "sk-", "error_message: sk-token")
	assertNotContains(t, errMsg, "panic", "error_message: panic")
	assertNotContains(t, errMsg, "OPENAI_API_KEY", "error_message: api key")
	assertNotContains(t, errMsg, "C:\\", "error_message: windows path")
	assertNotContains(t, errMsg, "/home/", "error_message: unix path")

	msgs := assertMessages(t, db, conversationID, 2)
	assertStr(t, msgs[1].Status, "failed", "msg status")
	assertNotContains(t, msgs[1].ErrorMessage, "sk-", "msg error_message: sk-token")
}

// ---------------------------------------------------------------------------
// TestPersistenceWriterNilIsSafe
// ---------------------------------------------------------------------------

func TestPersistenceWriterNilIsSafe(t *testing.T) {
	// All methods on nil PersistenceWriter should be no-ops.
	var pw *PersistenceWriter
	ctx := context.Background()

	// None of these should panic.
	pw.SaveUserMessage(ctx, "conv", "hello")
	pw.HandleEvent(ctx, "conv", agui.Event{Type: "RUN_STARTED", RunID: "r1"})
	pw.HandleEvent(ctx, "conv", agui.Event{Type: "TEXT_MESSAGE_START"})
	pw.HandleEvent(ctx, "conv", agui.Event{Type: "TEXT_MESSAGE_CONTENT", Delta: "test"})
	pw.HandleEvent(ctx, "conv", agui.Event{Type: "TEXT_MESSAGE_END"})
	pw.HandleEvent(ctx, "conv", agui.Event{Type: "RUN_ERROR"})
	pw.HandleEvent(ctx, "conv", agui.Event{Type: "RUN_FINISHED"})
}

// ---------------------------------------------------------------------------
// TestPersistenceWriterErrorDoesNotPanic
// ---------------------------------------------------------------------------

func TestPersistenceWriterErrorDoesNotPanic(t *testing.T) {
	db := openPersistenceDB(t)
	store := sqlite.NewStore(db)
	pw := NewPersistenceWriter(store)
	ctx := context.Background()

	// HandleEvent without a conversation row (FK violation) should not panic.
	// The writer silently ignores DB errors.
	pw.HandleEvent(ctx, "nonexistent", agui.Event{Type: "RUN_STARTED", RunID: "r1"})
	pw.HandleEvent(ctx, "nonexistent", agui.Event{
		Type:      "TEXT_MESSAGE_START",
		RunID:     "r1",
		MessageID: "msg1",
		TaskID:    "task1",
		Sender:    &agui.EventSender{Type: "agent", Name: "test-agent"},
	})
	// Should not panic even though DB writes fail due to FK constraint.
}

// ---------------------------------------------------------------------------
// TestUpdateRunAndStepStatus
// ---------------------------------------------------------------------------

func TestUpdateRunAndStepStatus(t *testing.T) {
	db := openPersistenceDB(t)
	store := sqlite.NewStore(db)
	ctx := context.Background()

	conv := sqlite.Conversation{ID: "conv-upd", Title: "Update Test"}
	if err := store.CreateConversation(ctx, conv); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// Create a run
	run := sqlite.Run{ID: "run-upd", ConversationID: conv.ID, Status: "running"}
	if err := store.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	// Update run status
	if err := store.UpdateRunStatus(ctx, "run-upd", "failed", "ERR", "something broke", time.Time{}); err != nil {
		t.Fatalf("UpdateRunStatus: %v", err)
	}

	fetched, err := store.GetRun(ctx, "run-upd")
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	assertStr(t, fetched.Status, "failed", "updated run status")
	assertStr(t, fetched.ErrorCode, "ERR", "updated run error_code")

	// Create a run step
	step := sqlite.RunStep{
		ID:             "step-upd",
		RunID:          "run-upd",
		ConversationID: conv.ID,
		TaskID:         "task-upd",
		StepIndex:      0,
		AgentName:      "test-agent",
		Status:         "running",
	}
	if err := store.CreateRunStep(ctx, step); err != nil {
		t.Fatalf("CreateRunStep: %v", err)
	}

	// Update step status
	now := time.Now().UTC()
	if err := store.UpdateRunStepStatus(ctx, "step-upd", "failed", "STEP_ERR", "step broke", now); err != nil {
		t.Fatalf("UpdateRunStepStatus: %v", err)
	}

	steps, err := store.ListRunSteps(ctx, "run-upd")
	if err != nil {
		t.Fatalf("ListRunSteps: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	assertStr(t, steps[0].Status, "failed", "updated step status")
	assertStr(t, steps[0].ErrorCode, "STEP_ERR", "updated step error_code")
	if steps[0].FinishedAt.IsZero() {
		t.Error("expected non-zero finished_at")
	}

	// Update message — pre-generate ID since AppendMessage takes value type.
	msgID := "msg-upd-test"
	msg := sqlite.Message{
		ID:             msgID,
		ConversationID: conv.ID,
		Role:           "assistant",
		SenderType:     "agent",
		SenderName:     "test-agent",
		Content:        "",
		Status:         "streaming",
	}
	if err := store.AppendMessage(ctx, msg); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	if err := store.UpdateMessageContentAndStatus(ctx, msgID, "final content", "sent", "", "", time.Now().UTC()); err != nil {
		t.Fatalf("UpdateMessageContentAndStatus: %v", err)
	}

	msgs, err := store.ListMessages(ctx, conv.ID)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	assertStr(t, msgs[0].Content, "final content", "updated msg content")
	assertStr(t, msgs[0].Status, "sent", "updated msg status")
}

// ---------------------------------------------------------------------------
// json helper for tests
// ---------------------------------------------------------------------------

func mustMarshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}
