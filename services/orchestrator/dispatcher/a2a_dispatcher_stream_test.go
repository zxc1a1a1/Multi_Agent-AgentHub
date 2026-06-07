package dispatcher

import (
	"context"
	"fmt"
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
