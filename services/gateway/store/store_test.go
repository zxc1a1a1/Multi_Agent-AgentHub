package store

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestCreateConversation(t *testing.T) {
	t.Setenv("DATABASE_URL", "fake-value")

	st := NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}
	if conv.ID == "" {
		t.Fatalf("expected non-empty id")
	}
	if conv.UserID != "user-1" || conv.AgentName != "code-agent" {
		t.Fatalf("unexpected conversation fields: %+v", conv)
	}
}

func TestGetConversation(t *testing.T) {
	st := NewMemoryStore()
	created, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	got, err := st.GetConversation(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get conversation failed: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected same id, got %q", got.ID)
	}
}

func TestListConversationsByUserID(t *testing.T) {
	st := NewMemoryStore()
	if _, err := st.CreateConversation(context.Background(), "user-1", "agent-a"); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, err := st.CreateConversation(context.Background(), "user-2", "agent-b"); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, err := st.CreateConversation(context.Background(), "user-1", "agent-c"); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	list, err := st.ListConversations(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 conversations, got %d", len(list))
	}
	for _, item := range list {
		if item.UserID != "user-1" {
			t.Fatalf("unexpected user in list: %+v", item)
		}
	}
}

func TestAppendAndListMessages(t *testing.T) {
	st := NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	msg, err := st.AppendMessage(context.Background(), Message{
		ConversationID: conv.ID,
		Author:         "user",
		Role:           "user",
		Text:           "hello",
	})
	if err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if msg.ID == "" {
		t.Fatalf("expected non-empty message id")
	}

	list, err := st.ListMessages(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("list messages failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 message, got %d", len(list))
	}
	if list[0].Text != "hello" {
		t.Fatalf("unexpected message text: %q", list[0].Text)
	}
}

func TestDeleteConversationAlsoDeletesMessages(t *testing.T) {
	st := NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, err := st.AppendMessage(context.Background(), Message{
		ConversationID: conv.ID,
		Author:         "user",
		Role:           "user",
		Text:           "msg",
	}); err != nil {
		t.Fatalf("append failed: %v", err)
	}

	if err := st.DeleteConversation(context.Background(), conv.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if _, err := st.GetConversation(context.Background(), conv.ID); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected conversation not found after delete, got: %v", err)
	}
	if _, err := st.ListMessages(context.Background(), conv.ID); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected messages not found after delete, got: %v", err)
	}
}

func TestNotFoundErrors(t *testing.T) {
	st := NewMemoryStore()
	if _, err := st.GetConversation(context.Background(), "missing"); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected not found for get, got: %v", err)
	}
	if _, err := st.ListMessages(context.Background(), "missing"); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected not found for list messages, got: %v", err)
	}
	if err := st.DeleteConversation(context.Background(), "missing"); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected not found for delete, got: %v", err)
	}
	if _, err := st.AppendMessage(context.Background(), Message{ConversationID: "missing"}); !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected not found for append message, got: %v", err)
	}
}

func TestReturnsCopies(t *testing.T) {
	st := NewMemoryStore()
	conv, err := st.CreateConversation(context.Background(), "user-1", "agent-a")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, err := st.AppendMessage(context.Background(), Message{
		ConversationID: conv.ID,
		Author:         "user",
		Role:           "user",
		Text:           "hello",
	}); err != nil {
		t.Fatalf("append failed: %v", err)
	}

	gotConv, err := st.GetConversation(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	gotConv.UserID = "mutated"

	gotMsgs, err := st.ListMessages(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	gotMsgs[0].Text = "mutated"

	gotConvAgain, err := st.GetConversation(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if gotConvAgain.UserID != "user-1" {
		t.Fatalf("internal conversation state was mutated")
	}
	gotMsgsAgain, err := st.ListMessages(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if gotMsgsAgain[0].Text != "hello" {
		t.Fatalf("internal message state was mutated")
	}
}

func TestConcurrentCreateAppendList(t *testing.T) {
	st := NewMemoryStore()
	ctx := context.Background()
	conv, err := st.CreateConversation(ctx, "user-1", "code-agent")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	const workers = 50
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := st.AppendMessage(ctx, Message{
				ConversationID: conv.ID,
				Author:         "user",
				Role:           "user",
				Text:           "load",
			}); err != nil {
				t.Errorf("append failed: %v", err)
				return
			}
			if _, err := st.ListConversations(ctx, "user-1"); err != nil {
				t.Errorf("list conversations failed: %v", err)
			}
			if _, err := st.ListMessages(ctx, conv.ID); err != nil {
				t.Errorf("list messages failed: %v", err)
			}
		}()
	}
	wg.Wait()

	msgs, err := st.ListMessages(ctx, conv.ID)
	if err != nil {
		t.Fatalf("list messages failed: %v", err)
	}
	if len(msgs) != workers {
		t.Fatalf("expected %d messages, got %d", workers, len(msgs))
	}
}
