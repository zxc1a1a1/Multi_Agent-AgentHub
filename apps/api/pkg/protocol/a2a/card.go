package a2a

// AgentCard is a starter representation of an A2A Agent Card.
// In production, prefer the official A2A Go SDK type.
type AgentCard struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	URL         string  `json:"url"`
	Version     string  `json:"version"`
	Skills      []Skill `json:"skills"`
}

// Skill describes one capability exposed by an A2A agent.
type Skill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
