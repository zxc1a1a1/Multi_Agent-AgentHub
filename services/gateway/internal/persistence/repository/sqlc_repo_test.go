package repository_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/db"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/domain"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence/goosemigrate"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence/repository"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	d, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	d.SetMaxOpenConns(1)
	d.SetMaxIdleConns(1)
	t.Cleanup(func() { d.Close() })
	// Use goose migrations from db/migrations or a test directory.
	if err := goosemigrate.Up(d, "../../../db/migrations"); err != nil {
		t.Fatalf("goose up: %v", err)
	}
	return d
}

func ctx() context.Context { return context.Background() }

func newTestRepos(t *testing.T) (*sql.DB, *repository.ConversationRepo, *repository.MessageRepo, *repository.RunRepo, *repository.EventRepo, *repository.RunStepRepo) {
	t.Helper()
	d := openTestDB(t)
	q := db.New(d)
	return d, repository.NewConversationRepo(q), repository.NewMessageRepo(q, d), repository.NewRunRepo(q), repository.NewEventRepo(q, d), repository.NewRunStepRepo(q)
}

// ---------------------------------------------------------------------------
// ConversationRepository tests
// ---------------------------------------------------------------------------

func TestConversationCRUD(t *testing.T) {
	_, conv, _, _, _, _ := newTestRepos(t)

	// Create
	c, err := conv.Create(ctx(), domain.CreateConversationInput{
		ID:           "conv-test-1",
		UserID:       "user-1",
		Title:        "Test Conv",
		Mode:         "direct",
		ResponseMode: "separate",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.ID != "conv-test-1" {
		t.Errorf("id: expected conv-test-1, got %s", c.ID)
	}
	if c.Version != 1 {
		t.Errorf("version: expected 1, got %d", c.Version)
	}
	if c.Status != "active" {
		t.Errorf("status: expected active, got %s", c.Status)
	}

	// Get
	fetched, err := conv.Get(ctx(), "user-1", "conv-test-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fetched.Title != "Test Conv" {
		t.Errorf("title: expected 'Test Conv', got %s", fetched.Title)
	}

	// Get with wrong user should fail
	_, err = conv.Get(ctx(), "user-2", "conv-test-1")
	if err != domain.ErrNotFound {
		t.Errorf("Get wrong user: expected ErrNotFound, got %v", err)
	}

	// List
	list, err := conv.List(ctx(), "user-1", domain.Pagination{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("List: expected 1, got %d", len(list))
	}

	// Update (optimistic version)
	updated, err := conv.Update(ctx(), "user-1", "conv-test-1", 1, "Updated Title")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("title: expected 'Updated Title', got %s", updated.Title)
	}
	if updated.Version != 2 {
		t.Errorf("version: expected 2, got %d", updated.Version)
	}

	// Update with stale version should fail
	_, err = conv.Update(ctx(), "user-1", "conv-test-1", 1, "Stale")
	if err != domain.ErrConflict {
		t.Errorf("stale update: expected ErrConflict, got %v", err)
	}

	// SetPinned
	if err := conv.SetPinned(ctx(), "user-1", "conv-test-1", true); err != nil {
		t.Errorf("SetPinned(true): %v", err)
	}

	// SetArchived
	if err := conv.SetArchived(ctx(), "user-1", "conv-test-1", true); err != nil {
		t.Errorf("SetArchived(true): %v", err)
	}

	// SoftDelete
	if err := conv.SoftDelete(ctx(), "user-1", "conv-test-1"); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	// Get after soft delete should fail
	_, err = conv.Get(ctx(), "user-1", "conv-test-1")
	if err != domain.ErrNotFound {
		t.Errorf("Get after delete: expected ErrNotFound, got %v", err)
	}
}

func TestConversationUserIsolation(t *testing.T) {
	_, conv, _, _, _, _ := newTestRepos(t)

	// Create conv for user-1
	_, err := conv.Create(ctx(), domain.CreateConversationInput{
		ID: "isol-1", UserID: "user-1", Title: "U1 Conv", Mode: "direct", ResponseMode: "separate",
	})
	if err != nil {
		t.Fatalf("Create user-1: %v", err)
	}

	// Create conv for user-2
	_, err = conv.Create(ctx(), domain.CreateConversationInput{
		ID: "isol-2", UserID: "user-2", Title: "U2 Conv", Mode: "direct", ResponseMode: "separate",
	})
	if err != nil {
		t.Fatalf("Create user-2: %v", err)
	}

	// user-1 should only see their conversation
	list, err := conv.List(ctx(), "user-1", domain.Pagination{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("List user-1: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("user-1 list: expected 1, got %d", len(list))
	}
	if list[0].ID != "isol-1" {
		t.Errorf("user-1 list: expected isol-1, got %s", list[0].ID)
	}

	// user-1 cannot access user-2's conversation
	_, err = conv.Get(ctx(), "user-1", "isol-2")
	if err != domain.ErrNotFound {
		t.Errorf("cross-user access: expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// MessageRepository tests
// ---------------------------------------------------------------------------

func TestMessageCreateAndList(t *testing.T) {
	_, conv, msg, _, _, _ := newTestRepos(t)

	// Need a conversation first.
	_, err := conv.Create(ctx(), domain.CreateConversationInput{
		ID: "msg-conv", UserID: "user-1", Title: "Msg Test", Mode: "direct", ResponseMode: "separate",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// Create messages.
	m1, err := msg.Create(ctx(), domain.CreateMessageInput{
		ConversationID: "msg-conv", Role: "user", SenderType: "user",
		Content: "Hello", Status: "sent",
	})
	if err != nil {
		t.Fatalf("Create m1: %v", err)
	}
	if m1.Sequence != 1 {
		t.Errorf("m1 sequence: expected 1, got %d", m1.Sequence)
	}

	m2, err := msg.Create(ctx(), domain.CreateMessageInput{
		ConversationID: "msg-conv", Role: "assistant", SenderType: "agent",
		Content: "Hi there", Status: "sent",
	})
	if err != nil {
		t.Fatalf("Create m2: %v", err)
	}
	if m2.Sequence != 2 {
		t.Errorf("m2 sequence: expected 2, got %d", m2.Sequence)
	}

	// List.
	list, err := msg.List(ctx(), "msg-conv", domain.Pagination{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("list: expected 2, got %d", len(list))
	}
}

func TestMessageIdempotency(t *testing.T) {
	_, conv, msg, _, _, _ := newTestRepos(t)

	_, err := conv.Create(ctx(), domain.CreateConversationInput{
		ID: "idem-conv", UserID: "user-1", Title: "Idem", Mode: "direct", ResponseMode: "separate",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	// First create with clientMessageId.
	m1, err := msg.Create(ctx(), domain.CreateMessageInput{
		ConversationID:  "idem-conv",
		Role:            "user",
		SenderType:      "user",
		Content:         "First",
		Status:          "sent",
		ClientMessageID: "client-id-1",
	})
	if err != nil {
		t.Fatalf("Create m1: %v", err)
	}

	// Second create with same clientMessageId should be idempotent.
	m2, err := msg.Create(ctx(), domain.CreateMessageInput{
		ConversationID:  "idem-conv",
		Role:            "user",
		SenderType:      "user",
		Content:         "Second (should be ignored)",
		Status:          "sent",
		ClientMessageID: "client-id-1",
	})
	if err != nil {
		t.Fatalf("Create m2: %v", err)
	}
	if m2.ID != m1.ID {
		t.Errorf("idempotent create: expected same ID %s, got %s", m1.ID, m2.ID)
	}

	// Only one message should exist.
	list, err := msg.List(ctx(), "idem-conv", domain.Pagination{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("after idempotent create: expected 1, got %d", len(list))
	}
}

func TestMessageSoftDelete(t *testing.T) {
	_, conv, msg, _, _, _ := newTestRepos(t)

	_, err := conv.Create(ctx(), domain.CreateConversationInput{
		ID: "softdel-conv", UserID: "user-1", Title: "SoftDel", Mode: "direct", ResponseMode: "separate",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	m, err := msg.Create(ctx(), domain.CreateMessageInput{
		ConversationID: "softdel-conv", Role: "user", SenderType: "user",
		Content: "To Delete", Status: "sent",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Soft delete.
	if err := msg.SoftDelete(ctx(), m.ID, "softdel-conv"); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	// Deleted message should not appear in list.
	list, err := msg.List(ctx(), "softdel-conv", domain.Pagination{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("List after delete: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("after soft delete: expected 0, got %d", len(list))
	}
}

// ---------------------------------------------------------------------------
// RunRepository tests
// ---------------------------------------------------------------------------

func TestRunCreateAndGet(t *testing.T) {
	_, conv, _, run, _, _ := newTestRepos(t)

	_, err := conv.Create(ctx(), domain.CreateConversationInput{
		ID: "run-conv", UserID: "user-1", Title: "Run Test", Mode: "direct", ResponseMode: "separate",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	r, err := run.Create(ctx(), domain.CreateRunInput{
		ID: "run-1", ConversationID: "run-conv", Mode: "direct", Status: "pending",
	})
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if r.Status != "pending" {
		t.Errorf("status: expected pending, got %s", r.Status)
	}

	// Get.
	fetched, err := run.Get(ctx(), "run-1")
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if fetched.ID != "run-1" {
		t.Errorf("id: expected run-1, got %s", fetched.ID)
	}
}

func TestRunStateTransitions(t *testing.T) {
	_, conv, _, runRepo, _, _ := newTestRepos(t)

	_, err := conv.Create(ctx(), domain.CreateConversationInput{
		ID: "trans-conv", UserID: "user-1", Title: "Trans", Mode: "direct", ResponseMode: "separate",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	_, err = runRepo.Create(ctx(), domain.CreateRunInput{
		ID: "trans-1", ConversationID: "trans-conv", Mode: "direct", Status: "pending",
	})
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	// Valid: pending -> planning.
	if err := runRepo.CompareAndSetStatus(ctx(), "trans-1", "pending", "planning", "", ""); err != nil {
		t.Fatalf("pending->planning: %v", err)
	}

	// Valid: planning -> executing.
	if err := runRepo.CompareAndSetStatus(ctx(), "trans-1", "planning", "executing", "", ""); err != nil {
		t.Fatalf("planning->executing: %v", err)
	}

	// Invalid: executing -> pending (can't go backward).
	if err := runRepo.CompareAndSetStatus(ctx(), "trans-1", "executing", "pending", "", ""); err != domain.ErrInvalidArgument {
		t.Errorf("executing->pending: expected ErrInvalidArgument, got %v", err)
	}

	// Stale CAS (wrong current state — run is actually "executing", not "planning").
	if err := runRepo.CompareAndSetStatus(ctx(), "trans-1", "planning", "executing", "", ""); err != domain.ErrConflict {
		t.Errorf("stale CAS: expected ErrConflict, got %v", err)
	}

	// Valid: executing -> completed.
	if err := runRepo.CompareAndSetStatus(ctx(), "trans-1", "executing", "completed", "", ""); err != nil {
		t.Fatalf("executing->completed: %v", err)
	}

	// Terminal state cannot transition further.
	if err := runRepo.CompareAndSetStatus(ctx(), "trans-1", "completed", "executing", "", ""); err != domain.ErrInvalidArgument {
		t.Errorf("completed->executing: expected ErrInvalidArgument, got %v", err)
	}
}

func TestRunCancel(t *testing.T) {
	_, conv, _, runRepo, _, _ := newTestRepos(t)

	_, err := conv.Create(ctx(), domain.CreateConversationInput{
		ID: "cancel-conv", UserID: "user-1", Title: "Cancel", Mode: "direct", ResponseMode: "separate",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	_, err = runRepo.Create(ctx(), domain.CreateRunInput{
		ID: "cancel-1", ConversationID: "cancel-conv", Mode: "direct", Status: "executing",
	})
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	if err := runRepo.Cancel(ctx(), "cancel-1"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	fetched, err := runRepo.Get(ctx(), "cancel-1")
	if err != nil {
		t.Fatalf("Get canceled: %v", err)
	}
	if fetched.Status != "canceled" {
		t.Errorf("status: expected canceled, got %s", fetched.Status)
	}
}

// ---------------------------------------------------------------------------
// EventRepository tests
// ---------------------------------------------------------------------------

func TestEventAppendAndReplay(t *testing.T) {
	_, conv, _, runRepo, evt, _ := newTestRepos(t)

	_, err := conv.Create(ctx(), domain.CreateConversationInput{
		ID: "evt-conv", UserID: "user-1", Title: "Event", Mode: "direct", ResponseMode: "separate",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	_, err = runRepo.Create(ctx(), domain.CreateRunInput{
		ID: "evt-run", ConversationID: "evt-conv", Mode: "direct", Status: "executing",
	})
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	// Append events.
	e1, err := evt.Append(ctx(), domain.CreateEventInput{
		ConversationID: "evt-conv", RunID: "evt-run", EventType: "text_delta", Payload: `{"delta":"hello"}`,
	})
	if err != nil {
		t.Fatalf("Append e1: %v", err)
	}
	if e1.Sequence != 1 {
		t.Errorf("e1 sequence: expected 1, got %d", e1.Sequence)
	}

	e2, err := evt.Append(ctx(), domain.CreateEventInput{
		ConversationID: "evt-conv", RunID: "evt-run", EventType: "text_delta", Payload: `{"delta":" world"}`,
	})
	if err != nil {
		t.Fatalf("Append e2: %v", err)
	}
	if e2.Sequence != 2 {
		t.Errorf("e2 sequence: expected 2, got %d", e2.Sequence)
	}

	// Replay after sequence 0.
	events, err := evt.ListAfter(ctx(), "evt-run", 0, 10)
	if err != nil {
		t.Fatalf("ListAfter(0): %v", err)
	}
	if len(events) != 2 {
		t.Errorf("replay all: expected 2, got %d", len(events))
	}

	// Replay after sequence 1.
	events, err = evt.ListAfter(ctx(), "evt-run", 1, 10)
	if err != nil {
		t.Fatalf("ListAfter(1): %v", err)
	}
	if len(events) != 1 {
		t.Errorf("replay after 1: expected 1, got %d", len(events))
	}
	if events[0].Sequence != 2 {
		t.Errorf("replay after 1: expected sequence 2, got %d", events[0].Sequence)
	}
}

func TestEventPayloadValidation(t *testing.T) {
	_, conv, _, runRepo, evt, _ := newTestRepos(t)

	_, err := conv.Create(ctx(), domain.CreateConversationInput{
		ID: "pay-conv", UserID: "user-1", Title: "Payload", Mode: "direct", ResponseMode: "separate",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	_, err = runRepo.Create(ctx(), domain.CreateRunInput{
		ID: "pay-run", ConversationID: "pay-conv", Mode: "direct", Status: "executing",
	})
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	// Invalid JSON payload should be rejected.
	_, err = evt.Append(ctx(), domain.CreateEventInput{
		ConversationID: "pay-conv", RunID: "pay-run", EventType: "test", Payload: "not json",
	})
	if err != domain.ErrInvalidArgument {
		t.Errorf("invalid JSON: expected ErrInvalidArgument, got %v", err)
	}

	// Empty payload should default to {}.
	e, err := evt.Append(ctx(), domain.CreateEventInput{
		ConversationID: "pay-conv", RunID: "pay-run", EventType: "test", Payload: "",
	})
	if err != nil {
		t.Fatalf("Append empty: %v", err)
	}
	if e.Payload != "{}" {
		t.Errorf("empty payload: expected '{}', got '%s'", e.Payload)
	}
}

// ---------------------------------------------------------------------------
// Repository rebuild test (restart persistence)
// ---------------------------------------------------------------------------

func TestRepositoryRebuild(t *testing.T) {
	d, conv, msg, _, _, _ := newTestRepos(t)

	_, err := conv.Create(ctx(), domain.CreateConversationInput{
		ID: "rebuild-conv", UserID: "user-1", Title: "Rebuild", Mode: "direct", ResponseMode: "separate",
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	_, err = msg.Create(ctx(), domain.CreateMessageInput{
		ConversationID: "rebuild-conv", Role: "user", SenderType: "user",
		Content: "Before rebuild", Status: "sent",
	})
	if err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}

	// Rebuild repos on same database (simulates restart).
	q2 := db.New(d)
	conv2 := repository.NewConversationRepo(q2)
	msg2 := repository.NewMessageRepo(q2, d)

	// Data should still be there.
	fetched, err := conv2.Get(ctx(), "user-1", "rebuild-conv")
	if err != nil {
		t.Fatalf("Get after rebuild: %v", err)
	}
	if fetched.Title != "Rebuild" {
		t.Errorf("title after rebuild: expected 'Rebuild', got %s", fetched.Title)
	}

	list, err := msg2.List(ctx(), "rebuild-conv", domain.Pagination{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("List after rebuild: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("list after rebuild: expected 1, got %d", len(list))
	}
}
