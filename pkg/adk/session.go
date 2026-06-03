package adk

import (
	"context"
	"sync"
	"time"
)

// Session is the ADK-level conversation execution container.
type Session struct {
	ID        string
	UserID    string
	Events    []Event
	State     *SessionState
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SessionState is a thread-safe key-value store attached to a Session.
type SessionState struct {
	mu   sync.RWMutex
	data map[string]any
}

// NewSessionState creates a state and copies initial values.
func NewSessionState(initial map[string]any) *SessionState {
	data := make(map[string]any, len(initial))
	for key, value := range initial {
		data[key] = value
	}
	return &SessionState{data: data}
}

// Get fetches one value by key.
func (s *SessionState) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.data[key]
	return value, ok
}

// Set sets one key/value pair.
func (s *SessionState) Set(key string, val any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data == nil {
		s.data = make(map[string]any)
	}
	s.data[key] = val
}

// All returns a defensive copy of current state.
func (s *SessionState) All() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copied := make(map[string]any, len(s.data))
	for key, value := range s.data {
		copied[key] = value
	}
	return copied
}

// SessionService defines session persistence operations.
type SessionService interface {
	Create(ctx context.Context, userID string, initialState map[string]any) (*Session, error)
	Get(ctx context.Context, id string) (*Session, error)
	GetOrCreate(ctx context.Context, id string) (*Session, error)
	AppendEvent(ctx context.Context, sessionID string, event Event) error
	UpdateState(ctx context.Context, sessionID string, delta map[string]any) error
	List(ctx context.Context, userID string) ([]*Session, error)
	Delete(ctx context.Context, id string) error
}
