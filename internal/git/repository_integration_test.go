//go:build integration

package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestOpenAndListWorktreesAgainstRealRepo(t *testing.T) {
	repoDir := initTestRepo(t)

	repo, err := Open(repoDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got := repo.MainWorktreePath(); got != repoDir {
		t.Errorf("MainWorktreePath = %q, want %q", got, repoDir)
	}
	if got := repo.RepoName(); got != filepath.Base(repoDir) {
		t.Errorf("RepoName = %q, want %q", got, filepath.Base(repoDir))
	}

	wts, err := repo.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	if len(wts) != 1 {
		t.Fatalf("expected 1 worktree, got %d: %+v", len(wts), wts)
	}
	if !wts[0].IsMain {
		t.Errorf("first worktree should be marked IsMain")
	}
}

func TestAddAndRemoveWorktree(t *testing.T) {
	repoDir := initTestRepo(t)
	repo, err := Open(repoDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	wtPath := filepath.Join(t.TempDir(), "feature")
	if err := repo.AddWorktree(wtPath, "u/feature", ""); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}

	wts, err := repo.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	if len(wts) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(wts))
	}

	dirty, err := repo.HasUncommittedChanges(wtPath)
	if err != nil || dirty {
		t.Errorf("expected clean worktree, dirty=%v err=%v", dirty, err)
	}

	// Dirty it
	if err := os.WriteFile(filepath.Join(wtPath, "x.txt"), []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	dirty, err = repo.HasUncommittedChanges(wtPath)
	if err != nil || !dirty {
		t.Errorf("expected dirty worktree, dirty=%v err=%v", dirty, err)
	}

	if err := repo.RemoveWorktree(wtPath, true); err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}

	exists, err := repo.BranchExists("u/feature")
	if err != nil {
		t.Fatalf("BranchExists: %v", err)
	}
	if !exists {
		t.Errorf("expected branch to still exist after worktree removal")
	}
	if err := repo.DeleteBranch("u/feature", true); err != nil {
		t.Fatalf("DeleteBranch: %v", err)
	}
	exists, _ = repo.BranchExists("u/feature")
	if exists {
		t.Errorf("expected branch to be gone")
	}
}

func TestHasUnpushedCommitsNoUpstream(t *testing.T) {
	repoDir := initTestRepo(t)
	repo, err := Open(repoDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, err := repo.HasUnpushedCommits(repoDir, "")
	if err != nil {
		t.Fatalf("HasUnpushedCommits: %v", err)
	}
	if !got {
		t.Errorf("expected unpushed=true when no upstream is configured")
	}
}

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-b", "main")
	run("config", "user.name", "Test")
	run("config", "user.email", "test@example.com")
	run("config", "commit.gpgsign", "false")
	run("commit", "--allow-empty", "-m", "initial")
	return dir
}
