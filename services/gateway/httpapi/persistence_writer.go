package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/domain"
)

// PersistenceWriter consumes translated AG-UI events and writes Run / RunStep /
// Message rows to SQLite via domain repositories without affecting the SSE stream.
//
// It is optional: when nil, handleChat behaves exactly as before (MemoryStore only).
type PersistenceWriter struct {
	conv domain.ConversationRepository
	msg  domain.MessageRepository
	run  domain.RunRepository
	evt  domain.EventRepository
	step domain.RunStepRepository

	mu            sync.Mutex
	runID         string
	currentMsgID  string
	currentTaskID string
	currentStepID string
	deltaBuf      strings.Builder
	stepIndex     int
	runFailed     bool
}

// NewPersistenceWriter returns a PersistenceWriter backed by domain repositories.
func NewPersistenceWriter(conv domain.ConversationRepository, msg domain.MessageRepository, run domain.RunRepository, evt domain.EventRepository, step domain.RunStepRepository) *PersistenceWriter {
	return &PersistenceWriter{conv: conv, msg: msg, run: run, evt: evt, step: step}
}

// SaveUserMessage persists the user message before the SSE loop starts.
func (w *PersistenceWriter) SaveUserMessage(ctx context.Context, conversationID, text string) error {
	if w == nil {
		return nil
	}
	_, err := w.msg.Create(ctx, domain.CreateMessageInput{
		ConversationID: conversationID,
		Role:           "user",
		SenderType:     "user",
		Content:        text,
		Status:         "sent",
	})
	return err
}

// HandleEvent processes one AG-UI event and writes the corresponding DB rows.
func (w *PersistenceWriter) HandleEvent(ctx context.Context, conversationID string, evt agui.Event) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	switch evt.Type {
	case agui.PublicTypeRunStarted:
		w.handleRunStarted(ctx, conversationID, evt)
	case agui.PublicTypeTextMessageStart:
		w.handleTextMessageStart(ctx, conversationID, evt)
	case agui.PublicTypeTextMessageContent:
		w.handleTextMessageContent(evt)
	case agui.PublicTypeTextMessageEnd:
		w.handleTextMessageEnd(ctx, evt)
	case agui.PublicTypeRunError:
		w.handleRunError(ctx, evt)
	case agui.PublicTypeRunFinished:
		w.handleRunFinished(ctx)
	}
}

func (w *PersistenceWriter) handleRunStarted(ctx context.Context, conversationID string, evt agui.Event) {
	w.runID = evt.RunID
	w.runFailed = false
	_, _ = w.run.Create(ctx, domain.CreateRunInput{
		ID:             evt.RunID,
		ConversationID: conversationID,
		Mode:           "direct",
		Status:         "executing",
	})
}

func (w *PersistenceWriter) handleTextMessageStart(ctx context.Context, conversationID string, evt agui.Event) {
	// Finalize the previous message if one is still buffered.
	w.finalizeCurrentMessage(ctx)

	if evt.MessageID == "" {
		return
	}

	// Track task and create RunStep if this is a new task.
	if evt.TaskID != "" && evt.TaskID != w.currentTaskID {
		w.currentTaskID = evt.TaskID
		w.stepIndex++
		stepID := newPWCryptoID()
		sType := senderType(evt)
		sName := senderName(evt)
		agentName := ""
		if sType == "agent" {
			agentName = sName
		}
		if err := w.step.Create(ctx, domain.CreateRunStepInput{
			ID:             stepID,
			RunID:          w.runID,
			ConversationID: conversationID,
			TaskID:         evt.TaskID,
			StepIndex:      w.stepIndex - 1,
			AgentName:      agentName,
			Status:         "running",
		}); err == nil {
			w.currentStepID = stepID
		}
	}

	// Build sender fields.
	sType := senderType(evt)
	sName := senderName(evt)
	agentName := ""
	if sType == "agent" {
		agentName = sName
	}

	msg, err := w.msg.Create(ctx, domain.CreateMessageInput{
		ConversationID: conversationID,
		RunID:          w.runID,
		MessageID:      evt.MessageID,
		Role:           "assistant",
		SenderType:     sType,
		SenderName:     sName,
		AgentName:      agentName,
		Content:        "",
		Status:         "streaming",
	})
	if err == nil {
		w.currentMsgID = msg.ID
		w.deltaBuf.Reset()
	}
}

func (w *PersistenceWriter) handleTextMessageContent(evt agui.Event) {
	delta := evt.Delta
	if delta == "" {
		return
	}
	if w.currentMsgID != "" {
		w.deltaBuf.WriteString(delta)
	}
}

func (w *PersistenceWriter) handleTextMessageEnd(ctx context.Context, evt agui.Event) {
	w.flushCurrentMessage(ctx)

	// Mark step completed.
	if w.currentStepID != "" {
		_ = w.step.UpdateStatus(ctx, w.currentStepID, "completed", "", "")
	}
}

func (w *PersistenceWriter) handleRunError(ctx context.Context, evt agui.Event) {
	w.runFailed = true

	errCode := ""
	errMsg := "run failed"
	if evt.Error != nil {
		errCode = evt.Error.Code
		if evt.Error.Message != "" {
			errMsg = evt.Error.Message
		}
	}

	_ = w.run.CompareAndSetStatus(ctx, w.runID, "executing", "failed", errCode, errMsg)

	if w.currentStepID != "" {
		_ = w.step.UpdateStatus(ctx, w.currentStepID, "failed", errCode, errMsg)
	}

	if w.currentMsgID != "" {
		content := w.deltaBuf.String()
		_ = w.msg.UpdateContentAndStatus(ctx, w.currentMsgID, content, "failed", errCode, errMsg)
	}
}

func (w *PersistenceWriter) handleRunFinished(ctx context.Context) {
	_ = w.run.CompareAndSetStatus(ctx, w.runID, "executing", "completed", "", "")
}

// finalizeCurrentMessage writes the buffered content to the current message and marks it sent.
func (w *PersistenceWriter) finalizeCurrentMessage(ctx context.Context) {
	if w.currentMsgID == "" {
		return
	}
	w.flushCurrentMessage(ctx)
	w.currentMsgID = ""
}

func (w *PersistenceWriter) flushCurrentMessage(ctx context.Context) {
	if w.currentMsgID == "" {
		return
	}
	content := w.deltaBuf.String()
	if content == "" {
		return
	}
	_ = w.msg.UpdateContentAndStatus(ctx, w.currentMsgID, content, "sent", "", "")
}

func senderType(evt agui.Event) string {
	if evt.Sender != nil && evt.Sender.Type != "" {
		return evt.Sender.Type
	}
	return "agent"
}

func senderName(evt agui.Event) string {
	if evt.Sender != nil && evt.Sender.Name != "" {
		return evt.Sender.Name
	}
	return evt.Author
}

func newPWCryptoID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}
	return "fallback-" + hex.EncodeToString([]byte(time.Now().String()))
}
