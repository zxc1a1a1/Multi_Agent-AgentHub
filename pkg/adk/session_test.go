package adk

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

var errSessionNotFound = errors.New("session not found")

type mockSessionService struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	nextID   int
}

func newMockSessionService() *mockSessionService {
	return &mockSessionService{
		sessions: make(map[string]*Session),
		nextID:   1,
	}
}

func (m *mockSessionService) Create(ctx context.Context, userID string, initialState map[string]any) (*Session, error) {
	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	id := fmt.Sprintf("session-%d", m.nextID)
	m.nextID++

	session := &Session{
		ID:        id,
		UserID:    userID,
		Events:    make([]Event, 0),
		State:     NewSessionState(initialState),
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.sessions[id] = session
	return session, nil
}

func (m *mockSessionService) GetOrCreate(ctx context.Context, id string) (*Session, error) {
	session, err := m.Get(ctx, id)
	if err == nil {
		return session, nil
	}
	if !errors.Is(err, errSessionNotFound) {
		return nil, err
	}

	now := time.Now()
	session = &Session{
		ID:        id,
		UserID:    id,
		Events:    make([]Event, 0),
		State:     NewSessionState(nil),
		CreatedAt: now,
		UpdatedAt: now,
	}

	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()

	return session, nil
}

func (m *mockSessionService) Get(ctx context.Context, id string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[id]
	if !ok {
		return nil, errSessionNotFound
	}
	return session, nil
}

func (m *mockSessionService) AppendEvent(ctx context.Context, sessionID string, event Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return errSessionNotFound
	}

	session.Events = append(session.Events, event)
	session.UpdatedAt = time.Now()
	return nil
}

func (m *mockSessionService) UpdateState(ctx context.Context, sessionID string, delta map[string]any) error {
	m.mu.RLock()
	session, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return errSessionNotFound
	}

	for key, value := range delta {
		session.State.Set(key, value)
	}

	m.mu.Lock()
	session.UpdatedAt = time.Now()
	m.mu.Unlock()
	return nil
}

func (m *mockSessionService) List(ctx context.Context, userID string) ([]*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Session, 0)
	for _, session := range m.sessions {
		if session.UserID == userID {
			result = append(result, session)
		}
	}
	return result, nil
}

func (m *mockSessionService) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.sessions[id]; !ok {
		return errSessionNotFound
	}
	delete(m.sessions, id)
	return nil
}

func TestNewSessionState_NilInit(t *testing.T) {
	state := NewSessionState(nil)
	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if value, ok := state.Get("missing"); ok || value != nil {
		t.Fatalf("expected missing key, got ok=%v value=%#v", ok, value)
	}
	if got := state.All(); len(got) != 0 {
		t.Fatalf("expected empty state, got: %#v", got)
	}
}

func TestSessionState_GetSet(t *testing.T) {
	state := NewSessionState(nil)
	state.Set("phase", "1.5")
	state.Set("done", true)

	phase, ok := state.Get("phase")
	if !ok || phase != "1.5" {
		t.Fatalf("unexpected phase: ok=%v value=%#v", ok, phase)
	}
	done, ok := state.Get("done")
	if !ok || done != true {
		t.Fatalf("unexpected done: ok=%v value=%#v", ok, done)
	}
}

func TestSessionState_AllReturnsCopy(t *testing.T) {
	state := NewSessionState(map[string]any{
		"phase": "1.4",
		"done":  false,
	})

	snapshot := state.All()
	snapshot["phase"] = "changed-outside"
	snapshot["new"] = 123

	phase, ok := state.Get("phase")
	if !ok || phase != "1.4" {
		t.Fatalf("internal state should not change, got: %#v", phase)
	}
	if _, ok := state.Get("new"); ok {
		t.Fatal("internal state should not include external key")
	}
}

func TestSessionState_ConcurrentAccess(t *testing.T) {
	state := NewSessionState(nil)
	const workers = 64
	const rounds = 100

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(worker int) {
			defer wg.Done()
			key := fmt.Sprintf("worker-%d", worker)
			for round := 0; round < rounds; round++ {
				state.Set(key, round)
				value, ok := state.Get(key)
				if !ok {
					t.Errorf("worker %d missing key", worker)
					return
				}
				if _, ok := value.(int); !ok {
					t.Errorf("worker %d unexpected value type: %T", worker, value)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	all := state.All()
	if len(all) != workers {
		t.Fatalf("unexpected key count: got=%d want=%d", len(all), workers)
	}
}

func TestSession_Construction(t *testing.T) {
	now := time.Now()
	session := Session{
		ID:     "session-1",
		UserID: "user-1",
		Events: []Event{
			{
				ID:      "event-1",
				Author:  "tester",
				Content: &Content{Role: RoleUser, Parts: []Part{TextPart{Text: "hello"}}},
			},
		},
		State:     NewSessionState(map[string]any{"phase": "1.5"}),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if session.ID != "session-1" {
		t.Fatalf("unexpected session id: %q", session.ID)
	}
	if session.UserID != "user-1" {
		t.Fatalf("unexpected user id: %q", session.UserID)
	}
	if len(session.Events) != 1 {
		t.Fatalf("unexpected events count: %d", len(session.Events))
	}
	if session.State == nil {
		t.Fatal("session state must not be nil")
	}
	if session.CreatedAt.IsZero() || session.UpdatedAt.IsZero() {
		t.Fatal("timestamps must be non-zero")
	}
}

func TestSessionService_InterfaceCompliance(t *testing.T) {
	var _ SessionService = (*mockSessionService)(nil)
}

func TestMockSessionService_CreateAndGet(t *testing.T) {
	service := newMockSessionService()
	ctx := context.Background()

	created, err := service.Create(ctx, "user-1", map[string]any{"phase": "1.5"})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty session ID")
	}
	if created.UserID != "user-1" {
		t.Fatalf("unexpected user id: %q", created.UserID)
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatal("expected non-zero timestamps")
	}

	got, err := service.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	phase, ok := got.State.Get("phase")
	if !ok || phase != "1.5" {
		t.Fatalf("unexpected phase in state: ok=%v value=%#v", ok, phase)
	}
}

func TestMockSessionService_AppendEvent(t *testing.T) {
	service := newMockSessionService()
	ctx := context.Background()

	session, err := service.Create(ctx, "user-1", nil)
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	before := session.UpdatedAt

	event := Event{
		ID:      "event-1",
		Author:  "tester",
		Content: &Content{Role: RoleAssistant, Parts: []Part{TextPart{Text: "reply"}}},
	}
	if err := service.AppendEvent(ctx, session.ID, event); err != nil {
		t.Fatalf("append event error: %v", err)
	}

	got, err := service.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if len(got.Events) != 1 {
		t.Fatalf("unexpected events count: %d", len(got.Events))
	}
	if got.Events[0].ID != "event-1" {
		t.Fatalf("unexpected event id: %q", got.Events[0].ID)
	}
	if got.UpdatedAt.Before(before) {
		t.Fatal("updatedAt should not go backwards")
	}
}

func TestMockSessionService_UpdateState(t *testing.T) {
	service := newMockSessionService()
	ctx := context.Background()

	session, err := service.Create(ctx, "user-1", map[string]any{
		"phase": "1.4",
		"keep":  "yes",
	})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}

	if err := service.UpdateState(ctx, session.ID, map[string]any{
		"phase": "1.5",
		"done":  true,
	}); err != nil {
		t.Fatalf("update state error: %v", err)
	}

	got, err := service.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	all := got.State.All()
	if all["phase"] != "1.5" {
		t.Fatalf("unexpected phase: %#v", all["phase"])
	}
	if all["keep"] != "yes" {
		t.Fatalf("expected existing key to remain, got: %#v", all["keep"])
	}
	if all["done"] != true {
		t.Fatalf("unexpected done: %#v", all["done"])
	}
}

func TestMockSessionService_ListAndDelete(t *testing.T) {
	service := newMockSessionService()
	ctx := context.Background()

	s1, err := service.Create(ctx, "user-1", nil)
	if err != nil {
		t.Fatalf("create s1 error: %v", err)
	}
	_, err = service.Create(ctx, "user-1", nil)
	if err != nil {
		t.Fatalf("create s2 error: %v", err)
	}
	_, err = service.Create(ctx, "user-2", nil)
	if err != nil {
		t.Fatalf("create s3 error: %v", err)
	}

	user1Sessions, err := service.List(ctx, "user-1")
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(user1Sessions) != 2 {
		t.Fatalf("unexpected user-1 session count: %d", len(user1Sessions))
	}

	if err := service.Delete(ctx, s1.ID); err != nil {
		t.Fatalf("delete error: %v", err)
	}
	if _, err := service.Get(ctx, s1.ID); !errors.Is(err, errSessionNotFound) {
		t.Fatalf("expected errSessionNotFound after delete, got: %v", err)
	}
}
