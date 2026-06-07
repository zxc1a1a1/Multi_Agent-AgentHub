package a2a

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// collectStream drains a SendJSONRPCStream sequence into texts and the first error.
func collectStream(seq func(yield func(StreamChunk) bool)) ([]string, error) {
	var texts []string
	var firstErr error
	seq(func(c StreamChunk) bool {
		if c.Err != nil {
			if firstErr == nil {
				firstErr = c.Err
			}
			return false
		}
		for _, p := range c.Event.Parts {
			if p.Type == "text" && p.Text != "" {
				texts = append(texts, p.Text)
			}
		}
		return true
	})
	return texts, firstErr
}

// TestSendJSONRPCStream_SSE verifies the client parses SSE frames and yields
// each EventDTO as it arrives.
func TestSendJSONRPCStream_SSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		for _, chunk := range []string{"Hello", " ", "world"} {
			fmt.Fprintf(w, "data: {\"author\":\"code-agent\",\"role\":\"assistant\",\"parts\":[{\"type\":\"text\",\"text\":%q}]}\n\n", chunk)
			if flusher != nil {
				flusher.Flush()
			}
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	client := NewClient()
	texts, err := collectStream(client.SendJSONRPCStream(context.Background(), srv.URL, RunRequest{
		SessionID: "sess-1",
		Message:   Message{Role: "user", Content: "hi"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := strings.Join(texts, "")
	if got != "Hello world" {
		t.Fatalf("unexpected streamed text: got=%q want=%q", got, "Hello world")
	}
	if len(texts) != 3 {
		t.Fatalf("expected 3 streamed chunks, got %d (%v)", len(texts), texts)
	}
}

// TestSendJSONRPCStream_JSONFallback verifies that when the server returns a
// buffered application/json RunResponse, the client yields each event.
func TestSendJSONRPCStream_JSONFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"taskId":"t1","status":"completed","events":[`+
			`{"author":"web-agent","role":"assistant","parts":[{"type":"text","text":"part-A"}]},`+
			`{"author":"web-agent","role":"assistant","parts":[{"type":"text","text":"part-B"}]}`+
			`]}`)
	}))
	defer srv.Close()

	client := NewClient()
	texts, err := collectStream(client.SendJSONRPCStream(context.Background(), srv.URL, RunRequest{
		SessionID: "sess-2",
		Message:   Message{Role: "user", Content: "hi"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Join(texts, "|") != "part-A|part-B" {
		t.Fatalf("unexpected fallback text: got=%v", texts)
	}
}

// TestSendJSONRPCStream_ThinkingRedacted verifies thinking parts are redacted.
func TestSendJSONRPCStream_ThinkingRedacted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: {\"author\":\"a\",\"parts\":[{\"type\":\"thinking\",\"text\":\"secret reasoning\"}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	client := NewClient()
	var sawThinkingText bool
	client.SendJSONRPCStream(context.Background(), srv.URL, RunRequest{
		SessionID: "sess-3",
		Message:   Message{Role: "user", Content: "hi"},
	})(func(c StreamChunk) bool {
		for _, p := range c.Event.Parts {
			if p.Type == "thinking" && (p.Text != "" || p.Content != "") {
				sawThinkingText = true
			}
		}
		return true
	})
	if sawThinkingText {
		t.Fatal("thinking part text must be redacted in streamed output")
	}
}

// TestSendJSONRPCStream_HTTPError verifies a non-2xx status surfaces as a chunk error.
func TestSendJSONRPCStream_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":{"code":"internal","message":"boom"}}`)
	}))
	defer srv.Close()

	client := NewClient()
	_, err := collectStream(client.SendJSONRPCStream(context.Background(), srv.URL, RunRequest{
		SessionID: "sess-4",
		Message:   Message{Role: "user", Content: "hi"},
	}))
	if err == nil {
		t.Fatal("expected an error for 500 response")
	}
}
