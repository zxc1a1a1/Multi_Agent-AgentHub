package store

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	drivermysql "github.com/go-sql-driver/mysql"
)

func newMockStore(t *testing.T) (*MySQLStore, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &MySQLStore{db: db}, mock
}

func TestAppendMessageUpdatesConversationAndInserts(t *testing.T) {
	s, mock := newMockStore(t)
	now := time.Now().UTC()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM conversations WHERE id = ? FOR UPDATE")).
		WithArgs("conv-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("conv-1"))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE conversations SET updated_at = ? WHERE id = ?")).
		WithArgs(now, "conv-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO messages").
		WithArgs("m-1", "conv-1", "user", "user-1", "hello", now).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	msg, err := s.AppendMessage(context.Background(), Message{
		ID: "m-1", ConversationID: "conv-1", Author: "user-1", Role: "user", Text: "hello", CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.ID != "m-1" {
		t.Fatalf("unexpected message: %#v", msg)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAppendMessageUnknownConversation(t *testing.T) {
	s, mock := newMockStore(t)
	now := time.Now().UTC()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM conversations WHERE id = ? FOR UPDATE")).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	_, err := s.AppendMessage(context.Background(), Message{
		ConversationID: "missing", Author: "u", Role: "user", Text: "x", CreatedAt: now,
	})
	if !errors.Is(err, ErrConversationNotFound) {
		t.Fatalf("expected ErrConversationNotFound, got %v", err)
	}
}

func TestRecordChatRequestDeduplicates(t *testing.T) {
	s, mock := newMockStore(t)
	dup := &drivermysql.MySQLError{Number: 1062, Message: "duplicate"}
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO chat_requests").WillReturnError(dup)
	mock.ExpectQuery("SELECT request_id, conversation_id").
		WithArgs("req-1").
		WillReturnRows(sqlmock.NewRows([]string{"request_id", "conversation_id", "user_message_id", "run_id", "status"}).
			AddRow("req-1", "conv-1", "m-existing", "", "accepted"))
	mock.ExpectCommit()

	req, dedup, err := s.RecordChatRequest(context.Background(), ChatRequest{
		RequestID: "req-1", ConversationID: "conv-1",
	}, Message{ConversationID: "conv-1", Author: "u", Role: "user", Text: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dedup {
		t.Fatal("expected deduplicated request")
	}
	if req.UserMessageID != "m-existing" {
		t.Fatalf("unexpected existing message id: %s", req.UserMessageID)
	}
}

func TestRecordChatRequestWritesMessageAndRequest(t *testing.T) {
	s, mock := newMockStore(t)
	now := time.Now().UTC()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO chat_requests").
		WithArgs("req-2", "conv-2", sqlmock.AnyArg(), nil).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE conversations SET updated_at").
		WithArgs(sqlmock.AnyArg(), "conv-2").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO messages").
		WithArgs(sqlmock.AnyArg(), "conv-2", "user", "u", "hello", "req-2", nil, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	req, dedup, err := s.RecordChatRequest(context.Background(), ChatRequest{
		RequestID: "req-2", ConversationID: "conv-2",
	}, Message{ConversationID: "conv-2", Author: "u", Role: "user", Text: "hello", CreatedAt: now})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dedup {
		t.Fatal("first request must not be marked deduplicated")
	}
	if req.UserMessageID == "" {
		t.Fatal("expected generated user message id")
	}
}

func TestAppendMessageRejectsEmptyText(t *testing.T) {
	s, _ := newMockStore(t)
	if _, err := s.AppendMessage(context.Background(), Message{ConversationID: "c", Author: "u", Role: "user", Text: "   "}); !errors.Is(err, ErrEmptyMessage) {
		t.Fatalf("expected ErrEmptyMessage, got %v", err)
	}
	if _, _, err := s.RecordChatRequest(context.Background(), ChatRequest{RequestID: "r", ConversationID: "c"}, Message{ConversationID: "c", Role: "user", Text: ""}); !errors.Is(err, ErrEmptyMessage) {
		t.Fatalf("expected ErrEmptyMessage, got %v", err)
	}
}
