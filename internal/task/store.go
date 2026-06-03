// Package task persists worktree-manager task metadata under
// <repo>/.worktree-manager/tasks/<slug>.json.
package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	stateDir      = ".worktree-manager"
	tasksSubdir   = "tasks"
	gitignoreLine = ".worktree-manager/"
)

// Task is the on-disk record for a managed worktree task.
type Task struct {
	Slug          string    `json:"slug"`
	Branch        string    `json:"branch"`
	Base          string    `json:"base,omitempty"`
	WorktreePath  string    `json:"worktree_path"`
	WorkspacePath string    `json:"workspace_path,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// Store wraps the on-disk task store rooted at <repoRoot>/.worktree-manager/.
type Store struct {
	repoRoot string
}

// NewStore returns a Store anchored at repoRoot.
func NewStore(repoRoot string) *Store {
	return &Store{repoRoot: repoRoot}
}

func (s *Store) dir() string {
	return filepath.Join(s.repoRoot, stateDir, tasksSubdir)
}

func (s *Store) path(slug string) string {
	return filepath.Join(s.dir(), slug+".json")
}

// Save writes the task record, creating the state directory if needed.
func (s *Store) Save(t *Task) error {
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	if err := os.MkdirAll(s.dir(), 0o755); err != nil {
		return fmt.Errorf("create task dir: %w", err)
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	if err := os.WriteFile(s.path(t.Slug), data, 0o600); err != nil {
		return fmt.Errorf("write task: %w", err)
	}
	return nil
}

// Get reads a task by slug.
func (s *Store) Get(slug string) (*Task, error) {
	data, err := os.ReadFile(s.path(slug))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("task %q not found", slug)
		}
		return nil, fmt.Errorf("read task %s: %w", slug, err)
	}
	var t Task
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parse task %s: %w", slug, err)
	}
	return &t, nil
}

// Delete removes the task file. Missing files are not an error.
func (s *Store) Delete(slug string) error {
	if err := os.Remove(s.path(slug)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete task %s: %w", slug, err)
	}
	return nil
}

// List returns every task stored under .worktree-manager/tasks/.
func (s *Store) List() ([]*Task, error) {
	entries, err := os.ReadDir(s.dir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read task dir: %w", err)
	}
	var out []*Task
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		slug := strings.TrimSuffix(e.Name(), ".json")
		t, err := s.Get(slug)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// EnsureGitignore appends ".worktree-manager/" to <repoRoot>/.gitignore if it
// is not already present. Idempotent. Creates the file when missing.
func (s *Store) EnsureGitignore() error {
	path := filepath.Join(s.repoRoot, ".gitignore")
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read .gitignore: %w", err)
	}
	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSpace(line) == gitignoreLine {
			return nil
		}
	}
	var buf strings.Builder
	buf.Write(existing)
	if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
		buf.WriteString("\n")
	}
	buf.WriteString(gitignoreLine)
	buf.WriteString("\n")
	if err := os.WriteFile(path, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}
	return nil
}
