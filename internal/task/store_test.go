package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	repoRoot := t.TempDir()
	store := NewStore(repoRoot)

	in := &Task{
		Slug:          "add-semantic-indexing",
		Branch:        "matt/add-semantic-indexing",
		WorktreePath:  filepath.Join(repoRoot, ".worktrees", "foo"),
		WorkspacePath: filepath.Join(repoRoot, ".agentic", "add-semantic-indexing"),
		Base:          "main",
	}
	if err := store.Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Get(in.Slug)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Slug != in.Slug || got.Branch != in.Branch || got.Base != in.Base {
		t.Errorf("round-trip mismatch: got %+v want %+v", got, in)
	}
	if got.CreatedAt.IsZero() {
		t.Errorf("CreatedAt should be populated")
	}

	all, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 task, got %d", len(all))
	}

	if err := store.Delete(in.Slug); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get(in.Slug); err == nil {
		t.Errorf("expected Get to fail after Delete")
	}
}

func TestGetReturnsErrNotFound(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.Get("missing"); err == nil {
		t.Errorf("expected error for missing task")
	}
}

func TestEnsureGitignoreAppendIdempotent(t *testing.T) {
	repoRoot := t.TempDir()
	gitignore := filepath.Join(repoRoot, ".gitignore")
	if err := os.WriteFile(gitignore, []byte("bin/\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	store := NewStore(repoRoot)
	if err := store.EnsureGitignore(); err != nil {
		t.Fatalf("EnsureGitignore: %v", err)
	}
	if err := store.EnsureGitignore(); err != nil {
		t.Fatalf("EnsureGitignore (second call): %v", err)
	}

	data, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatal(err)
	}
	count := strings.Count(string(data), ".worktree-manager/")
	if count != 1 {
		t.Errorf("expected .worktree-manager/ to appear once, found %d times in:\n%s", count, data)
	}
}

func TestEnsureGitignoreCreatesFileIfMissing(t *testing.T) {
	repoRoot := t.TempDir()
	store := NewStore(repoRoot)
	if err := store.EnsureGitignore(); err != nil {
		t.Fatalf("EnsureGitignore: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatalf("expected .gitignore created: %v", err)
	}
	if !strings.Contains(string(data), ".worktree-manager/") {
		t.Errorf("expected .gitignore to contain .worktree-manager/, got:\n%s", data)
	}
}
