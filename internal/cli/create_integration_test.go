//go:build integration

package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattjmcnaughton/worktree-manager/internal/task"
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

	taskPath := taskFilePath(t, repoDir, wantSlug)
	if _, err := os.Stat(taskPath); err != nil {
		t.Errorf("expected task metadata at %s: %v", taskPath, err)
	}
	if _, err := os.Stat(filepath.Join(repoDir, ".worktree-manager")); !os.IsNotExist(err) {
		t.Errorf("expected no .worktree-manager dir in repo; stat err=%v", err)
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
	// macOS symlinks /var/folders -> /private/var/folders; git resolves them,
	// so anchor the test on the resolved path to avoid spurious mismatches.
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	gitMust(t, dir, "init", "-b", "main")
	gitMust(t, dir, "config", "user.name", "Test")
	gitMust(t, dir, "config", "user.email", "test@example.com")
	gitMust(t, dir, "config", "commit.gpgsign", "false")
	gitMust(t, dir, "commit", "--allow-empty", "-m", "initial")
	return dir
}

func TestRunCreateReadsGlobalConfigFromXDGConfigHome(t *testing.T) {
	repoDir := initTestRepo(t)

	// Point XDG_CONFIG_HOME at a path nowhere near $HOME so we can prove the
	// global config really is being resolved via the env var.
	customConfig := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(customConfig); err == nil {
		customConfig = resolved
	}
	t.Setenv("XDG_CONFIG_HOME", customConfig)

	globalDir := filepath.Join(customConfig, "worktree-manager")
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(globalDir, "config.yml"), `defaults:
  user: globalbob
agentic:
  enabled: true
`)

	var buf bytes.Buffer
	if err := runCreate(createOpts{Task: "needs global cfg"}, repoDir, &buf); err != nil {
		t.Fatalf("runCreate: %v", err)
	}

	wantBranch := "globalbob/needs-global-cfg"
	out, _ := gitOutput(repoDir, "branch", "--list", wantBranch)
	if !strings.Contains(out, wantBranch) {
		t.Errorf("expected branch %q derived from XDG global config; git branch output: %q", wantBranch, out)
	}
}

// taskFilePath returns the on-disk task metadata path for slug under the test
// run's XDG_STATE_HOME. Tests use it to assert the new XDG layout.
func taskFilePath(t *testing.T, repoDir, slug string) string {
	t.Helper()
	store := task.NewStore(repoDir)
	repoDirOnDisk, err := store.RepoDir()
	if err != nil {
		t.Fatalf("RepoDir: %v", err)
	}
	return filepath.Join(repoDirOnDisk, "tasks", slug+".json")
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
