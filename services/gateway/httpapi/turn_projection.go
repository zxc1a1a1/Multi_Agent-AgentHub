package httpapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

// liveTurn accumulates one agent turn so the Gateway can persist each turn as a
// separate assistant message instead of merging a group-chat run into one row.
type liveTurn struct {
	key    string
	author string
	text   strings.Builder
}

type turnAccumulator struct {
	order []string
	turns map[string]*liveTurn
}

func newTurnAccumulator() *turnAccumulator {
	return &turnAccumulator{turns: make(map[string]*liveTurn)}
}

func (a *turnAccumulator) ensure(key string) *liveTurn {
	if turn, ok := a.turns[key]; ok {
		return turn
	}
	turn := &liveTurn{key: key}
	a.turns[key] = turn
	a.order = append(a.order, key)
	return turn
}

func (a *turnAccumulator) started(item agui.Event) {
	turn := a.ensure(liveTurnKey(item))
	if author := eventAuthor(item, ""); author != "" {
		turn.author = author
	}
}

func (a *turnAccumulator) content(item agui.Event, fallbackAuthor string) {
	delta := item.Delta
	if delta == "" {
		delta = item.Text
	}
	if delta == "" {
		return
	}
	turn := a.ensure(liveTurnKey(item))
	if turn.author == "" {
		turn.author = eventAuthor(item, fallbackAuthor)
	}
	turn.text.WriteString(delta)
}

func (a *turnAccumulator) finished(item agui.Event) (liveTurn, bool) {
	key := liveTurnKey(item)
	turn, ok := a.turns[key]
	if !ok {
		return liveTurn{}, false
	}
	if author := eventAuthor(item, turn.author); author != "" {
		turn.author = author
	}
	delete(a.turns, key)
	return *turn, true
}

func (a *turnAccumulator) drain() []liveTurn {
	out := make([]liveTurn, 0, len(a.order))
	for _, key := range a.order {
		if turn, ok := a.turns[key]; ok {
			out = append(out, *turn)
			delete(a.turns, key)
		}
	}
	return out
}

func liveTurnKey(item agui.Event) string {
	switch item.Type {
	case agui.PublicTypeAgentTurnStarted, agui.PublicTypeAgentTurnContent, agui.PublicTypeAgentTurnFinished:
		return fmt.Sprintf("turn:%d", item.TurnIndex)
	}
	if item.MessageID != "" {
		return item.MessageID
	}
	return "default"
}

func eventAuthor(item agui.Event, fallback string) string {
	if name := strings.TrimSpace(item.AgentName); name != "" {
		return name
	}
	if item.Sender != nil {
		if name := strings.TrimSpace(item.Sender.Name); name != "" && item.Sender.Type != "user" {
			return name
		}
	}
	if author := strings.TrimSpace(item.Author); author != "" && author != "assistant" && author != "user" {
		return author
	}
	return fallback
}

// projectTurn persists one finished turn. Sequence is the public journal seq of
// the turn-finished event, so replay and projection share one cursor.
func (s *Server) projectTurn(ctx context.Context, conversationID, runID string, turn liveTurn, seq int64) error {
	if s.liveProjectionDisabled {
		return nil
	}
	text := strings.TrimSpace(turn.text.String())
	if text == "" {
		return nil
	}
	author := strings.TrimSpace(turn.author)
	if author == "" {
		author = "assistant"
	}
	msg := store.Message{
		ConversationID: conversationID,
		Author:         author,
		Role:           string(assistantRole),
		Text:           text,
		RunID:          runID,
	}
	persistCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if projector, ok := s.store.(store.ProjectionStore); ok {
		_, err := projector.ProjectAssistantMessage(persistCtx, msg, runID, seq, 0)
		return err
	}
	_, err := s.store.AppendMessage(persistCtx, msg)
	return err
}

const assistantRole = "assistant"
