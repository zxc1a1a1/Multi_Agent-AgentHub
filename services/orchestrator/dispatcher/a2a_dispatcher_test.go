package dispatcher

import (
	"context"
	"testing"
)

func TestDispatchMissingURL(t *testing.T) {
	d := NewA2ADispatcher()
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       "",
		AgentName:      "test-agent",
		ConversationID: "conv-1",
		Message:        "hello",
	})
	if err == nil {
		t.Error("expected error for missing URL")
	}
}

func TestDispatchMissingConversationID(t *testing.T) {
	d := NewA2ADispatcher()
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       "http://example.com",
		AgentName:      "test-agent",
		ConversationID: "",
		Message:        "hello",
	})
	if err == nil {
		t.Error("expected error for missing conversationID")
	}
}

func TestDispatchMissingMessage(t *testing.T) {
	d := NewA2ADispatcher()
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       "http://example.com",
		AgentName:      "test-agent",
		ConversationID: "conv-1",
		Message:        "",
	})
	if err == nil {
		t.Error("expected error for missing message")
	}
}

func TestDispatchNilDispatcher(t *testing.T) {
	var d *A2ADispatcher
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       "http://example.com",
		AgentName:      "test-agent",
		ConversationID: "conv-1",
		Message:        "hello",
	})
	if err == nil {
		t.Error("expected error for nil dispatcher")
	}
}

func TestDispatcherBadRequest(t *testing.T) {
	d := NewA2ADispatcher()
	// Use a non-routable IP to simulate connection failure.
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       "http://10.255.255.1:1",
		AgentName:      "test-agent",
		ConversationID: "conv-1",
		Message:        "hello",
	})
	if err == nil {
		t.Error("expected error for bad agent URL")
	}
}

func TestNewA2ADispatcher(t *testing.T) {
	d := NewA2ADispatcher()
	if d == nil {
		t.Fatal("expected non-nil dispatcher")
	}
	if d.client == nil {
		t.Error("expected non-nil client in dispatcher")
	}
}
