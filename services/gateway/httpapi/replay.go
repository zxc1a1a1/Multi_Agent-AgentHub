package httpapi

import (
	"encoding/json"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/persistence/sqlite"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

// ReplayMessage is the enriched API response format for GET /api/conversations/{id}/messages.
// It carries sender identity, run/step linkage, status, and error fields so the frontend can
// reconstruct multi-agent message bubbles after a page refresh.
type ReplayMessage struct {
	ID               string          `json:"id"`
	ConversationID   string          `json:"conversationId"`
	RunID            string          `json:"runId,omitempty"`
	StepID           string          `json:"stepId,omitempty"`
	SSEMessageID     string          `json:"sseMessageId,omitempty"`
	SenderType       string          `json:"senderType"`
	SenderName       string          `json:"senderName,omitempty"`
	SenderDisplayName string         `json:"senderDisplayName,omitempty"`
	AgentName        string          `json:"agentName,omitempty"`
	Role             string          `json:"role"`
	Author           string          `json:"author"`
	Content          string          `json:"content"`
	Text             string          `json:"text"`
	Status           string          `json:"status"`
	ErrorCode        string          `json:"errorCode,omitempty"`
	ErrorMessage     string          `json:"errorMessage,omitempty"`
	Artifacts        json.RawMessage `json:"artifacts,omitempty"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt,omitempty"`
}

// sqliteToReplayMessages converts SQLite-backed messages to the enriched replay format.
// It preserves sender identity (senderType, senderName, senderDisplayName, agentName),
// run/step linkage, status, error fields, and artifact metadata.
func sqliteToReplayMessages(msgs []sqlite.Message) []ReplayMessage {
	out := make([]ReplayMessage, 0, len(msgs))
	for _, m := range msgs {
		sType := m.SenderType
		if sType == "" {
			sType = inferSenderType(m.Role, m.SenderName)
		}
		sName := m.SenderName
		if sName == "" && sType == "agent" {
			sName = m.AgentName
		}

		rm := ReplayMessage{
			ID:                m.ID,
			ConversationID:    m.ConversationID,
			RunID:             m.RunID,
			StepID:            m.StepID,
			SSEMessageID:      m.MessageID,
			SenderType:        sType,
			SenderName:        sName,
			SenderDisplayName: senderDisplayName(sType, sName, m.AgentName),
			AgentName:         m.AgentName,
			Role:              m.Role,
			Author:            sName,
			Content:           m.Content,
			Text:              m.Content,
			Status:            m.Status,
			ErrorCode:         m.ErrorCode,
			ErrorMessage:      m.ErrorMessage,
			CreatedAt:         m.CreatedAt,
			UpdatedAt:         m.UpdatedAt,
		}

		if artifacts := parseArtifactsFromMetadata(m.MetadataJSON); artifacts != nil {
			rm.Artifacts = artifacts
		}

		out = append(out, rm)
	}
	return out
}

// memoryStoreToReplayMessages converts MemoryStore-backed messages to the replay format.
// This provides backward compatibility: the fields author/role/text are preserved so
// older frontends that only read author/text continue to work.
func memoryStoreToReplayMessages(msgs []store.Message) []ReplayMessage {
	out := make([]ReplayMessage, 0, len(msgs))
	for _, m := range msgs {
		sType := "user"
		sName := ""
		if m.Role == "assistant" || m.Author == "assistant" {
			sType = "agent"
			sName = m.Author
		}
		if m.Role == "user" || m.Author == "user" {
			sType = "user"
			sName = ""
		}

		author := m.Author
		if author == "" {
			author = sName
		}

		rm := ReplayMessage{
			ID:                m.ID,
			ConversationID:    m.ConversationID,
			SenderType:        sType,
			SenderName:        sName,
			SenderDisplayName: senderDisplayName(sType, sName, ""),
			AgentName:         m.Author,
			Role:              m.Role,
			Author:            author,
			Content:           m.Text,
			Text:              m.Text,
			Status:            "sent",
			CreatedAt:         m.CreatedAt,
		}
		out = append(out, rm)
	}
	return out
}

// senderDisplayName resolves a human-readable display name from sender type, name, and agent name.
func senderDisplayName(senderType, senderName, agentName string) string {
	if senderType == "user" {
		return ""
	}
	// Prefer senderName if it's already a display name (e.g. from event.Sender.DisplayName).
	// If senderName looks like a raw name (e.g. "web-agent", "code-agent"), map it.
	name := senderName
	if name == "" {
		name = agentName
	}
	if name == "" {
		return "Assistant"
	}
	return knownDisplayName(name)
}

// knownDisplayName maps internal agent IDs to human-readable display names.
func knownDisplayName(name string) string {
	switch name {
	case "web-agent", "Web Agent":
		return "Web Agent"
	case "code-agent", "Code Agent":
		return "Code Agent"
	case "orchestrator", "Orchestrator":
		return "Orchestrator"
	case "user", "":
		return ""
	case "assistant":
		return "Assistant"
	default:
		return name
	}
}

func inferSenderType(role, senderName string) string {
	if role == "user" {
		return "user"
	}
	if senderName == "user" {
		return "user"
	}
	return "agent"
}

// parseArtifactsFromMetadata extracts the artifacts array from a metadata JSON string.
// Returns nil if metadata is empty or does not contain a valid artifacts array.
func parseArtifactsFromMetadata(metadataJSON string) json.RawMessage {
	if metadataJSON == "" || metadataJSON == "{}" {
		return nil
	}
	var meta map[string]json.RawMessage
	if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
		return nil
	}
	artifacts, ok := meta["artifacts"]
	if !ok || len(artifacts) == 0 {
		return nil
	}
	// Validate it's a JSON array.
	var arr []json.RawMessage
	if err := json.Unmarshal(artifacts, &arr); err != nil {
		return nil
	}
	return artifacts
}
