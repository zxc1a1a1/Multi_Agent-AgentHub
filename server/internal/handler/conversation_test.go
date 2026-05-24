package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/server/internal/model"
)

type fakeConversationDB struct {
	getMessagesCalls int
	getMessagesFn    func(string, int) ([]model.Message, error)
}

func (f *fakeConversationDB) ListConversations() ([]model.Conversation, error) {
	return nil, nil
}

func (f *fakeConversationDB) CreateConversation(title, agentName string) (*model.Conversation, error) {
	return nil, nil
}

func (f *fakeConversationDB) GetMessages(conversationID string, limit int) ([]model.Message, error) {
	f.getMessagesCalls++
	if f.getMessagesFn != nil {
		return f.getMessagesFn(conversationID, limit)
	}
	return []model.Message{}, nil
}

func (f *fakeConversationDB) SaveMessage(conversationID, senderType, senderName, content string, artifacts []model.ArtifactData) error {
	return nil
}

func (f *fakeConversationDB) UpdateConversationTitle(id, title string) error {
	return nil
}

func (f *fakeConversationDB) ConversationExists(conversationID string) (bool, error) {
	return true, nil
}

func TestListMessagesInvalidConversationIDReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := &fakeConversationDB{}
	h := &Handler{db: db}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/conversations/not-a-uuid/messages", nil)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}

	h.ListMessages(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if db.getMessagesCalls != 0 {
		t.Fatalf("GetMessages should not be called for invalid id, got calls=%d", db.getMessagesCalls)
	}
}

func TestListMessagesEmptyConversationIDReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := &fakeConversationDB{}
	h := &Handler{db: db}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/conversations//messages", nil)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: " "}}

	h.ListMessages(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if db.getMessagesCalls != 0 {
		t.Fatalf("GetMessages should not be called for empty id, got calls=%d", db.getMessagesCalls)
	}
}

func TestListMessagesValidConversationIDReturns200(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := &fakeConversationDB{
		getMessagesFn: func(conversationID string, limit int) ([]model.Message, error) {
			if conversationID != "11111111-1111-1111-1111-111111111111" {
				t.Fatalf("unexpected conversationID=%s", conversationID)
			}
			if limit != 50 {
				t.Fatalf("expected default limit=50, got=%d", limit)
			}
			return []model.Message{}, nil
		},
	}
	h := &Handler{db: db}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/api/conversations/11111111-1111-1111-1111-111111111111/messages", nil)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "11111111-1111-1111-1111-111111111111"}}

	h.ListMessages(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if db.getMessagesCalls != 1 {
		t.Fatalf("expected GetMessages called once, got=%d", db.getMessagesCalls)
	}
}
