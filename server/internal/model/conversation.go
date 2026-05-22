package model

import "time"

type Conversation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	AgentName string    `json:"agentName"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
