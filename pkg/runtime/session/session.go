package session

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const thinkingRedactedValue = "[redacted]"

var (
	// ErrSessionNotFound indicates the requested session does not exist.
	ErrSessionNotFound = errors.New("session not found")

	sessionIDCounter uint64
)

const (
	insertSessionSQL = `
INSERT INTO sessions (id, user_id, state_json, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
`

	getSessionSQL = `
SELECT id, user_id, state_json, created_at, updated_at
FROM sessions
WHERE id = ?
`

	getSessionEventsSQL = `
SELECT id, seq, author, content_json, actions_json, partial, final, timestamp
FROM session_events
WHERE session_id = ?
ORDER BY seq ASC
`

	getSessionStateSQL = `
SELECT state_json
FROM sessions
WHERE id = ?
`

	getSessionExistsSQL = `
SELECT id
FROM sessions
WHERE id = ?
`

	getNextEventSeqSQL = `
SELECT COALESCE(MAX(seq), 0) + 1
FROM session_events
WHERE session_id = ?
`

	insertEventSQL = `
INSERT INTO session_events (
	id, session_id, seq, author, content_json, actions_json, partial, final, timestamp
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	updateSessionStateSQL = `
UPDATE sessions
SET state_json = ?, updated_at = ?
WHERE id = ?
`

	updateSessionUpdatedAtSQL = `
UPDATE sessions
SET updated_at = ?
WHERE id = ?
`

	listSessionsSQL = `
SELECT id, user_id, state_json, created_at, updated_at
FROM sessions
WHERE user_id = ?
ORDER BY updated_at DESC, id DESC
`

	deleteSessionEventsSQL = `
DELETE FROM session_events
WHERE session_id = ?
`

	deleteSessionSQL = `
DELETE FROM sessions
WHERE id = ?
`
)

type contentDTO struct {
	Role  adk.Role  `json:"role"`
	Parts []partDTO `json:"parts"`
}

type partDTO struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	CallID    string          `json:"call_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
}

type eventActionsDTO struct {
	StateDelta    map[string]any `json:"state_delta,omitempty"`
	TransferAgent string         `json:"transfer_agent,omitempty"`
	ArtifactDelta []artifactDTO  `json:"artifact_delta,omitempty"`
}

type artifactDTO struct {
	Type     string            `json:"type"`
	Title    string            `json:"title"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// SQLSessionService is a SQL-backed adk.SessionService implementation.
//
// It expects the following pre-created schema (or compatible schema):
// - sessions(id, user_id, state_json, created_at, updated_at)
// - session_events(id, session_id, seq, author, content_json, actions_json, partial, final, timestamp)
type SQLSessionService struct {
	db *sql.DB
	mu sync.Mutex
}

// NewSQLSessionService creates a SQLSessionService with injected DB dependency.
func NewSQLSessionService(db *sql.DB) (*SQLSessionService, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	return &SQLSessionService{db: db}, nil
}

func (s *SQLSessionService) Create(ctx context.Context, userID string, initialState map[string]any) (*adk.Session, error) {
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	stateCopy := cloneState(initialState)
	stateJSON, err := marshalState(stateCopy)
	if err != nil {
		return nil, fmt.Errorf("marshal initial state: %w", err)
	}

	now := time.Now()
	sessionID := generateSessionID()
	_, err = s.db.ExecContext(ctx, insertSessionSQL, sessionID, userID, stateJSON, now, now)
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}

	return &adk.Session{
		ID:        sessionID,
		UserID:    userID,
		Events:    make([]adk.Event, 0),
		State:     adk.NewSessionState(stateCopy),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *SQLSessionService) Get(ctx context.Context, id string) (*adk.Session, error) {
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var sessionID string
	var userID string
	var stateJSON []byte
	var createdAt time.Time
	var updatedAt time.Time

	err := s.db.QueryRowContext(ctx, getSessionSQL, id).Scan(&sessionID, &userID, &stateJSON, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, id)
		}
		return nil, fmt.Errorf("query session: %w", err)
	}

	stateMap, err := unmarshalState(stateJSON)
	if err != nil {
		return nil, fmt.Errorf("decode state json: %w", err)
	}

	events, err := s.getEvents(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return &adk.Session{
		ID:        sessionID,
		UserID:    userID,
		Events:    events,
		State:     adk.NewSessionState(stateMap),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func (s *SQLSessionService) GetOrCreate(ctx context.Context, id string) (*adk.Session, error) {
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if id == "" {
		return nil, fmt.Errorf("session id must not be empty")
	}

	session, err := s.Get(ctx, id)
	if err == nil {
		return session, nil
	}
	if !errors.Is(err, ErrSessionNotFound) {
		return nil, err
	}

	now := time.Now()
	stateJSON := []byte("{}")
	_, err = s.db.ExecContext(ctx, insertSessionSQL, id, id, stateJSON, now, now)
	if err != nil {
		// Handle race: another goroutine may have created the session between Get and Insert.
		if strings.Contains(err.Error(), "duplicate") {
			session, getErr := s.Get(ctx, id)
			if getErr != nil {
				return nil, fmt.Errorf("get session after concurrent insert: %w", getErr)
			}
			return session, nil
		}
		return nil, fmt.Errorf("insert session for GetOrCreate: %w", err)
	}

	return &adk.Session{
		ID:        id,
		UserID:    id,
		Events:    make([]adk.Event, 0),
		State:     adk.NewSessionState(map[string]any{}),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *SQLSessionService) AppendEvent(ctx context.Context, sessionID string, event adk.Event) error {
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	exists, err := s.sessionExists(ctx, sessionID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: %s", ErrSessionNotFound, sessionID)
	}

	contentJSON, err := marshalContent(event.Content)
	if err != nil {
		return fmt.Errorf("marshal content json: %w", err)
	}
	actionsJSON, err := marshalActions(event.Actions)
	if err != nil {
		return fmt.Errorf("marshal actions json: %w", err)
	}

	timestamp := event.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	seq, err := s.nextEventSeq(ctx, sessionID)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(
		ctx,
		insertEventSQL,
		event.ID,
		sessionID,
		seq,
		event.Author,
		contentJSON,
		actionsJSON,
		event.Partial,
		event.Final,
		timestamp,
	)
	if err != nil {
		return fmt.Errorf("insert session event: %w", err)
	}

	return s.updateSessionTimestamp(ctx, sessionID, time.Now())
}

func (s *SQLSessionService) UpdateState(ctx context.Context, sessionID string, delta map[string]any) error {
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var stateJSON []byte
	err := s.db.QueryRowContext(ctx, getSessionStateSQL, sessionID).Scan(&stateJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: %s", ErrSessionNotFound, sessionID)
		}
		return fmt.Errorf("query current state json: %w", err)
	}

	stateMap, err := unmarshalState(stateJSON)
	if err != nil {
		return fmt.Errorf("decode current state json: %w", err)
	}
	for key, value := range delta {
		stateMap[key] = value
	}

	updatedStateJSON, err := marshalState(stateMap)
	if err != nil {
		return fmt.Errorf("marshal updated state json: %w", err)
	}

	now := time.Now()
	res, err := s.db.ExecContext(ctx, updateSessionStateSQL, updatedStateJSON, now, sessionID)
	if err != nil {
		return fmt.Errorf("update session state: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read update state affected rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("%w: %s", ErrSessionNotFound, sessionID)
	}

	return nil
}

func (s *SQLSessionService) List(ctx context.Context, userID string) ([]*adk.Session, error) {
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, listSessionsSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	out := make([]*adk.Session, 0)
	for rows.Next() {
		var sessionID string
		var user string
		var stateJSON []byte
		var createdAt time.Time
		var updatedAt time.Time

		if err := rows.Scan(&sessionID, &user, &stateJSON, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan listed session: %w", err)
		}

		stateMap, err := unmarshalState(stateJSON)
		if err != nil {
			return nil, fmt.Errorf("decode listed session state json: %w", err)
		}

		out = append(out, &adk.Session{
			ID:        sessionID,
			UserID:    user,
			Events:    make([]adk.Event, 0),
			State:     adk.NewSessionState(stateMap),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate listed sessions: %w", err)
	}

	return out, nil
}

func (s *SQLSessionService) Delete(ctx context.Context, id string) error {
	ctx = ensureContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}

	exists, err := s.sessionExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: %s", ErrSessionNotFound, id)
	}

	if _, err := s.db.ExecContext(ctx, deleteSessionEventsSQL, id); err != nil {
		return fmt.Errorf("delete session events: %w", err)
	}

	res, err := s.db.ExecContext(ctx, deleteSessionSQL, id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read delete affected rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("%w: %s", ErrSessionNotFound, id)
	}

	return nil
}

func (s *SQLSessionService) getEvents(ctx context.Context, sessionID string) ([]adk.Event, error) {
	rows, err := s.db.QueryContext(ctx, getSessionEventsSQL, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query session events: %w", err)
	}
	defer rows.Close()

	events := make([]adk.Event, 0)
	for rows.Next() {
		var eventID string
		var seq int64
		var author string
		var contentJSON []byte
		var actionsJSON []byte
		var partial bool
		var final bool
		var timestamp time.Time

		if err := rows.Scan(
			&eventID,
			&seq,
			&author,
			&contentJSON,
			&actionsJSON,
			&partial,
			&final,
			&timestamp,
		); err != nil {
			return nil, fmt.Errorf("scan session event: %w", err)
		}
		_ = seq

		content, err := unmarshalContent(contentJSON)
		if err != nil {
			return nil, fmt.Errorf("decode event content json: %w", err)
		}
		actions, err := unmarshalActions(actionsJSON)
		if err != nil {
			return nil, fmt.Errorf("decode event actions json: %w", err)
		}

		events = append(events, adk.Event{
			ID:        eventID,
			Author:    author,
			Content:   content,
			Actions:   actions,
			Partial:   partial,
			Final:     final,
			Timestamp: timestamp,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session events: %w", err)
	}

	return events, nil
}

func (s *SQLSessionService) sessionExists(ctx context.Context, sessionID string) (bool, error) {
	var id string
	err := s.db.QueryRowContext(ctx, getSessionExistsSQL, sessionID).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("query session exists: %w", err)
	}
	return true, nil
}

func (s *SQLSessionService) nextEventSeq(ctx context.Context, sessionID string) (int64, error) {
	var next int64
	err := s.db.QueryRowContext(ctx, getNextEventSeqSQL, sessionID).Scan(&next)
	if err != nil {
		return 0, fmt.Errorf("query next event seq: %w", err)
	}
	if next <= 0 {
		return 1, nil
	}
	return next, nil
}

func (s *SQLSessionService) updateSessionTimestamp(ctx context.Context, sessionID string, updatedAt time.Time) error {
	res, err := s.db.ExecContext(ctx, updateSessionUpdatedAtSQL, updatedAt, sessionID)
	if err != nil {
		return fmt.Errorf("update session timestamp: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read update timestamp affected rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("%w: %s", ErrSessionNotFound, sessionID)
	}
	return nil
}

func ensureContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

func generateSessionID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return hex.EncodeToString(bytes[:])
	}

	counter := atomic.AddUint64(&sessionIDCounter, 1)
	return fmt.Sprintf("session-%d-%d", time.Now().UnixNano(), counter)
}

func cloneState(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func marshalState(state map[string]any) ([]byte, error) {
	if state == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(state)
}

func unmarshalState(data []byte) (map[string]any, error) {
	if len(data) == 0 || string(data) == "null" {
		return map[string]any{}, nil
	}

	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if out == nil {
		return map[string]any{}, nil
	}
	return out, nil
}

func marshalContent(content *adk.Content) ([]byte, error) {
	if content == nil {
		return []byte("null"), nil
	}

	dto := contentDTO{
		Role:  content.Role,
		Parts: make([]partDTO, 0, len(content.Parts)),
	}
	for _, part := range content.Parts {
		switch p := part.(type) {
		case adk.TextPart:
			dto.Parts = append(dto.Parts, partDTO{Type: "text", Text: p.Text})
		case *adk.TextPart:
			if p == nil {
				dto.Parts = append(dto.Parts, partDTO{Type: "text"})
			} else {
				dto.Parts = append(dto.Parts, partDTO{Type: "text", Text: p.Text})
			}
		case adk.ToolCallPart:
			dto.Parts = append(dto.Parts, partDTO{
				Type:      "tool_call",
				ID:        p.ID,
				Name:      p.Name,
				Arguments: append([]byte(nil), p.Arguments...),
			})
		case *adk.ToolCallPart:
			if p == nil {
				dto.Parts = append(dto.Parts, partDTO{Type: "tool_call"})
			} else {
				dto.Parts = append(dto.Parts, partDTO{
					Type:      "tool_call",
					ID:        p.ID,
					Name:      p.Name,
					Arguments: append([]byte(nil), p.Arguments...),
				})
			}
		case adk.ToolResultPart:
			dto.Parts = append(dto.Parts, partDTO{
				Type:    "tool_result",
				CallID:  p.CallID,
				Name:    p.Name,
				Content: p.Content,
				IsError: p.IsError,
			})
		case *adk.ToolResultPart:
			if p == nil {
				dto.Parts = append(dto.Parts, partDTO{Type: "tool_result"})
			} else {
				dto.Parts = append(dto.Parts, partDTO{
					Type:    "tool_result",
					CallID:  p.CallID,
					Name:    p.Name,
					Content: p.Content,
					IsError: p.IsError,
				})
			}
		case adk.ThinkingPart:
			dto.Parts = append(dto.Parts, partDTO{Type: "thinking", Thinking: thinkingRedactedValue})
		case *adk.ThinkingPart:
			dto.Parts = append(dto.Parts, partDTO{Type: "thinking", Thinking: thinkingRedactedValue})
		default:
			return nil, fmt.Errorf("unsupported part type: %T", part)
		}
	}

	return json.Marshal(dto)
}

func unmarshalContent(data []byte) (*adk.Content, error) {
	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}

	var dto contentDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}

	out := &adk.Content{
		Role:  dto.Role,
		Parts: make([]adk.Part, 0, len(dto.Parts)),
	}
	for _, part := range dto.Parts {
		switch part.Type {
		case "text":
			out.Parts = append(out.Parts, adk.TextPart{Text: part.Text})
		case "tool_call":
			out.Parts = append(out.Parts, adk.ToolCallPart{
				ID:        part.ID,
				Name:      part.Name,
				Arguments: append([]byte(nil), part.Arguments...),
			})
		case "tool_result":
			out.Parts = append(out.Parts, adk.ToolResultPart{
				CallID:  part.CallID,
				Name:    part.Name,
				Content: part.Content,
				IsError: part.IsError,
			})
		case "thinking":
			out.Parts = append(out.Parts, adk.ThinkingPart{})
		default:
			return nil, fmt.Errorf("unsupported part dto type: %q", part.Type)
		}
	}

	return out, nil
}

func marshalActions(actions *adk.EventActions) ([]byte, error) {
	if actions == nil {
		return []byte("null"), nil
	}

	dto := eventActionsDTO{
		StateDelta:    cloneState(actions.StateDelta),
		TransferAgent: actions.TransferAgent,
		ArtifactDelta: make([]artifactDTO, 0, len(actions.ArtifactDelta)),
	}
	for _, artifact := range actions.ArtifactDelta {
		dto.ArtifactDelta = append(dto.ArtifactDelta, artifactDTO{
			Type:     artifact.Type,
			Title:    artifact.Title,
			Content:  artifact.Content,
			Metadata: cloneStringMap(artifact.Metadata),
		})
	}
	return json.Marshal(dto)
}

func unmarshalActions(data []byte) (*adk.EventActions, error) {
	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}

	var dto eventActionsDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}

	actions := &adk.EventActions{
		StateDelta:    cloneState(dto.StateDelta),
		TransferAgent: dto.TransferAgent,
		ArtifactDelta: make([]adk.Artifact, 0, len(dto.ArtifactDelta)),
	}
	for _, artifact := range dto.ArtifactDelta {
		actions.ArtifactDelta = append(actions.ArtifactDelta, adk.Artifact{
			Type:     artifact.Type,
			Title:    artifact.Title,
			Content:  artifact.Content,
			Metadata: cloneStringMap(artifact.Metadata),
		})
	}

	return actions, nil
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
