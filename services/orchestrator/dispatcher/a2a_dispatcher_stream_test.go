package dispatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func collectDispatchStream(seq func(yield func(DispatchChunk) bool)) ([]string, error) {
	var texts []string
	var firstErr error
	seq(func(c DispatchChunk) bool {
		if c.Err != nil {
			if firstErr == nil {
				firstErr = c.Err
			}
			return false
		}
		if c.Text != "" {
			texts = append(texts, c.Text)
		}
		return true
	})
	return texts, firstErr
}

// TestDispatchStream_SSE verifies the dispatcher yields text chunks from an SSE agent.
func TestDispatchStream_SSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		for _, chunk := range []string{"foo", "bar", "baz"} {
			fmt.Fprintf(w, "data: {\"author\":\"a\",\"parts\":[{\"type\":\"text\",\"text\":%q}]}\n\n", chunk)
			if flusher != nil {
				flusher.Flush()
			}
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	d := NewA2ADispatcher()
	texts, err := collectDispatchStream(d.DispatchStream(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "a",
		ConversationID: "conv-1",
		Message:        "hi",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Join(texts, "") != "foobarbaz" {
		t.Fatalf("unexpected streamed text: got=%v", texts)
	}
	if len(texts) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(texts))
	}
}

// TestDispatchStream_ValidatesInput verifies input validation surfaces as a chunk error.
func TestDispatchStream_ValidatesInput(t *testing.T) {
	d := NewA2ADispatcher()
	_, err := collectDispatchStream(d.DispatchStream(context.Background(), DispatchInput{
		AgentURL:       "",
		ConversationID: "conv-1",
		Message:        "hi",
	}))
	if err == nil {
		t.Fatal("expected error for missing agent url")
	}
}




func TestDispatchInput_Mode_PropagatesToRunRequest(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "{\"taskId\":\"t1\",\"status\":\"completed\",\"events\":[{\"author\":\"a\",\"role\":\"assistant\",\"parts\":[{\"type\":\"text\",\"text\":\"ok\"}]}]}")
	}))
	defer srv.Close()

	d := NewA2ADispatcher()
	_, err := d.Dispatch(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "test-agent",
		ConversationID: "conv-1",
		Message:        "plan this",
		Mode:           "plan_only",
	})
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	var rpcReq struct {
		Params struct {
			Mode string `json:"mode"`
		} `json:"params"`
	}
	if err := json.Unmarshal(capturedBody, &rpcReq); err != nil {
		t.Fatalf("failed to decode captured request: %v", err)
	}
	if rpcReq.Params.Mode != "plan_only" {
		t.Errorf("expected RunRequest.Mode=plan_only, got=%q", rpcReq.Params.Mode)
	}
}

func TestDispatchStream_ModeDefaultEmpty_DoesNotBreakOldExecution(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		fmt.Fprint(w, "data: {\"author\":\"a\",\"parts\":[{\"type\":\"text\",\"text\":\"hello\"}]}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	d := NewA2ADispatcher()
	texts, err := collectDispatchStream(d.DispatchStream(context.Background(), DispatchInput{
		AgentURL:       srv.URL,
		AgentName:      "a",
		ConversationID: "conv-1",
		Message:        "hi",
	}))
	if err != nil {
		t.Fatalf("unexpected error with default mode: %v", err)
	}
	if len(texts) == 0 {
		t.Fatal("expected at least one text chunk")
	}
}
