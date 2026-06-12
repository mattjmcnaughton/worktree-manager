package task

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
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
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	store := NewStore(t.TempDir())
	if _, err := store.Get("missing"); err == nil {
		t.Errorf("expected error for missing task")
	}
}

func TestSaveWritesUnderXDGStateHomeWithSidecar(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateDir)
	repoRoot := t.TempDir()
	store := NewStore(repoRoot)

	if err := store.Save(&Task{Slug: "alpha", Branch: "u/alpha", WorktreePath: "/tmp/alpha"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	repoDir, err := store.RepoDir()
	if err != nil {
		t.Fatalf("RepoDir: %v", err)
	}
	if !strings.HasPrefix(repoDir, stateDir) {
		t.Errorf("repo dir %q should live under XDG state home %q", repoDir, stateDir)
	}
	if !strings.Contains(repoDir, filepath.Join("worktree-manager", "repos")) {
		t.Errorf("repo dir %q should sit under worktree-manager/repos/", repoDir)
	}

	taskPath := filepath.Join(repoDir, "tasks", "alpha.json")
	if _, err := os.Stat(taskPath); err != nil {
		t.Errorf("expected task file at %s: %v", taskPath, err)
	}

	sidecar := filepath.Join(repoDir, "repo.json")
	data, err := os.ReadFile(sidecar)
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	var payload struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("parse sidecar: %v", err)
	}
	wantPath, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		wantPath = repoRoot
	}
	if payload.Path != wantPath {
		t.Errorf("sidecar path = %q, want %q", payload.Path, wantPath)
	}
}

func TestSaveDoesNotWriteIntoRepoRoot(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	repoRoot := t.TempDir()
	store := NewStore(repoRoot)

	if err := store.Save(&Task{Slug: "beta", Branch: "u/beta", WorktreePath: "/tmp/beta"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(filepath.Join(repoRoot, ".worktree-manager")); !os.IsNotExist(err) {
		t.Errorf("expected no .worktree-manager dir inside repo; stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, ".gitignore")); !os.IsNotExist(err) {
		t.Errorf("expected no .gitignore written to repo root; stat err=%v", err)
	}
}
