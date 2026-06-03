//go:build integration

package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCreateProducesUsableWorktreeTask(t *testing.T) {
	repoDir := initTestRepo(t)
	writeFile(t, filepath.Join(repoDir, "shared.txt"), "shared content\n")
	gitMust(t, repoDir, "add", "shared.txt")
	gitMust(t, repoDir, "commit", "-m", "add shared")

	cfg := `version: "1"
defaults:
  user: tester
  base: main
agentic:
  enabled: true
  create_task_workspace: true
hooks:
  pre_create:
    - type: command
      command: 'echo pre > "$WORKTREE_MANAGER_MAIN_WORKTREE/pre_create_ran.txt"'
      work_dir: main
  post_create:
    - type: copy
      from: shared.txt
      to: shared.txt
    - type: command
      command: echo post > post_create_ran.txt
      work_dir: worktree
`
	writeFile(t, filepath.Join(repoDir, ".worktree-manager.yml"), cfg)

	var buf bytes.Buffer
	err := runCreate(createOpts{Task: "add semantic indexing"}, repoDir, &buf)
	if err != nil {
		t.Fatalf("runCreate: %v", err)
	}

	wantSlug := "add-semantic-indexing"
	wantBranch := "tester/" + wantSlug
	repoName := filepath.Base(repoDir)
	wtPath := filepath.Join(repoDir, ".worktrees", repoName+"-"+wantSlug)

	if _, err := os.Stat(wtPath); err != nil {
		t.Fatalf("worktree path not created at %s: %v", wtPath, err)
	}
	out, _ := gitOutput(repoDir, "branch", "--list", wantBranch)
	if !strings.Contains(out, wantBranch) {
		t.Errorf("expected branch %q; `git branch --list` output: %q", wantBranch, out)
	}

	if _, err := os.Stat(filepath.Join(repoDir, ".agentic", wantSlug)); err != nil {
		t.Errorf("expected .agentic/%s: %v", wantSlug, err)
	}

	taskPath := filepath.Join(repoDir, ".worktree-manager", "tasks", wantSlug+".json")
	if _, err := os.Stat(taskPath); err != nil {
		t.Errorf("expected task metadata at %s: %v", taskPath, err)
	}

	if _, err := os.Stat(filepath.Join(repoDir, "pre_create_ran.txt")); err != nil {
		t.Errorf("pre_create hook did not run (no pre_create_ran.txt in main): %v", err)
	}

	got, err := os.ReadFile(filepath.Join(wtPath, "shared.txt"))
	if err != nil || string(got) != "shared content\n" {
		t.Errorf("post_create copy hook: err=%v contents=%q", err, got)
	}

	if _, err := os.Stat(filepath.Join(wtPath, "post_create_ran.txt")); err != nil {
		t.Errorf("post_create command hook did not run (no post_create_ran.txt in worktree): %v", err)
	}

	gi, _ := os.ReadFile(filepath.Join(repoDir, ".gitignore"))
	if !strings.Contains(string(gi), ".worktree-manager/") {
		t.Errorf("expected .worktree-manager/ in .gitignore; got %q", gi)
	}

	output := buf.String()
	for _, marker := range []string{wantSlug, wantBranch, wtPath} {
		if !strings.Contains(output, marker) {
			t.Errorf("expected create summary to contain %q; got:\n%s", marker, output)
		}
	}
}

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	gitMust(t, dir, "init", "-b", "main")
	gitMust(t, dir, "config", "user.name", "Test")
	gitMust(t, dir, "config", "user.email", "test@example.com")
	gitMust(t, dir, "config", "commit.gpgsign", "false")
	gitMust(t, dir, "commit", "--allow-empty", "-m", "initial")
	return dir
}

func gitMust(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
