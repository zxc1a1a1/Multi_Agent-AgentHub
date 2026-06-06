package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// AgentCard represents the /.well-known/agent.json response from a child agent.
type AgentCard struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Version       string   `json:"version"`
	URL           string   `json:"url"`
	Skills        []string `json:"skills"`
	InputModes    []string `json:"inputModes"`
	OutputModes   []string `json:"outputModes"`
	Streaming     bool     `json:"streaming"`
}

// LoadAgentCard fetches the /.well-known/agent.json from the given base URL.
// Returns nil and an error on failure; callers should fall back to hardcoded defaults.
func LoadAgentCard(baseURL string) (*AgentCard, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/.well-known/agent.json", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch agent card: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent card returned status %d", resp.StatusCode)
	}

	var card AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, fmt.Errorf("decode agent card: %w", err)
	}

	return &card, nil
}

// LoadAllAgentCards fetches agent cards for all endpoints and merges their
// capabilities into the registry. Failures are logged and the hardcoded defaults
// are kept for agents whose cards could not be fetched.
func LoadAllAgentCards(endpoints []AgentEndpoint) []AgentEndpoint {
	if len(endpoints) == 0 {
		return nil
	}

	result := make([]AgentEndpoint, 0, len(endpoints))
	for _, ep := range endpoints {
		card, err := LoadAgentCard(ep.URL)
		if err != nil {
			log.Printf("[agent-card] failed to load card for %q at %s: %v, using hardcoded defaults", ep.Name, ep.URL, err)
			result = append(result, ep)
			continue
		}

		log.Printf("[agent-card] loaded card for %q (version=%s, streaming=%v)", card.Name, card.Version, card.Streaming)

		// Merge card data with hardcoded fallback for fields not in the card.
		merged := AgentEndpoint{
			Name:          ep.Name,
			URL:           ep.URL,
			Description:   firstNonEmpty(card.Description, ep.Description),
			CapabilityIDs: firstNonEmptySlice(card.Skills, ep.CapabilityIDs),
			OutputModes:   firstNonEmptySlice(card.OutputModes, ep.OutputModes),
			OutputTypes:   firstNonEmptySlice(card.OutputModes, ep.OutputTypes),
		}
		result = append(result, merged)
	}
	return result
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func firstNonEmptySlice(a, b []string) []string {
	if len(a) > 0 {
		return a
	}
	return b
}
