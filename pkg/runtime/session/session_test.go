package session

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

var (
	_ adk.SessionService = (*SQLSessionService)(nil)

	fakeDriverSeq uint64
)

type fakeSessionRow struct {
	ID        string
	UserID    string
	StateJSON []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

type fakeEventRow struct {
	ID          string
	SessionID   string
	Seq         int64
	Author      string
	ContentJSON []byte
	ActionsJSON []byte
	Partial     bool
	Final       bool
	Timestamp   time.Time
}

type fakeSessionStore struct {
	mu       sync.Mutex
	sessions map[string]fakeSessionRow
	events   map[string][]fakeEventRow
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{
		sessions: make(map[string]fakeSessionRow),
		events:   make(map[string][]fakeEventRow),
	}
}

type fakeDriver struct {
	store *fakeSessionStore
}

func (d *fakeDriver) Open(string) (driver.Conn, error) {
	return &fakeConn{store: d.store}, nil
}

type fakeConn struct {
	store *fakeSessionStore
}

var (
	_ driver.Conn           = (*fakeConn)(nil)
	_ driver.ExecerContext  = (*fakeConn)(nil)
	_ driver.QueryerContext = (*fakeConn)(nil)
)

func (c *fakeConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (c *fakeConn) Close() error {
	return nil
}

func (c *fakeConn) Begin() (driver.Tx, error) {
	return nil, errors.New("begin is not supported")
}

func (c *fakeConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	q := normalizeQuery(query)

	switch {
	case strings.HasPrefix(q, "insert into sessions"):
		id := argString(args, 0)
		userID := argString(args, 1)
		stateJSON := argBytes(args, 2)
		createdAt := argTime(args, 3)
		updatedAt := argTime(args, 4)

		c.store.mu.Lock()
		defer c.store.mu.Unlock()

		if _, exists := c.store.sessions[id]; exists {
			return nil, fmt.Errorf("duplicate session id: %s", id)
		}
		c.store.sessions[id] = fakeSessionRow{
			ID:        id,
			UserID:    userID,
			StateJSON: append([]byte(nil), stateJSON...),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}
		return driver.RowsAffected(1), nil

	case strings.HasPrefix(q, "insert into session_events"):
		sessionID := argString(args, 1)

		c.store.mu.Lock()
		defer c.store.mu.Unlock()

		if _, exists := c.store.sessions[sessionID]; !exists {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}

		row := fakeEventRow{
			ID:          argString(args, 0),
			SessionID:   sessionID,
			Seq:         argInt64(args, 2),
			Author:      argString(args, 3),
			ContentJSON: append([]byte(nil), argBytes(args, 4)...),
			ActionsJSON: append([]byte(nil), argBytes(args, 5)...),
			Partial:     argBool(args, 6),
			Final:       argBool(args, 7),
			Timestamp:   argTime(args, 8),
		}
		c.store.events[sessionID] = append(c.store.events[sessionID], row)
		return driver.RowsAffected(1), nil

	case strings.HasPrefix(q, "update sessions set updated_at = ? where id = ?"):
		updatedAt := argTime(args, 0)
		sessionID := argString(args, 1)

		c.store.mu.Lock()
		defer c.store.mu.Unlock()

		row, exists := c.store.sessions[sessionID]
		if !exists {
			return driver.RowsAffected(0), nil
		}
		row.UpdatedAt = updatedAt
		c.store.sessions[sessionID] = row
		return driver.RowsAffected(1), nil

	case strings.HasPrefix(q, "update sessions set state_json = ?, updated_at = ? where id = ?"):
		stateJSON := argBytes(args, 0)
		updatedAt := argTime(args, 1)
		sessionID := argString(args, 2)

		c.store.mu.Lock()
		defer c.store.mu.Unlock()

		row, exists := c.store.sessions[sessionID]
		if !exists {
			return driver.RowsAffected(0), nil
		}
		row.StateJSON = append([]byte(nil), stateJSON...)
		row.UpdatedAt = updatedAt
		c.store.sessions[sessionID] = row
		return driver.RowsAffected(1), nil

	case strings.HasPrefix(q, "delete from session_events where session_id = ?"):
		sessionID := argString(args, 0)

		c.store.mu.Lock()
		defer c.store.mu.Unlock()

		affected := int64(len(c.store.events[sessionID]))
		delete(c.store.events, sessionID)
		return driver.RowsAffected(affected), nil

	case strings.HasPrefix(q, "delete from sessions where id = ?"):
		sessionID := argString(args, 0)

		c.store.mu.Lock()
		defer c.store.mu.Unlock()

		if _, exists := c.store.sessions[sessionID]; !exists {
			return driver.RowsAffected(0), nil
		}
		delete(c.store.sessions, sessionID)
		delete(c.store.events, sessionID)
		return driver.RowsAffected(1), nil
	}

	return nil, fmt.Errorf("unsupported exec query: %q", q)
}

func (c *fakeConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	q := normalizeQuery(query)

	switch {
	case strings.HasPrefix(q, "select id, user_id, state_json, created_at, updated_at from sessions where id = ?"):
		sessionID := argString(args, 0)

		c.store.mu.Lock()
		row, exists := c.store.sessions[sessionID]
		c.store.mu.Unlock()

		rows := &fakeRows{
			columns: []string{"id", "user_id", "state_json", "created_at", "updated_at"},
		}
		if exists {
			rows.data = append(rows.data, []driver.Value{
				row.ID,
				row.UserID,
				append([]byte(nil), row.StateJSON...),
				row.CreatedAt,
				row.UpdatedAt,
			})
		}
		return rows, nil

	case strings.HasPrefix(q, "select id from sessions where id = ?"):
		sessionID := argString(args, 0)

		c.store.mu.Lock()
		_, exists := c.store.sessions[sessionID]
		c.store.mu.Unlock()

		rows := &fakeRows{columns: []string{"id"}}
		if exists {
			rows.data = append(rows.data, []driver.Value{sessionID})
		}
		return rows, nil

	case strings.HasPrefix(q, "select state_json from sessions where id = ?"):
		sessionID := argString(args, 0)

		c.store.mu.Lock()
		row, exists := c.store.sessions[sessionID]
		c.store.mu.Unlock()

		rows := &fakeRows{columns: []string{"state_json"}}
		if exists {
			rows.data = append(rows.data, []driver.Value{append([]byte(nil), row.StateJSON...)})
		}
		return rows, nil

	case strings.HasPrefix(q, "select coalesce(max(seq), 0) + 1 from session_events where session_id = ?"):
		sessionID := argString(args, 0)

		c.store.mu.Lock()
		events := append([]fakeEventRow(nil), c.store.events[sessionID]...)
		c.store.mu.Unlock()

		var maxSeq int64
		for _, row := range events {
			if row.Seq > maxSeq {
				maxSeq = row.Seq
			}
		}

		return &fakeRows{
			columns: []string{"next_seq"},
			data: [][]driver.Value{
				{maxSeq + 1},
			},
		}, nil

	case strings.HasPrefix(q, "select id, seq, author, content_json, actions_json, partial, final, timestamp from session_events where session_id = ? order by seq asc"):
		sessionID := argString(args, 0)

		c.store.mu.Lock()
		events := append([]fakeEventRow(nil), c.store.events[sessionID]...)
		c.store.mu.Unlock()

		sort.Slice(events, func(i, j int) bool {
			if events[i].Seq == events[j].Seq {
				return events[i].ID < events[j].ID
			}
			return events[i].Seq < events[j].Seq
		})

		rows := &fakeRows{
			columns: []string{"id", "seq", "author", "content_json", "actions_json", "partial", "final", "timestamp"},
			data:    make([][]driver.Value, 0, len(events)),
		}
		for _, row := range events {
			rows.data = append(rows.data, []driver.Value{
				row.ID,
				row.Seq,
				row.Author,
				append([]byte(nil), row.ContentJSON...),
				append([]byte(nil), row.ActionsJSON...),
				row.Partial,
				row.Final,
				row.Timestamp,
			})
		}
		return rows, nil

	case strings.HasPrefix(q, "select id, user_id, state_json, created_at, updated_at from sessions where user_id = ? order by updated_at desc, id desc"):
		userID := argString(args, 0)

		c.store.mu.Lock()
		rowsByUser := make([]fakeSessionRow, 0)
		for _, row := range c.store.sessions {
			if row.UserID == userID {
				rowsByUser = append(rowsByUser, row)
			}
		}
		c.store.mu.Unlock()

		sort.Slice(rowsByUser, func(i, j int) bool {
			if rowsByUser[i].UpdatedAt.Equal(rowsByUser[j].UpdatedAt) {
				return rowsByUser[i].ID > rowsByUser[j].ID
			}
			return rowsByUser[i].UpdatedAt.After(rowsByUser[j].UpdatedAt)
		})

		rows := &fakeRows{
			columns: []string{"id", "user_id", "state_json", "created_at", "updated_at"},
			data:    make([][]driver.Value, 0, len(rowsByUser)),
		}
		for _, row := range rowsByUser {
			rows.data = append(rows.data, []driver.Value{
				row.ID,
				row.UserID,
				append([]byte(nil), row.StateJSON...),
				row.CreatedAt,
				row.UpdatedAt,
			})
		}
		return rows, nil
	}

	return nil, fmt.Errorf("unsupported query: %q", q)
}

type fakeRows struct {
	columns []string
	data    [][]driver.Value
	index   int
}

func (r *fakeRows) Columns() []string {
	return append([]string(nil), r.columns...)
}

func (r *fakeRows) Close() error {
	return nil
}

func (r *fakeRows) Next(dest []driver.Value) error {
	if r.index >= len(r.data) {
		return io.EOF
	}

	row := r.data[r.index]
	r.index++
	for i := range row {
		dest[i] = row[i]
	}
	return nil
}

func normalizeQuery(query string) string {
	return strings.Join(strings.Fields(strings.ToLower(query)), " ")
}

func argString(args []driver.NamedValue, idx int) string {
	if idx >= len(args) {
		return ""
	}
	switch value := args[idx].Value.(type) {
	case string:
		return value
	case []byte:
		return string(value)
	default:
		return fmt.Sprint(value)
	}
}

func argBytes(args []driver.NamedValue, idx int) []byte {
	if idx >= len(args) {
		return nil
	}
	switch value := args[idx].Value.(type) {
	case []byte:
		return append([]byte(nil), value...)
	case string:
		return []byte(value)
	default:
		return []byte(fmt.Sprint(value))
	}
}

func argInt64(args []driver.NamedValue, idx int) int64 {
	if idx >= len(args) {
		return 0
	}
	switch value := args[idx].Value.(type) {
	case int64:
		return value
	case int:
		return int64(value)
	case int32:
		return int64(value)
	case float64:
		return int64(value)
	default:
		return 0
	}
}

func argBool(args []driver.NamedValue, idx int) bool {
	if idx >= len(args) {
		return false
	}
	switch value := args[idx].Value.(type) {
	case bool:
		return value
	default:
		return false
	}
}

func argTime(args []driver.NamedValue, idx int) time.Time {
	if idx >= len(args) {
		return time.Time{}
	}
	switch value := args[idx].Value.(type) {
	case time.Time:
		return value
	default:
		return time.Time{}
	}
}

func newFakeSQLDB(t *testing.T) (*sql.DB, *fakeSessionStore) {
	t.Helper()

	store := newFakeSessionStore()
	driverName := fmt.Sprintf("session-fake-driver-%d", atomic.AddUint64(&fakeDriverSeq, 1))
	sql.Register(driverName, &fakeDriver{store: store})

	db, err := sql.Open(driverName, "ignored")
	if err != nil {
		t.Fatalf("open fake sql db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db, store
}

func newServiceForTest(t *testing.T) (*SQLSessionService, *fakeSessionStore) {
	t.Helper()

	db, store := newFakeSQLDB(t)
	svc, err := NewSQLSessionService(db)
	if err != nil {
		t.Fatalf("new SQLSessionService: %v", err)
	}
	return svc, store
}

func TestNewSQLSessionService_NilDB(t *testing.T) {
	svc, err := NewSQLSessionService(nil)
	if err == nil {
		t.Fatal("expected error for nil DB")
	}
	if svc != nil {
		t.Fatal("expected nil service")
	}
}

func TestSQLSessionService_InterfaceCompliance(t *testing.T) {
	svc, _ := newServiceForTest(t)
	var service adk.SessionService = svc
	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestSQLSessionService_CreateAndGet(t *testing.T) {
	svc, _ := newServiceForTest(t)

	initialState := map[string]any{
		"phase": "4.1",
		"count": 1,
	}

	created, err := svc.Create(context.Background(), "user-1", initialState)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty session id")
	}
	if created.State == nil {
		t.Fatal("expected non-nil state")
	}

	initialState["phase"] = "mutated"

	got, err := svc.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if got.UserID != "user-1" {
		t.Fatalf("unexpected user id: %q", got.UserID)
	}
	value, ok := got.State.Get("phase")
	if !ok || value != "4.1" {
		t.Fatalf("state copy mismatch: ok=%v value=%#v", ok, value)
	}
}

func TestSQLSessionService_GetWithEvents(t *testing.T) {
	svc, _ := newServiceForTest(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, "user-1", map[string]any{"phase": "4.1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	assistantEvent := adk.Event{
		ID:     "evt-1",
		Author: "assistant",
		Content: &adk.Content{
			Role: adk.RoleAssistant,
			Parts: []adk.Part{
				adk.TextPart{Text: "planning"},
				adk.ToolCallPart{
					ID:        "call-1",
					Name:      "read_file",
					Arguments: json.RawMessage(`{"path":"README.md"}`),
				},
			},
		},
		Actions: &adk.EventActions{
			StateDelta:    map[string]any{"step": "analysis"},
			TransferAgent: "code-agent",
			ArtifactDelta: []adk.Artifact{
				{
					Type:    "markdown",
					Title:   "result",
					Content: "done",
					Metadata: map[string]string{
						"mimeType": "text/markdown",
					},
				},
			},
		},
		Timestamp: time.Now().UTC(),
	}
	if err := svc.AppendEvent(ctx, created.ID, assistantEvent); err != nil {
		t.Fatalf("append event: %v", err)
	}

	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get session with events: %v", err)
	}
	if len(got.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got.Events))
	}
	event := got.Events[0]
	if event.ID != "evt-1" || event.Author != "assistant" {
		t.Fatalf("unexpected event identity: %+v", event)
	}
	if event.Content == nil || event.Content.Role != adk.RoleAssistant {
		t.Fatalf("unexpected event content: %#v", event.Content)
	}
	if len(event.Content.Parts) != 2 {
		t.Fatalf("unexpected parts length: %d", len(event.Content.Parts))
	}
	if _, ok := event.Content.Parts[0].(adk.TextPart); !ok {
		t.Fatalf("expected first part text, got %T", event.Content.Parts[0])
	}
	callPart, ok := event.Content.Parts[1].(adk.ToolCallPart)
	if !ok {
		t.Fatalf("expected second part tool call, got %T", event.Content.Parts[1])
	}
	if callPart.ID != "call-1" || callPart.Name != "read_file" {
		t.Fatalf("unexpected tool call: %+v", callPart)
	}

	if event.Actions == nil {
		t.Fatal("expected event actions")
	}
	if event.Actions.TransferAgent != "code-agent" {
		t.Fatalf("unexpected transfer agent: %q", event.Actions.TransferAgent)
	}
	if len(event.Actions.ArtifactDelta) != 1 {
		t.Fatalf("unexpected artifact delta length: %d", len(event.Actions.ArtifactDelta))
	}
}

func TestSQLSessionService_GetNotFound(t *testing.T) {
	svc, _ := newServiceForTest(t)
	_, err := svc.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestSQLSessionService_AppendEvent(t *testing.T) {
	svc, _ := newServiceForTest(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, "user-1", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	before := created.UpdatedAt

	event := adk.Event{
		ID:        "evt-1",
		Author:    "assistant",
		Content:   &adk.Content{Role: adk.RoleAssistant, Parts: []adk.Part{adk.TextPart{Text: "ok"}}},
		Partial:   true,
		Final:     false,
		Timestamp: time.Now(),
	}
	if err := svc.AppendEvent(ctx, created.ID, event); err != nil {
		t.Fatalf("append: %v", err)
	}

	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got.Events))
	}
	if got.Events[0].ID != "evt-1" || !got.Events[0].Partial {
		t.Fatalf("unexpected event: %+v", got.Events[0])
	}
	if !got.UpdatedAt.After(before) && !got.UpdatedAt.Equal(before) {
		t.Fatalf("updatedAt went backwards: before=%s after=%s", before, got.UpdatedAt)
	}
}

func TestSQLSessionService_AppendEventNotFound(t *testing.T) {
	svc, _ := newServiceForTest(t)
	err := svc.AppendEvent(context.Background(), "missing", adk.Event{ID: "evt-1"})
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestSQLSessionService_UpdateState(t *testing.T) {
	svc, _ := newServiceForTest(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, "user-1", map[string]any{"phase": "4.1", "keep": true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := svc.UpdateState(ctx, created.ID, map[string]any{"phase": "4.2", "done": true}); err != nil {
		t.Fatalf("update state: %v", err)
	}

	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	all := got.State.All()
	if all["phase"] != "4.2" {
		t.Fatalf("unexpected phase: %#v", all["phase"])
	}
	if all["keep"] != true {
		t.Fatalf("unexpected keep: %#v", all["keep"])
	}
	if all["done"] != true {
		t.Fatalf("unexpected done: %#v", all["done"])
	}
}

func TestSQLSessionService_UpdateStateNotFound(t *testing.T) {
	svc, _ := newServiceForTest(t)
	err := svc.UpdateState(context.Background(), "missing", map[string]any{"done": true})
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestSQLSessionService_List(t *testing.T) {
	svc, _ := newServiceForTest(t)
	ctx := context.Background()

	if _, err := svc.Create(ctx, "user-a", map[string]any{"n": 1}); err != nil {
		t.Fatalf("create user-a #1: %v", err)
	}
	if _, err := svc.Create(ctx, "user-a", map[string]any{"n": 2}); err != nil {
		t.Fatalf("create user-a #2: %v", err)
	}
	if _, err := svc.Create(ctx, "user-b", map[string]any{"n": 3}); err != nil {
		t.Fatalf("create user-b: %v", err)
	}

	list, err := svc.List(ctx, "user-a")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 sessions for user-a, got %d", len(list))
	}
	for _, sess := range list {
		if sess.UserID != "user-a" {
			t.Fatalf("unexpected user id in list: %q", sess.UserID)
		}
	}
}

func TestSQLSessionService_Delete(t *testing.T) {
	svc, store := newServiceForTest(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, "user-a", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.AppendEvent(ctx, created.ID, adk.Event{
		ID:      "evt-1",
		Author:  "assistant",
		Content: &adk.Content{Role: adk.RoleAssistant, Parts: []adk.Part{adk.TextPart{Text: "hi"}}},
	}); err != nil {
		t.Fatalf("append: %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := svc.Get(ctx, created.ID); err == nil {
		t.Fatal("expected get not found after delete")
	}

	store.mu.Lock()
	_, sessionExists := store.sessions[created.ID]
	_, eventExists := store.events[created.ID]
	store.mu.Unlock()
	if sessionExists || eventExists {
		t.Fatalf("expected session and events deleted, got session=%v events=%v", sessionExists, eventExists)
	}
}

func TestSQLSessionService_DeleteNotFound(t *testing.T) {
	svc, _ := newServiceForTest(t)
	err := svc.Delete(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestContentDTO_TextPartRoundTrip(t *testing.T) {
	content := &adk.Content{
		Role:  adk.RoleAssistant,
		Parts: []adk.Part{adk.TextPart{Text: "hello"}},
	}

	data, err := marshalContent(content)
	if err != nil {
		t.Fatalf("marshal content: %v", err)
	}
	got, err := unmarshalContent(data)
	if err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}
	if got == nil || got.Role != adk.RoleAssistant {
		t.Fatalf("unexpected content: %#v", got)
	}
	if len(got.Parts) != 1 {
		t.Fatalf("unexpected parts length: %d", len(got.Parts))
	}
	text, ok := got.Parts[0].(adk.TextPart)
	if !ok || text.Text != "hello" {
		t.Fatalf("unexpected text part: %#v", got.Parts[0])
	}
}

func TestContentDTO_ToolCallPartRoundTrip(t *testing.T) {
	content := &adk.Content{
		Role: adk.RoleAssistant,
		Parts: []adk.Part{
			adk.ToolCallPart{
				ID:        "call-1",
				Name:      "run",
				Arguments: json.RawMessage(`{"k":"v"}`),
			},
		},
	}

	data, err := marshalContent(content)
	if err != nil {
		t.Fatalf("marshal content: %v", err)
	}
	got, err := unmarshalContent(data)
	if err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}
	call, ok := got.Parts[0].(adk.ToolCallPart)
	if !ok {
		t.Fatalf("unexpected part type: %T", got.Parts[0])
	}
	if call.ID != "call-1" || call.Name != "run" {
		t.Fatalf("unexpected tool call: %+v", call)
	}
	if string(call.Arguments) != `{"k":"v"}` {
		t.Fatalf("unexpected tool call arguments: %s", call.Arguments)
	}
}

func TestContentDTO_ToolResultPartRoundTrip(t *testing.T) {
	content := &adk.Content{
		Role: adk.RoleTool,
		Parts: []adk.Part{
			adk.ToolResultPart{
				CallID:  "call-1",
				Name:    "run",
				Content: "ok",
				IsError: true,
			},
		},
	}

	data, err := marshalContent(content)
	if err != nil {
		t.Fatalf("marshal content: %v", err)
	}
	got, err := unmarshalContent(data)
	if err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}
	result, ok := got.Parts[0].(adk.ToolResultPart)
	if !ok {
		t.Fatalf("unexpected part type: %T", got.Parts[0])
	}
	if result.CallID != "call-1" || result.Name != "run" || result.Content != "ok" || !result.IsError {
		t.Fatalf("unexpected tool result: %+v", result)
	}
}

func TestContentDTO_ThinkingPartRedacted(t *testing.T) {
	content := &adk.Content{
		Role:  adk.RoleAssistant,
		Parts: []adk.Part{adk.ThinkingPart{Thinking: "sensitive chain of thought"}},
	}

	data, err := marshalContent(content)
	if err != nil {
		t.Fatalf("marshal content: %v", err)
	}
	if strings.Contains(string(data), "sensitive chain of thought") {
		t.Fatalf("thinking content should be redacted, got: %s", string(data))
	}
	if !strings.Contains(string(data), thinkingRedactedValue) {
		t.Fatalf("expected redaction marker in json: %s", string(data))
	}

	got, err := unmarshalContent(data)
	if err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}
	if len(got.Parts) != 1 {
		t.Fatalf("unexpected part length: %d", len(got.Parts))
	}
	thinking, ok := got.Parts[0].(adk.ThinkingPart)
	if !ok {
		t.Fatalf("unexpected part type: %T", got.Parts[0])
	}
	if thinking.Thinking != "" {
		t.Fatalf("expected empty thinking after round trip, got: %q", thinking.Thinking)
	}
}

func TestSQLSessionService_InvalidJSON(t *testing.T) {
	t.Run("invalid state json", func(t *testing.T) {
		svc, store := newServiceForTest(t)
		ctx := context.Background()

		created, err := svc.Create(ctx, "user-1", map[string]any{"ok": true})
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		store.mu.Lock()
		row := store.sessions[created.ID]
		row.StateJSON = []byte("{invalid")
		store.sessions[created.ID] = row
		store.mu.Unlock()

		_, err = svc.Get(ctx, created.ID)
		if err == nil {
			t.Fatal("expected invalid state json error")
		}
	})

	t.Run("invalid content json", func(t *testing.T) {
		svc, store := newServiceForTest(t)
		ctx := context.Background()

		created, err := svc.Create(ctx, "user-1", map[string]any{"ok": true})
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		store.mu.Lock()
		store.events[created.ID] = []fakeEventRow{
			{
				ID:          "evt-1",
				SessionID:   created.ID,
				Seq:         1,
				Author:      "assistant",
				ContentJSON: []byte("{invalid"),
				ActionsJSON: []byte("null"),
				Partial:     false,
				Final:       true,
				Timestamp:   time.Now(),
			},
		}
		store.mu.Unlock()

		_, err = svc.Get(ctx, created.ID)
		if err == nil {
			t.Fatal("expected invalid content json error")
		}
	})
}

func TestSQLSessionService_DoesNotReadDotEnv(t *testing.T) {
	svc, _ := newServiceForTest(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, "user-1", map[string]any{"phase": "4.1"})
	if err != nil {
		t.Fatalf("create should not depend on .env: %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); err != nil {
		t.Fatalf("get should not depend on .env: %v", err)
	}
}
