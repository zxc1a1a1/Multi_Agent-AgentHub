package a2a

type A2AStreamEvent struct {
	Type     string      `json:"type"`
	Status   string      `json:"status,omitempty"`
	Content  string      `json:"content,omitempty"`
	Error    string      `json:"error,omitempty"`
	Artifact A2AArtifact `json:"artifact,omitempty"`
}

type A2AArtifact struct {
	Type     string            `json:"type"`
	Title    string            `json:"title"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata"`
}
