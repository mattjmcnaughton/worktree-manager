// Package task persists worktree-manager task metadata under
// $XDG_STATE_HOME/worktree-manager/repos/<sha8(repoRoot)>/tasks/<slug>.json,
// alongside a repo.json sidecar identifying the source repo.
package task

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

// Store wraps the on-disk task store rooted at
// $XDG_STATE_HOME/worktree-manager/repos/<sha8>/.
type Store struct {
	repoRoot string
}

// NewStore returns a Store anchored at repoRoot. The repo root is resolved to
// an absolute, symlink-followed path lazily on first read/write.
func NewStore(repoRoot string) *Store {
	return &Store{repoRoot: repoRoot}
}

// Save writes the task record, creating the state directory if needed and
// dropping a repo.json sidecar so the directory is identifiable later.
func (s *Store) Save(t *Task) error {
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	tasksDir, err := s.tasksDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		return fmt.Errorf("create task dir: %w", err)
	}
	if err := s.writeSidecar(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tasksDir, t.Slug+".json"), data, 0o600); err != nil {
		return fmt.Errorf("write task: %w", err)
	}
	return nil
}

// Get reads a task by slug.
func (s *Store) Get(slug string) (*Task, error) {
	tasksDir, err := s.tasksDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(tasksDir, slug+".json"))
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
	tasksDir, err := s.tasksDir()
	if err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(tasksDir, slug+".json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete task %s: %w", slug, err)
	}
	return nil
}

// List returns every task stored for this repo.
func (s *Store) List() ([]*Task, error) {
	tasksDir, err := s.tasksDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(tasksDir)
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

// RepoDir returns the on-disk directory used to store tasks (and the sidecar)
// for this store's repo. Exposed for tests and diagnostics.
func (s *Store) RepoDir() (string, error) {
	return s.repoDir()
}

func (s *Store) tasksDir() (string, error) {
	base, err := s.repoDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "tasks"), nil
}

func (s *Store) repoDir() (string, error) {
	state, err := stateHome()
	if err != nil {
		return "", err
	}
	resolved, err := resolveRepoRoot(s.repoRoot)
	if err != nil {
		return "", err
	}
	return filepath.Join(state, "worktree-manager", "repos", repoHash(resolved)), nil
}

func (s *Store) writeSidecar() error {
	base, err := s.repoDir()
	if err != nil {
		return err
	}
	sidecar := filepath.Join(base, "repo.json")
	if _, err := os.Stat(sidecar); err == nil {
		return nil
	}
	resolved, err := resolveRepoRoot(s.repoRoot)
	if err != nil {
		return err
	}
	payload := struct {
		Path string `json:"path"`
	}{Path: resolved}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sidecar: %w", err)
	}
	if err := os.WriteFile(sidecar, data, 0o600); err != nil {
		return fmt.Errorf("write sidecar: %w", err)
	}
	return nil
}

// stateHome returns $XDG_STATE_HOME, defaulting to $HOME/.local/state per the
// XDG Base Directory spec.
func stateHome() (string, error) {
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".local", "state"), nil
}

func resolveRepoRoot(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("absolute repo root: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	}
	return abs, nil
}

func repoHash(resolvedRoot string) string {
	sum := sha256.Sum256([]byte(resolvedRoot))
	return hex.EncodeToString(sum[:])[:8]
}
