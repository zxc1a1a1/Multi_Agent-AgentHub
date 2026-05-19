package a2ui

// Surface is the starter A2UI surface structure.
// In production, align this package with the official A2UI schema version used by the frontend.
type Surface struct {
	ID         string      `json:"id"`
	Title      string      `json:"title,omitempty"`
	Components []Component `json:"components"`
}

// Component describes one declarative UI component inside a Surface.
type Component struct {
	Type     string         `json:"type"`
	ID       string         `json:"id"`
	Props    map[string]any `json:"props,omitempty"`
	Children []Component    `json:"children,omitempty"`
}
