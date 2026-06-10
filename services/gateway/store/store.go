package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrConversationNotFound indicates missing conversation data.
	ErrConversationNotFound = errors.New("conversation not found")
	idCounter               uint64
)

type Conversation struct {
	ID        string     `json:"id"`
	Title     string     `json:"title,omitempty"`
	UserID    string     `json:"userId"`
	AgentName string     `json:"agentName"`
	Pinned    bool       `json:"pinned"`
	PinnedAt  *time.Time `json:"pinnedAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	Author         string    `json:"author"`
	Role           string    `json:"role"`
	Text           string    `json:"text"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Store interface {
	CreateConversation(ctx context.Context, userID, agentName string) (*Conversation, error)
	GetConversation(ctx context.Context, id string) (*Conversation, error)
	ListConversations(ctx context.Context, userID string) ([]Conversation, error)
	UpdateConversation(ctx context.Context, id string, patch ConversationPatch) (*Conversation, error)
	UpdateConversationPin(ctx context.Context, id string, pinned bool) (*Conversation, error)
	AppendMessage(ctx context.Context, msg Message) (*Message, error)
	ListMessages(ctx context.Context, conversationID string) ([]Message, error)
	DeleteConversation(ctx context.Context, id string) error
}

// ConversationPatch carries optional fields for updating a conversation.
type ConversationPatch struct {
	Title     *string `json:"title,omitempty"`
	Pinned    *bool   `json:"pinned,omitempty"`
}

type MemoryStore struct {
	mu            sync.RWMutex
	conversations map[string]Conversation
	messages      map[string][]Message
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		conversations: make(map[string]Conversation),
		messages:      make(map[string][]Message),
	}
}

func (s *MemoryStore) CreateConversation(ctx context.Context, userID, agentName string) (*Conversation, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	conv := Conversation{
		ID:        newID(),
		UserID:    userID,
		AgentName: agentName,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.conversations[conv.ID] = conv
	s.messages[conv.ID] = make([]Message, 0)

	out := copyConversation(conv)
	return &out, nil
}

func (s *MemoryStore) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	conv, ok := s.conversations[id]
	if !ok {
		return nil, ErrConversationNotFound
	}
	out := copyConversation(conv)
	return &out, nil
}

func (s *MemoryStore) ListConversations(ctx context.Context, userID string) ([]Conversation, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Conversation, 0)
	for _, conv := range s.conversations {
		if userID != "" && conv.UserID != userID {
			continue
		}
		out = append(out, copyConversation(conv))
	}
	sortConversations(out)
	return out, nil
}

func (s *MemoryStore) UpdateConversation(ctx context.Context, id string, patch ConversationPatch) (*Conversation, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[id]
	if !ok {
		return nil, ErrConversationNotFound
	}

	now := time.Now().UTC()
	if patch.Pinned != nil {
		conv.Pinned = *patch.Pinned
		if *patch.Pinned {
			conv.PinnedAt = &now
		} else {
			conv.PinnedAt = nil
		}
	}
	if patch.Title != nil {
		conv.Title = *patch.Title
	}
	conv.UpdatedAt = now
	s.conversations[id] = conv

	out := copyConversation(conv)
	return &out, nil
}

func (s *MemoryStore) UpdateConversationPin(ctx context.Context, id string, pinned bool) (*Conversation, error) {
	return s.UpdateConversation(ctx, id, ConversationPatch{Pinned: &pinned})
}

func sortConversations(out []Conversation) {
	sort.Slice(out, func(i, j int) bool {
		// Pinned first
		if out[i].Pinned && !out[j].Pinned {
			return true
		}
		if !out[i].Pinned && out[j].Pinned {
			return false
		}
		// Then by updatedAt DESC (newest first)
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
}

func (s *MemoryStore) AppendMessage(ctx context.Context, msg Message) (*Message, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if msg.ConversationID == "" {
		return nil, ErrConversationNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[msg.ConversationID]
	if !ok {
		return nil, ErrConversationNotFound
	}

	if msg.ID == "" {
		msg.ID = newID()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}

	msgCopy := copyMessage(msg)
	s.messages[msg.ConversationID] = append(s.messages[msg.ConversationID], msgCopy)

	conv.UpdatedAt = msgCopy.CreatedAt
	s.conversations[msg.ConversationID] = conv

	out := copyMessage(msgCopy)
	return &out, nil
}

func (s *MemoryStore) ListMessages(ctx context.Context, conversationID string) ([]Message, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.conversations[conversationID]; !ok {
		return nil, ErrConversationNotFound
	}
	msgs := s.messages[conversationID]
	out := make([]Message, len(msgs))
	for i := range msgs {
		out[i] = copyMessage(msgs[i])
	}
	return out, nil
}

func (s *MemoryStore) DeleteConversation(ctx context.Context, id string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.conversations[id]; !ok {
		return ErrConversationNotFound
	}
	delete(s.conversations, id)
	delete(s.messages, id)
	return nil
}

func copyConversation(in Conversation) Conversation {
	var pinnedAt *time.Time
	if in.PinnedAt != nil {
		t := *in.PinnedAt
		pinnedAt = &t
	}
	return Conversation{
		ID:        in.ID,
		Title:     in.Title,
		UserID:    in.UserID,
		AgentName: in.AgentName,
		Pinned:    in.Pinned,
		PinnedAt:  pinnedAt,
		CreatedAt: in.CreatedAt,
		UpdatedAt: in.UpdatedAt,
	}
}

func copyMessage(in Message) Message {
	return Message{
		ID:             in.ID,
		ConversationID: in.ConversationID,
		Author:         in.Author,
		Role:           in.Role,
		Text:           in.Text,
		CreatedAt:      in.CreatedAt,
	}
}

func contextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func newID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}

	n := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("fallback-%d-%d", time.Now().UnixNano(), n)
}
