//go:build integration

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDeleteHappyPathKeepsBranchAndAgentic(t *testing.T) {
	repoDir := setupRepoWithCreate(t, "")
	slug := "demo-task"
	wtPath := worktreePathFor(repoDir, slug)
	agenticPath := filepath.Join(repoDir, ".agentic", slug)
	taskPath := filepath.Join(repoDir, ".worktree-manager", "tasks", slug+".json")

	var buf bytes.Buffer
	if err := runDelete(deleteOpts{Slug: slug}, repoDir, &buf); err != nil {
		t.Fatalf("runDelete: %v", err)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree should be removed; stat err=%v", err)
	}
	if _, err := os.Stat(taskPath); !os.IsNotExist(err) {
		t.Errorf("task metadata should be removed; stat err=%v", err)
	}
	if _, err := os.Stat(agenticPath); err != nil {
		t.Errorf(".agentic/<slug>/ should remain by default: %v", err)
	}

	branch := "tester/" + slug
	out, _ := gitOutput(repoDir, "branch", "--list", branch)
	if !strings.Contains(out, branch) {
		t.Errorf("branch %q should remain (no --with-branch); got: %q", branch, out)
	}
}

func TestRunDeleteRefusesDirtyWithoutForce(t *testing.T) {
	repoDir := setupRepoWithCreate(t, "")
	slug := "demo-task"
	wtPath := worktreePathFor(repoDir, slug)

	writeFile(t, filepath.Join(wtPath, "dirty.txt"), "dirty\n")

	var buf bytes.Buffer
	err := runDelete(deleteOpts{Slug: slug}, repoDir, &buf)
	if err == nil {
		t.Fatalf("expected refusal due to dirty worktree, got nil")
	}
	if _, err := os.Stat(wtPath); err != nil {
		t.Errorf("worktree should still exist after refusal: %v", err)
	}

	buf.Reset()
	if err := runDelete(deleteOpts{Slug: slug, Force: true}, repoDir, &buf); err != nil {
		t.Fatalf("forced delete should succeed: %v", err)
	}
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree should be removed after force; stat err=%v", err)
	}
}

func TestRunDeleteWithBranchRemovesBranch(t *testing.T) {
	repoDir := setupRepoWithCreate(t, "")
	slug := "demo-task"

	var buf bytes.Buffer
	err := runDelete(deleteOpts{Slug: slug, WithBranch: true}, repoDir, &buf)
	if err != nil {
		t.Fatalf("runDelete with --with-branch: %v", err)
	}

	branch := "tester/" + slug
	out, _ := gitOutput(repoDir, "branch", "--list", branch)
	if strings.Contains(out, branch) {
		t.Errorf("expected branch %q deleted; got: %q", branch, out)
	}
}

func TestRunDeleteRunsHooksAndKeepsAgentic(t *testing.T) {
	hooksCfg := `hooks:
  pre_delete:
    - type: command
      command: 'echo pre > "$WORKTREE_MANAGER_MAIN_WORKTREE/pre_delete_ran.txt"'
      work_dir: main
  post_delete:
    - type: command
      command: 'echo post > "$WORKTREE_MANAGER_MAIN_WORKTREE/post_delete_ran.txt"'
      work_dir: main
`
	repoDir := setupRepoWithCreate(t, hooksCfg)
	slug := "demo-task"

	var buf bytes.Buffer
	if err := runDelete(deleteOpts{Slug: slug}, repoDir, &buf); err != nil {
		t.Fatalf("runDelete: %v", err)
	}

	if _, err := os.Stat(filepath.Join(repoDir, "pre_delete_ran.txt")); err != nil {
		t.Errorf("pre_delete artifact missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoDir, "post_delete_ran.txt")); err != nil {
		t.Errorf("post_delete artifact missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoDir, ".agentic", slug)); err != nil {
		t.Errorf(".agentic/<slug>/ should remain: %v", err)
	}
}

func TestRunDeleteUnknownSlugErrors(t *testing.T) {
	repoDir := initTestRepo(t)
	writeFile(t, filepath.Join(repoDir, ".worktree-manager.yml"), "defaults:\n  user: tester\n")

	var buf bytes.Buffer
	err := runDelete(deleteOpts{Slug: "no-such-slug"}, repoDir, &buf)
	if err == nil {
		t.Fatalf("expected error for unknown slug")
	}
}

// setupRepoWithCreate initializes a repo, writes config with the given extra
// hooks block, runs create for slug "demo-task", and returns the repo dir.
func setupRepoWithCreate(t *testing.T, extraCfg string) string {
	t.Helper()
	repoDir := initTestRepo(t)
	cfg := `version: "1"
defaults:
  user: tester
agentic:
  enabled: true
  create_task_workspace: true
`
	if extraCfg != "" {
		cfg += extraCfg
	}
	writeFile(t, filepath.Join(repoDir, ".worktree-manager.yml"), cfg)
	var buf bytes.Buffer
	if err := runCreate(createOpts{Task: "demo task"}, repoDir, &buf); err != nil {
		t.Fatalf("setupRepoWithCreate runCreate: %v", err)
	}
	return repoDir
}

func worktreePathFor(repoDir, slug string) string {
	return filepath.Join(repoDir, ".worktrees", filepath.Base(repoDir)+"-"+slug)
}
