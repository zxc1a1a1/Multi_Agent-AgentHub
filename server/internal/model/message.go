package model

import "time"

// Message represents a single message in a conversation
type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderType     string    `json:"senderType"`
	SenderName     string    `json:"senderName"`
	Content        string    `json:"content"`
	Artifacts      string    `json:"artifacts"`
	AGUIRunID      string    `json:"aguiRunId"`
	CreatedAt      time.Time `json:"createdAt"`
}

// ArtifactData represents a single artifact for JSON storage
type ArtifactData struct {
	Type     string            `json:"type"`
	Title    string            `json:"title"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata,omitempty"`
}
