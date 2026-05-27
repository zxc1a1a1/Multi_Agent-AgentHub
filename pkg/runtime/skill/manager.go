package skill

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
)

var ErrNilRepository = errors.New("skill repository is nil")

// Manager keeps an in-memory, concurrency-safe skill cache.
type Manager struct {
	repo Repository

	mu     sync.RWMutex
	skills map[string]Skill
}

func NewManager(repo Repository) (*Manager, error) {
	if repo == nil {
		return nil, ErrNilRepository
	}
	return &Manager{
		repo:   repo,
		skills: make(map[string]Skill),
	}, nil
}

func (m *Manager) Load(ctx context.Context) error {
	if m == nil {
		return errors.New("skill manager is nil")
	}

	loaded, err := m.repo.Load(ctx)
	if err != nil {
		return err
	}

	next := make(map[string]Skill, len(loaded))
	for _, item := range loaded {
		next[item.Name] = cloneSkill(item)
	}

	m.mu.Lock()
	m.skills = next
	m.mu.Unlock()
	return nil
}

func (m *Manager) Get(name string) (*Skill, bool) {
	if m == nil {
		return nil, false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, false
	}

	m.mu.RLock()
	item, ok := m.skills[name]
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}

	copied := cloneSkill(item)
	return &copied, true
}

func (m *Manager) List() []Skill {
	if m == nil {
		return nil
	}

	m.mu.RLock()
	out := make([]Skill, 0, len(m.skills))
	for _, item := range m.skills {
		out = append(out, cloneSkill(item))
	}
	m.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (m *Manager) Select(names ...string) []Skill {
	if m == nil {
		return nil
	}

	selected := make([]Skill, 0, len(names))
	for _, name := range names {
		item, ok := m.Get(name)
		if !ok || item == nil {
			continue
		}
		selected = append(selected, *item)
	}
	return selected
}

func (m *Manager) BuildInstructions(names ...string) string {
	var selected []Skill
	if len(names) == 0 {
		selected = m.List()
	} else {
		selected = m.Select(names...)
	}

	if len(selected) == 0 {
		return ""
	}

	var builder strings.Builder
	for i, item := range selected {
		if i > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString("[skill: ")
		builder.WriteString(item.Name)
		builder.WriteString("]\n")
		builder.WriteString(item.Instructions)
	}

	return builder.String()
}
