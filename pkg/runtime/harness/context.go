package harness

import (
	"errors"
	"fmt"
)

var ErrContextTooLarge = errors.New("required context exceeds budget")

// ContextItem is a provider-neutral unit of model context.
type ContextItem struct {
	Role       string `json:"role"` // system|user|assistant|tool
	Text       string `json:"text"`
	ToolCallID string `json:"toolCallId,omitempty"`
	ToolName   string `json:"toolName,omitempty"`
	ToolResult bool   `json:"toolResult,omitempty"`
	Pinned     bool   `json:"pinned,omitempty"`
}

func (i ContextItem) chars() int { return len(i.Text) }

type ContextLimits struct {
	MaxChars  int
	KeepFirst int
	KeepLast  int
}

func DefaultContextLimits() ContextLimits {
	return ContextLimits{MaxChars: 120_000, KeepFirst: 3, KeepLast: 20}
}

// BuildContext selects a bounded window while preserving completed tool
// call-result pairs, pinned items and system constraints. It returns items in
// original order. An orphan tool result is dropped rather than presented as a
// fabricated pair.
func BuildContext(items []ContextItem, limits ContextLimits) ([]ContextItem, error) {
	if limits.MaxChars <= 0 {
		return nil, fmt.Errorf("MaxChars must be positive")
	}
	if limits.KeepFirst < 0 || limits.KeepLast < 0 {
		return nil, fmt.Errorf("keep window must not be negative")
	}

	selected := make([]bool, len(items))
	for i, item := range items {
		if item.Pinned || item.Role == "system" {
			selected[i] = true
		}
	}
	for i := 0; i < len(items) && i < limits.KeepFirst; i++ {
		selected[i] = true
	}
	start := len(items) - limits.KeepLast
	if start < 0 {
		start = 0
	}
	for i := start; i < len(items); i++ {
		selected[i] = true
	}

	// Keep pairs together, and drop orphan tool results.
	for i, item := range items {
		if item.ToolCallID == "" {
			continue
		}
		if item.ToolResult {
			call := findCall(items, item.ToolCallID)
			if call < 0 {
				selected[i] = false
				continue
			}
			if selected[i] {
				selected[call] = true
			}
		}
	}
	for i, item := range items {
		if !item.ToolResult && item.ToolCallID != "" && selected[i] {
			if result := findResult(items, item.ToolCallID); result >= 0 {
				selected[result] = true
			}
		}
	}

	total := selectedChars(items, selected)
	for total > limits.MaxChars {
		dropped := false
		for i := 0; i < len(items) && total > limits.MaxChars; i++ {
			if !selected[i] || items[i].Pinned || items[i].Role == "system" {
				continue
			}
			drop := []int{i}
			if items[i].ToolCallID != "" {
				if items[i].ToolResult {
					if call := findCall(items, items[i].ToolCallID); call >= 0 && !items[call].Pinned && items[call].Role != "system" {
						drop = append(drop, call)
					}
				} else if result := findResult(items, items[i].ToolCallID); result >= 0 && !items[result].Pinned && items[result].Role != "system" {
					drop = append(drop, result)
				}
			}
			for _, idx := range drop {
				if selected[idx] {
					selected[idx] = false
					total -= items[idx].chars()
				}
			}
			dropped = true
		}
		if !dropped {
			break
		}
	}
	if total > limits.MaxChars {
		return nil, ErrContextTooLarge
	}

	out := make([]ContextItem, 0, len(items))
	for i, item := range items {
		if selected[i] {
			out = append(out, item)
		}
	}
	return out, nil
}

func findCall(items []ContextItem, callID string) int {
	for i, item := range items {
		if !item.ToolResult && item.ToolCallID == callID {
			return i
		}
	}
	return -1
}

func findResult(items []ContextItem, callID string) int {
	for i, item := range items {
		if item.ToolResult && item.ToolCallID == callID {
			return i
		}
	}
	return -1
}

func selectedChars(items []ContextItem, selected []bool) int {
	total := 0
	for i, item := range items {
		if selected[i] {
			total += item.chars()
		}
	}
	return total
}
