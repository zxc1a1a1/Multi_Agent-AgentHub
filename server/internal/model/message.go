package model

import "time"

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
