// Command fake-llm is a test-only OpenAI-compatible model endpoint used by the
// browser E2E harness. It is never wired into production startup.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type chatRequest struct {
	Messages []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"messages"`
}

type chatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage map[string]int `json:"usage"`
}

func contentText(raw json.RawMessage) string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		var b strings.Builder
		for _, part := range parts {
			b.WriteString(part.Text)
		}
		return b.String()
	}
	return string(raw)
}

func main() {
	addr := flag.String("addr", ":18080", "listen address")
	flag.Parse()
	http.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		system := ""
		for _, message := range req.Messages {
			if message.Role == "system" {
				system += contentText(message.Content)
			}
		}
		content := "Fake model: task completed."
		switch {
		case strings.Contains(system, "intent orchestrator"), strings.Contains(system, "JSON repair assistant"):
			content = `{"intent":"write a Go function","mode":"single","confidence":0.9,"steps":[{"id":"step-1","agent_name":"code-agent","input":"write a Go function that adds two numbers","reason":"code task"}],"user_visible_summary":"I will ask code-agent to write the function."}`
		case strings.Contains(strings.ToLower(system), "synthes"):
			content = "Summary: code-agent produced the requested Go function."
		}
		response := chatResponse{ID: "fake-chat", Object: "chat.completion", Usage: map[string]int{"prompt_tokens": 10, "completion_tokens": 20, "total_tokens": 30}}
		response.Choices = append(response.Choices, struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{Index: 0, FinishReason: "stop"})
		response.Choices[0].Message.Role = "assistant"
		response.Choices[0].Message.Content = content
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	log.Printf("fake-llm listening on %s", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(fmt.Errorf("fake-llm: %w", err))
	}
}
