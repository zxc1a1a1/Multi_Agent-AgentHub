// Package projection recovers assistant-message projections from the durable
// public-event journal when the Gateway that streamed a run dies before writing
// its final message.
package projection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

// Store is the subset of the Gateway store needed to rebuild projections.
type Store interface {
	ListRunsWithUnprojectedEvents(ctx context.Context, consumerKey string, limit int) ([]string, error)
	ProjectionOffset(ctx context.Context, runID, consumerKey string) (seq int64, subIndex int, found bool, err error)
	ListPublicEventsForRunPage(ctx context.Context, runID string, afterSeq int64, afterSubIndex, limit int) ([]store.PublicEvent, error)
	ConversationIDForRun(ctx context.Context, runID string) (string, error)
	ProjectAssistantMessage(ctx context.Context, msg store.Message, runID string, seq int64, subIndex int) (bool, error)
}

// Worker periodically drains runs whose journal is ahead of the assistant
// projection offset. It only projects committed turns, so a live stream is not
// double-written.
type Worker struct {
	store    Store
	consumer string
	interval time.Duration
	pageSize int
	runLimit int
	now      func() time.Time
}

func New(st Store) *Worker {
	return &Worker{
		store:    st,
		consumer: store.ProjectionConsumerAssistantMessage,
		interval: 500 * time.Millisecond,
		pageSize: 500,
		runLimit: 50,
		now:      time.Now,
	}
}

// WithInterval overrides the poll interval (tests).
func (w *Worker) WithInterval(d time.Duration) *Worker {
	if d > 0 {
		w.interval = d
	}
	return w
}

// Run blocks until the context is cancelled, recovering once per tick.
func (w *Worker) Run(ctx context.Context) {
	w.recoverSafely(ctx)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.recoverSafely(ctx)
		}
	}
}

func (w *Worker) recoverSafely(ctx context.Context) {
	if w == nil || w.store == nil {
		return
	}
	processed, err := w.RecoverOnce(ctx)
	if err != nil {
		log.Printf("gateway projection: recovery failed: %v", err)
		return
	}
	if processed > 0 {
		log.Printf("gateway projection: recovered %d turn message(s)", processed)
	}
}

// RecoverOnce processes every run that has unprojected journal events.
func (w *Worker) RecoverOnce(ctx context.Context) (int, error) {
	runs, err := w.store.ListRunsWithUnprojectedEvents(ctx, w.consumer, w.runLimit)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, runID := range runs {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		count, err := w.RecoverRun(ctx, runID)
		if err != nil {
			log.Printf("gateway projection: run=%s recovery failed: %v", runID, err)
			continue
		}
		total += count
	}
	return total, nil
}

// RecoverRun projects every committed turn of one run after its current offset.
func (w *Worker) RecoverRun(ctx context.Context, runID string) (int, error) {
	conversationID, err := w.store.ConversationIDForRun(ctx, runID)
	if err != nil {
		if errors.Is(err, store.ErrConversationNotFound) {
			return 0, nil
		}
		return 0, err
	}
	offsetSeq, offsetSub, _, err := w.store.ProjectionOffset(ctx, runID, w.consumer)
	if err != nil {
		return 0, err
	}

	all := make([]store.PublicEvent, 0)
	cursorSeq, cursorSub := offsetSeq, offsetSub
	for {
		page, err := w.store.ListPublicEventsForRunPage(ctx, runID, cursorSeq, cursorSub, w.pageSize)
		if err != nil {
			return 0, err
		}
		if len(page) == 0 {
			break
		}
		all = append(all, page...)
		last := page[len(page)-1]
		cursorSeq, cursorSub = last.Seq, last.SubIndex
		if len(page) < w.pageSize {
			break
		}
	}

	turns := collectTurns(all)
	if len(turns) == 0 {
		return 0, nil
	}
	sort.SliceStable(turns, func(i, j int) bool {
		if turns[i].endSeq == turns[j].endSeq {
			return turns[i].endSub < turns[j].endSub
		}
		return turns[i].endSeq < turns[j].endSeq
	})

	projected := 0
	for _, turn := range turns {
		text := strings.TrimSpace(turn.text.String())
		if text == "" || turn.endSeq == 0 {
			continue
		}
		author := strings.TrimSpace(turn.author)
		if author == "" {
			author = "assistant"
		}
		inserted, err := w.store.ProjectAssistantMessage(ctx, store.Message{
			ConversationID: conversationID,
			Author:         author,
			Role:           "assistant",
			Text:           text,
			RunID:          runID,
		}, runID, turn.endSeq, turn.endSub)
		if err != nil {
			return projected, err
		}
		if inserted {
			projected++
		}
	}
	return projected, nil
}

type turnState struct {
	key      string
	author   string
	text     strings.Builder
	endSeq   int64
	endSub   int
	open     bool
	position int
}

type journalItem struct {
	Type       string `json:"type"`
	MessageID  string `json:"messageId"`
	TurnIndex  *int   `json:"turnIndex"`
	AgentName  string `json:"agentName"`
	Author     string `json:"author"`
	Role       string `json:"role"`
	Delta      string `json:"delta"`
	Text       string `json:"text"`
	Sender     *struct {
		Type string `json:"type"`
		Name string `json:"name"`
	} `json:"sender"`
}

// collectTurns groups assistant content by turn and only returns turns that
// reached AGENT_TURN_FINISHED / RUN_FINISHED. Incomplete trailing turns are
// left for a later tick.
func collectTurns(events []store.PublicEvent) []turnState {
	turns := make(map[string]*turnState)
	order := make([]string, 0)
	completed := make([]turnState, 0)

	ensure := func(key string) *turnState {
		if state, ok := turns[key]; ok {
			return state
		}
		state := &turnState{key: key, open: true, position: len(order)}
		turns[key] = state
		order = append(order, key)
		return state
	}

	for _, event := range events {
		var item journalItem
		if err := json.Unmarshal(event.Payload, &item); err != nil {
			continue
		}
		eventType := item.Type
		if eventType == "" {
			eventType = event.Type
		}
		isUser := item.Role == "user" || (item.Sender != nil && item.Sender.Type == "user")
		switch eventType {
		case "TEXT_MESSAGE_CONTENT", "AGENT_TURN_CONTENT", "message", "message.delta":
			if isUser {
				continue
			}
			delta := item.Delta
			if delta == "" {
				delta = item.Text
			}
			if delta == "" {
				continue
			}
			key := turnKey(item)
			state := ensure(key)
			if author := turnAuthor(item); author != "" {
				state.author = author
			}
			state.text.WriteString(delta)
		case "AGENT_TURN_FINISHED":
			key := turnKey(item)
			if state, ok := turns[key]; ok && state.open {
				state.endSeq = event.Seq
				state.endSub = event.SubIndex
				state.open = false
			}
		case "RUN_FINISHED":
			for _, key := range order {
				state := turns[key]
				if !state.open {
					continue
				}
				state.endSeq = event.Seq
				state.endSub = event.SubIndex
				state.open = false
			}
		}
	}

	for _, key := range order {
		state := *turns[key]
		if !state.open && state.endSeq > 0 {
			completed = append(completed, state)
		}
	}
	return completed
}

func turnKey(item journalItem) string {
	if item.MessageID != "" {
		return item.MessageID
	}
	if item.TurnIndex != nil {
		return fmt.Sprintf("turn:%d", *item.TurnIndex)
	}
	return "default"
}

func turnAuthor(item journalItem) string {
	if item.AgentName != "" {
		return item.AgentName
	}
	if item.Sender != nil && item.Sender.Name != "" && item.Sender.Type != "user" {
		return item.Sender.Name
	}
	if item.Author != "" && item.Author != "assistant" && item.Author != "user" {
		return item.Author
	}
	return ""
}
