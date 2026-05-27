package skill

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const skillFileName = "SKILL.md"

var (
	ErrEmptySkillRoot = errors.New("skill root is required")
	ErrSkillNotFound  = errors.New("skill not found")
)

// FileRepository loads skills from local directories.
type FileRepository struct {
	root string
}

func NewFileRepository(root string) (*FileRepository, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, ErrEmptySkillRoot
	}
	return &FileRepository{root: filepath.Clean(root)}, nil
}

func (r *FileRepository) Load(ctx context.Context) ([]Skill, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, errors.New("file repository is nil")
	}

	entries, err := os.ReadDir(r.root)
	if err != nil {
		return nil, fmt.Errorf("read skill root: %w", err)
	}

	skills := make([]Skill, 0, len(entries))
	for _, entry := range entries {
		if err := contextError(ctx); err != nil {
			return nil, err
		}
		if !entry.IsDir() || isHiddenName(entry.Name()) {
			continue
		}

		skillItem, ok, loadErr := r.loadSkill(entry.Name())
		if loadErr != nil {
			return nil, loadErr
		}
		if !ok {
			continue
		}
		skills = append(skills, skillItem)
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})
	return skills, nil
}

func (r *FileRepository) Get(ctx context.Context, name string) (*Skill, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, errors.New("file repository is nil")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("skill name is required")
	}

	skills, err := r.Load(ctx)
	if err != nil {
		return nil, err
	}
	for _, loaded := range skills {
		if loaded.Name == name {
			found := cloneSkill(loaded)
			return &found, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrSkillNotFound, name)
}

func (r *FileRepository) loadSkill(dirName string) (Skill, bool, error) {
	skillDir := filepath.Join(r.root, dirName)
	skillFile := filepath.Join(skillDir, skillFileName)

	data, err := readLimitedFile(skillFile, MaxInstructionsBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Skill{}, false, nil
		}
		return Skill{}, false, fmt.Errorf("read %s: %w", skillFile, err)
	}

	instructions := string(data)
	return Skill{
		Name:         dirName,
		Description:  extractDescription(instructions),
		Instructions: instructions,
		Path:         skillDir,
	}, true, nil
}

func readLimitedFile(path string, maxBytes int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if maxBytes <= 0 {
		return io.ReadAll(file)
	}

	data, err := io.ReadAll(io.LimitReader(file, int64(maxBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxBytes {
		data = data[:maxBytes]
	}
	return data, nil
}

func extractDescription(instructions string) string {
	lines := strings.Split(instructions, "\n")
	paragraph := make([]string, 0, 4)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(paragraph) > 0 {
				break
			}
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			if heading != "" {
				return heading
			}
		}

		paragraph = append(paragraph, trimmed)
	}

	if len(paragraph) == 0 {
		return ""
	}
	return strings.Join(paragraph, " ")
}

func isHiddenName(name string) bool {
	return strings.HasPrefix(name, ".")
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
