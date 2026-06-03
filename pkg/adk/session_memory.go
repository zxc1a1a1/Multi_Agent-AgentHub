package adk

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	errMemorySessionNotFound = errors.New("session not found")
	sessionIDCounter         uint64
)

// MemorySessionService is an in-memory SessionService implementation for local runtime and tests.
type MemorySessionService struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewMemorySessionService creates a thread-safe in-memory session service.
func NewMemorySessionService() SessionService {
	return &MemorySessionService{
		sessions: make(map[string]*Session),
	}
}

func (s *MemorySessionService) GetOrCreate(ctx context.Context, id string) (*Session, error) {
	session, err := s.Get(ctx, id)
	if err == nil {
		return session, nil
	}
	if !errors.Is(err, errMemorySessionNotFound) {
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

	s.mu.Lock()
	s.sessions[id] = session
	s.mu.Unlock()

	return cloneSession(session), nil
}

func (s *MemorySessionService) Create(ctx context.Context, userID string, initialState map[string]any) (*Session, error) {
	now := time.Now()
	session := &Session{
		ID:        generateSessionID(),
		UserID:    userID,
		Events:    make([]Event, 0),
		State:     NewSessionState(initialState),
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	s.sessions[session.ID] = session
	s.mu.Unlock()

	return cloneSession(session), nil
}

func (s *MemorySessionService) Get(ctx context.Context, id string) (*Session, error) {
	s.mu.RLock()
	session, ok := s.sessions[id]
	if !ok {
		s.mu.RUnlock()
		return nil, fmt.Errorf("%w: %s", errMemorySessionNotFound, id)
	}
	cloned := cloneSession(session)
	s.mu.RUnlock()
	return cloned, nil
}

func (s *MemorySessionService) AppendEvent(ctx context.Context, sessionID string, event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("%w: %s", errMemorySessionNotFound, sessionID)
	}

	session.Events = append(session.Events, cloneEvent(event))
	session.UpdatedAt = time.Now()
	return nil
}

func (s *MemorySessionService) UpdateState(ctx context.Context, sessionID string, delta map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("%w: %s", errMemorySessionNotFound, sessionID)
	}

	for key, value := range delta {
		session.State.Set(key, value)
	}
	session.UpdatedAt = time.Now()
	return nil
}

func (s *MemorySessionService) List(ctx context.Context, userID string) ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Session, 0)
	for _, session := range s.sessions {
		if session.UserID == userID {
			result = append(result, cloneSession(session))
		}
	}
	return result, nil
}

func (s *MemorySessionService) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[id]; !ok {
		return fmt.Errorf("%w: %s", errMemorySessionNotFound, id)
	}
	delete(s.sessions, id)
	return nil
}

func generateSessionID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return hex.EncodeToString(bytes[:])
	}

	counter := atomic.AddUint64(&sessionIDCounter, 1)
	return fmt.Sprintf("session-%d-%d", time.Now().UnixNano(), counter)
}

func cloneSession(session *Session) *Session {
	if session == nil {
		return nil
	}

	var stateCopy *SessionState
	if session.State != nil {
		stateCopy = NewSessionState(session.State.All())
	} else {
		stateCopy = NewSessionState(nil)
	}

	return &Session{
		ID:        session.ID,
		UserID:    session.UserID,
		Events:    cloneEvents(session.Events),
		State:     stateCopy,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	}
}

func cloneEvents(events []Event) []Event {
	copied := make([]Event, len(events))
	for i := range events {
		copied[i] = cloneEvent(events[i])
	}
	return copied
}

func cloneEvent(event Event) Event {
	return Event{
		ID:        event.ID,
		Author:    event.Author,
		Content:   cloneContent(event.Content),
		Actions:   cloneEventActions(event.Actions),
		Partial:   event.Partial,
		Final:     event.Final,
		Timestamp: event.Timestamp,
	}
}

func cloneContent(content *Content) *Content {
	if content == nil {
		return nil
	}

	parts := make([]Part, len(content.Parts))
	for i := range content.Parts {
		parts[i] = clonePart(content.Parts[i])
	}

	return &Content{
		Role:  content.Role,
		Parts: parts,
	}
}

func clonePart(part Part) Part {
	switch p := part.(type) {
	case TextPart:
		return p
	case ToolCallPart:
		return ToolCallPart{
			ID:        p.ID,
			Name:      p.Name,
			Arguments: append([]byte(nil), p.Arguments...),
		}
	case ToolResultPart:
		return p
	case ThinkingPart:
		return p
	default:
		return part
	}
}

func cloneEventActions(actions *EventActions) *EventActions {
	if actions == nil {
		return nil
	}

	stateDelta := make(map[string]any, len(actions.StateDelta))
	for key, value := range actions.StateDelta {
		stateDelta[key] = value
	}

	artifactDelta := make([]Artifact, len(actions.ArtifactDelta))
	for i := range actions.ArtifactDelta {
		artifactDelta[i] = cloneArtifact(actions.ArtifactDelta[i])
	}

	return &EventActions{
		StateDelta:    stateDelta,
		TransferAgent: actions.TransferAgent,
		ArtifactDelta: artifactDelta,
	}
}

func cloneArtifact(artifact Artifact) Artifact {
	metadata := make(map[string]string, len(artifact.Metadata))
	for key, value := range artifact.Metadata {
		metadata[key] = value
	}

	return Artifact{
		Type:     artifact.Type,
		Title:    artifact.Title,
		Content:  artifact.Content,
		Metadata: metadata,
	}
}
