package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/model"
)

type fakeDB struct {
	conversationExistsFn func(string) (bool, error)
	saveMessageFn        func(string, string, string, string, []model.ArtifactData) error
	getMessagesFn        func(string, int) ([]model.Message, error)

	conversationExistsCalls int
	saveMessageCalls        int
	getMessagesCalls        int
}

func (f *fakeDB) ListConversations() ([]model.Conversation, error) {
	return nil, nil
}

func (f *fakeDB) CreateConversation(title, agentName string) (*model.Conversation, error) {
	return nil, nil
}

func (f *fakeDB) GetMessages(conversationID string, limit int) ([]model.Message, error) {
	f.getMessagesCalls++
	if f.getMessagesFn != nil {
		return f.getMessagesFn(conversationID, limit)
	}
	return nil, nil
}

func (f *fakeDB) SaveMessage(conversationID, senderType, senderName, content string, artifacts []model.ArtifactData) error {
	f.saveMessageCalls++
	if f.saveMessageFn != nil {
		return f.saveMessageFn(conversationID, senderType, senderName, content, artifacts)
	}
	return nil
}

func (f *fakeDB) UpdateConversationTitle(id, title string) error {
	return nil
}

func (f *fakeDB) ConversationExists(conversationID string) (bool, error) {
	f.conversationExistsCalls++
	if f.conversationExistsFn != nil {
		return f.conversationExistsFn(conversationID)
	}
	return false, nil
}

type fakeOrchestrator struct {
	processFn func(context.Context, model.AGUIRunRequest, []model.Message, chan<- model.AGUIEvent)
}

func (f *fakeOrchestrator) Process(ctx context.Context, req model.AGUIRunRequest, history []model.Message, events chan<- model.AGUIEvent) {
	if f.processFn != nil {
		f.processFn(ctx, req, history, events)
	}
}

func buildAGUIRunBody(threadID string) []byte {
	req := model.AGUIRunRequest{
		ThreadID: threadID,
		RunID:    "run-1",
		Messages: []model.AGUIMessage{
			{Role: "user", Content: "hello"},
		},
		Tools: []model.AGUITool{
			{Name: "code_preview"},
		},
	}
	body, _ := json.Marshal(req)
	return body
}

type closeNotifyRecorder struct {
	*httptest.ResponseRecorder
	closeCh chan bool
}

func newCloseNotifyRecorder() *closeNotifyRecorder {
	return &closeNotifyRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		closeCh:          make(chan bool, 1),
	}
}

func (r *closeNotifyRecorder) CloseNotify() <-chan bool {
	return r.closeCh
}

func newAGUIContext(body []byte) (*gin.Context, *closeNotifyRecorder) {
	rec := newCloseNotifyRecorder()
	c, _ := gin.CreateTestContext(rec)
	r, _ := http.NewRequest(http.MethodPost, "/api/agui/run", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	c.Request = r
	return c, rec
}

func TestHandleAGUIRunThreadIDInvalidReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := &fakeDB{
		conversationExistsFn: func(string) (bool, error) { return true, nil },
	}
	h := &Handler{db: db}

	c, rec := newAGUIContext(buildAGUIRunBody("not-a-uuid"))
	h.HandleAGUIRun(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if db.conversationExistsCalls != 0 {
		t.Fatalf("ConversationExists should not be called for invalid threadId")
	}
	if db.saveMessageCalls != 0 {
		t.Fatalf("SaveMessage should not be called for invalid threadId")
	}
}

func TestHandleAGUIRunThreadIDEmptyReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := &fakeDB{
		conversationExistsFn: func(string) (bool, error) { return true, nil },
	}
	h := &Handler{db: db}

	c, rec := newAGUIContext(buildAGUIRunBody(""))
	h.HandleAGUIRun(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if db.conversationExistsCalls != 0 {
		t.Fatalf("ConversationExists should not be called for empty threadId")
	}
}

func TestHandleAGUIRunConversationMissingReturns404(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := &fakeDB{
		conversationExistsFn: func(string) (bool, error) { return false, nil },
	}
	h := &Handler{db: db}

	c, rec := newAGUIContext(buildAGUIRunBody("11111111-1111-1111-1111-111111111111"))
	h.HandleAGUIRun(c)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if db.conversationExistsCalls != 1 {
		t.Fatalf("expected ConversationExists called once, got %d", db.conversationExistsCalls)
	}
	if db.saveMessageCalls != 0 {
		t.Fatalf("SaveMessage should not be called when conversation missing")
	}
	if db.getMessagesCalls != 0 {
		t.Fatalf("GetMessages should not be called when conversation missing")
	}
}

func TestHandleAGUIRunConversationExistsCheckFailureReturns500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := &fakeDB{
		conversationExistsFn: func(string) (bool, error) { return false, assertErr{} },
	}
	h := &Handler{db: db}

	c, rec := newAGUIContext(buildAGUIRunBody("11111111-1111-1111-1111-111111111111"))
	h.HandleAGUIRun(c)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestHandleAGUIRunConversationExistsFlowsToStreamAndSave(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := &fakeDB{
		conversationExistsFn: func(string) (bool, error) { return true, nil },
		getMessagesFn:        func(string, int) ([]model.Message, error) { return nil, nil },
	}

	orc := &fakeOrchestrator{
		processFn: func(ctx context.Context, req model.AGUIRunRequest, history []model.Message, events chan<- model.AGUIEvent) {
			events <- model.AGUIEvent{Type: "TEXT_MESSAGE_CONTENT", Content: "hello"}
			events <- model.AGUIEvent{Type: "RUN_FINISHED", RunID: req.RunID}
		},
	}

	h := &Handler{
		db:  db,
		orc: orc,
	}

	c, rec := newAGUIContext(buildAGUIRunBody("11111111-1111-1111-1111-111111111111"))
	h.HandleAGUIRun(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if db.conversationExistsCalls != 1 {
		t.Fatalf("expected ConversationExists called once, got %d", db.conversationExistsCalls)
	}
	if db.saveMessageCalls < 2 {
		t.Fatalf("expected user+agent SaveMessage calls, got %d", db.saveMessageCalls)
	}
	if db.getMessagesCalls != 1 {
		t.Fatalf("expected GetMessages called once, got %d", db.getMessagesCalls)
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "db failure" }
