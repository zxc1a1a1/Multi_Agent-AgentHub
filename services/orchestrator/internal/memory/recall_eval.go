package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"
)

// R1-04 evaluation surface. RecallRanker is the seam for an optional semantic
// or hybrid reranker; the default production path stays the SQL keyword recall
// (MEMORY_HYBRID_RECALL=off). The synonym ranker below is an offline stand-in
// for an embedding provider, not a production implementation.

type RecallMemory struct {
	ID         string `json:"id"`
	LogicalKey string `json:"logicalKey"`
	Content    string `json:"content"`
	VerifiedAt string `json:"verifiedAt"`
}

type RecallQuery struct {
	ID       string   `json:"id"`
	Query    string   `json:"query"`
	Relevant []string `json:"relevant"`
}

type RecallDataset struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Notice        string              `json:"notice"`
	Synonyms      map[string][]string `json:"synonyms"`
	Memories      []RecallMemory      `json:"memories"`
	Queries       []RecallQuery       `json:"queries"`
}

type HybridRecallMode string

const (
	HybridRecallOff    HybridRecallMode = "off"
	HybridRecallShadow HybridRecallMode = "shadow"
	HybridRecallOn     HybridRecallMode = "on"
)

func ParseHybridRecallMode(raw string) (HybridRecallMode, error) {
	switch HybridRecallMode(strings.ToLower(strings.TrimSpace(raw))) {
	case "", HybridRecallOff:
		return HybridRecallOff, nil
	case HybridRecallShadow:
		return HybridRecallShadow, nil
	case HybridRecallOn:
		return HybridRecallOn, nil
	default:
		return "", fmt.Errorf("unknown hybrid recall mode %q", raw)
	}
}

type RecallRanker interface {
	Name() string
	Rank(query string, candidates []RecallMemory, limit int) []RecallMemory
}

func LoadRecallDataset(path string) (*RecallDataset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var dataset RecallDataset
	if err := json.Unmarshal(data, &dataset); err != nil {
		return nil, err
	}
	if dataset.SchemaVersion != 1 || len(dataset.Memories) == 0 || len(dataset.Queries) == 0 {
		return nil, fmt.Errorf("invalid recall dataset")
	}
	return &dataset, nil
}

func recallTerms(query string) []string {
	fields := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	seen := map[string]bool{}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if len([]rune(field)) < 2 || seen[field] {
			continue
		}
		seen[field] = true
		out = append(out, field)
	}
	return out
}

func scoreAndSort(candidates []RecallMemory, terms []string, limit int) []RecallMemory {
	type scored struct {
		memory RecallMemory
		score  int
	}
	hits := make([]scored, 0, len(candidates))
	for _, memory := range candidates {
		haystack := strings.ToLower(memory.Content + " " + memory.LogicalKey)
		score := 0
		for _, term := range terms {
			if strings.Contains(haystack, term) {
				score++
			}
		}
		if score > 0 {
			hits = append(hits, scored{memory: memory, score: score})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		if hits[i].memory.VerifiedAt != hits[j].memory.VerifiedAt {
			return hits[i].memory.VerifiedAt > hits[j].memory.VerifiedAt
		}
		return hits[i].memory.ID < hits[j].memory.ID
	})
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]RecallMemory, 0, len(hits))
	for _, hit := range hits {
		out = append(out, hit.memory)
	}
	return out
}

// KeywordRanker mirrors the production SQL INSTR behavior (terms only).
type KeywordRanker struct{}

func (KeywordRanker) Name() string { return "keyword" }
func (KeywordRanker) Rank(query string, candidates []RecallMemory, limit int) []RecallMemory {
	return scoreAndSort(candidates, recallTerms(query), limit)
}

// SynonymRanker is an offline stand-in for an embedding/hybrid reranker. It
// expands query terms with a static synonym map; it is never wired to prod.
type SynonymRanker struct {
	Synonyms map[string][]string
}

func (r SynonymRanker) Name() string { return "synthetic_semantic" }
func (r SynonymRanker) Rank(query string, candidates []RecallMemory, limit int) []RecallMemory {
	terms := recallTerms(query)
	expanded := make([]string, 0, len(terms)*2)
	seen := map[string]bool{}
	for _, term := range terms {
		if !seen[term] {
			seen[term] = true
			expanded = append(expanded, term)
		}
		for _, synonym := range r.Synonyms[term] {
			lower := strings.ToLower(synonym)
			if !seen[lower] {
				seen[lower] = true
				expanded = append(expanded, lower)
			}
		}
		// Reverse direction: a synonym in the query expands to its base topic.
		for base, synonyms := range r.Synonyms {
			for _, synonym := range synonyms {
				if strings.ToLower(synonym) != term {
					continue
				}
				if !seen[base] {
					seen[base] = true
					expanded = append(expanded, base)
				}
			}
		}
	}
	return scoreAndSort(candidates, expanded, limit)
}

type RecallMetrics struct {
	Queries           int     `json:"queries"`
	RecallAt5         float64 `json:"recallAt5"`
	MRR               float64 `json:"mrr"`
	AvgInjectedTokens float64 `json:"avgInjectedTokens"`
	P95LatencyMs      float64 `json:"p95LatencyMs"`
}

func EvaluateRecall(dataset *RecallDataset, ranker RecallRanker, limit int) RecallMetrics {
	if limit <= 0 {
		limit = 5
	}
	metrics := RecallMetrics{Queries: len(dataset.Queries)}
	if len(dataset.Queries) == 0 {
		return metrics
	}
	var recallSum, mrrSum, tokenSum float64
	durations := make([]time.Duration, 0, len(dataset.Queries))
	for _, query := range dataset.Queries {
		start := time.Now()
		ranked := ranker.Rank(query.Query, dataset.Memories, limit)
		durations = append(durations, time.Since(start))
		relevant := map[string]bool{}
		for _, id := range query.Relevant {
			relevant[id] = true
		}
		hits := 0
		firstRank := 0
		tokens := 0
		for i, memory := range ranked {
			tokens += len(memory.Content)/4 + 1
			if relevant[memory.ID] {
				hits++
				if firstRank == 0 {
					firstRank = i + 1
				}
			}
		}
		if len(relevant) > 0 {
			recallSum += float64(hits) / float64(len(relevant))
		}
		if firstRank > 0 {
			mrrSum += 1 / float64(firstRank)
		}
		tokenSum += float64(tokens)
	}
	metrics.RecallAt5 = recallSum / float64(len(dataset.Queries))
	metrics.MRR = mrrSum / float64(len(dataset.Queries))
	metrics.AvgInjectedTokens = tokenSum / float64(len(dataset.Queries))
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p95 := durations[(len(durations)*95)/100]
	metrics.P95LatencyMs = float64(p95.Microseconds()) / 1000
	return metrics
}

type RecallDecision struct {
	Baseline           RecallMetrics `json:"baseline"`
	Candidate          RecallMetrics `json:"candidate"`
	RelativeRecallGain float64       `json:"relativeRecallGain"`
	ShadowGO           bool          `json:"shadowGo"`
	ProductionGO       bool          `json:"productionGo"`
	Reason             string        `json:"reason"`
}

// Decide applies the R1-04 thresholds. The candidate is a synthetic ranker, so
// production enablement always requires a separate embedding-provider review.
func Decide(baseline, candidate RecallMetrics, maxTokens int) RecallDecision {
	decision := RecallDecision{Baseline: baseline, Candidate: candidate}
	if baseline.RecallAt5 > 0 {
		decision.RelativeRecallGain = (candidate.RecallAt5 - baseline.RecallAt5) / baseline.RecallAt5
	}
	tokensOK := maxTokens <= 0 || candidate.AvgInjectedTokens <= float64(maxTokens)
	decision.ShadowGO = decision.RelativeRecallGain >= 0.20 && candidate.P95LatencyMs <= 300 && tokensOK
	decision.ProductionGO = false
	decision.Reason = "synthetic semantic ranker only; production needs a reviewed embedding provider"
	if !decision.ShadowGO {
		decision.Reason = "thresholds not met: recall gain >=20%, p95 <=300ms, token budget"
	}
	return decision
}
