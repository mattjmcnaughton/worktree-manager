package hooks

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattjmcnaughton/worktree-manager/internal/config"
)

func TestRunCopyHookCopiesFile(t *testing.T) {
	main := t.TempDir()
	worktree := t.TempDir()
	if err := os.WriteFile(filepath.Join(main, ".env"), []byte("X=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Hooks: config.Hooks{PostCreate: []config.Hook{
		{Type: "copy", From: ".env", To: ".env"},
	}}}
	exec := NewExecutor(cfg, main)

	var buf bytes.Buffer
	err := exec.Run(PhasePostCreate, Context{
		Slug:         "demo",
		Branch:       "u/demo",
		WorktreePath: worktree,
		MainPath:     main,
	}, &buf)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(worktree, ".env"))
	if err != nil || string(got) != "X=1\n" {
		t.Errorf("copy result: %v / %q", err, got)
	}
}

func TestRunCopyHookCopiesDirectoryRecursively(t *testing.T) {
	main := t.TempDir()
	worktree := t.TempDir()
	if err := os.MkdirAll(filepath.Join(main, ".claude", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(main, ".claude", "top.md"), []byte("top\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(main, ".claude", "nested", "deep.md"), []byte("deep\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Hooks: config.Hooks{PostCreate: []config.Hook{
		{Type: "copy", From: ".claude"},
	}}}
	exec := NewExecutor(cfg, main)

	var buf bytes.Buffer
	if err := exec.Run(PhasePostCreate, Context{WorktreePath: worktree, MainPath: main}, &buf); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, rel := range []string{".claude/top.md", ".claude/nested/deep.md"} {
		if _, err := os.Stat(filepath.Join(worktree, rel)); err != nil {
			t.Errorf("expected %s under worktree: %v", rel, err)
		}
	}
}

func TestRunCopyHookOptionalSkipsMissingSource(t *testing.T) {
	main := t.TempDir()
	worktree := t.TempDir()

	cfg := &config.Config{Hooks: config.Hooks{PostCreate: []config.Hook{
		{Type: "copy", From: ".env", To: ".env", Optional: true},
	}}}
	exec := NewExecutor(cfg, main)

	var buf bytes.Buffer
	err := exec.Run(PhasePostCreate, Context{WorktreePath: worktree, MainPath: main}, &buf)
	if err != nil {
		t.Errorf("optional copy with missing source should not fail; got %v", err)
	}
}

func TestRunCopyHookRequiredFailsOnMissingSource(t *testing.T) {
	main := t.TempDir()
	worktree := t.TempDir()

	cfg := &config.Config{Hooks: config.Hooks{PostCreate: []config.Hook{
		{Type: "copy", From: ".env", To: ".env"},
	}}}
	exec := NewExecutor(cfg, main)

	var buf bytes.Buffer
	err := exec.Run(PhasePostCreate, Context{WorktreePath: worktree, MainPath: main}, &buf)
	if err == nil {
		t.Errorf("required copy with missing source should fail")
	}
}

func TestRunSymlinkHookCreatesLink(t *testing.T) {
	main := t.TempDir()
	worktree := t.TempDir()
	target := filepath.Join(main, "data")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Hooks: config.Hooks{PostCreate: []config.Hook{
		{Type: "symlink", From: "data", To: "data"},
	}}}
	exec := NewExecutor(cfg, main)

	var buf bytes.Buffer
	if err := exec.Run(PhasePostCreate, Context{WorktreePath: worktree, MainPath: main}, &buf); err != nil {
		t.Fatalf("Run: %v", err)
	}
	link := filepath.Join(worktree, "data")
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink at %s", link)
	}
}

func TestRunCommandHookInjectsEnvAndUsesWorkDir(t *testing.T) {
	main := t.TempDir()
	worktree := t.TempDir()
	if err := os.WriteFile(filepath.Join(worktree, "marker"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Hooks: config.Hooks{PostCreate: []config.Hook{
		{
			Type:    "command",
			Command: "echo slug=$WORKTREE_MANAGER_SLUG branch=$WORKTREE_MANAGER_BRANCH phase=$WORKTREE_MANAGER_PHASE && pwd > pwd.txt",
			WorkDir: "worktree",
		},
	}}}
	exec := NewExecutor(cfg, main)

	var buf bytes.Buffer
	err := exec.Run(PhasePostCreate, Context{
		Slug:         "demo",
		Branch:       "u/demo",
		WorktreePath: worktree,
		MainPath:     main,
	}, &buf)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(buf.String(), "slug=demo") {
		t.Errorf("expected slug=demo in output, got: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "branch=u/demo") {
		t.Errorf("expected branch=u/demo in output, got: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "phase=post_create") {
		t.Errorf("expected phase=post_create in output, got: %q", buf.String())
	}
	pwd, err := os.ReadFile(filepath.Join(worktree, "pwd.txt"))
	if err != nil {
		t.Fatalf("read pwd.txt: %v", err)
	}
	resolvedWt, _ := filepath.EvalSymlinks(worktree)
	resolvedGot, _ := filepath.EvalSymlinks(strings.TrimSpace(string(pwd)))
	if resolvedGot != resolvedWt {
		t.Errorf("expected cwd=%s, got %s", resolvedWt, resolvedGot)
	}
}

func TestRunCommandHookOptionalSwallowsFailure(t *testing.T) {
	cfg := &config.Config{Hooks: config.Hooks{PostCreate: []config.Hook{
		{Type: "command", Command: "exit 7", Optional: true},
	}}}
	exec := NewExecutor(cfg, t.TempDir())

	var buf bytes.Buffer
	err := exec.Run(PhasePostCreate, Context{WorktreePath: t.TempDir(), MainPath: t.TempDir()}, &buf)
	if err != nil {
		t.Errorf("optional command failure should not bubble up; got %v", err)
	}
}

func TestRunPhaseSelectsCorrectHooks(t *testing.T) {
	cfg := &config.Config{Hooks: config.Hooks{
		PreCreate:  []config.Hook{{Type: "command", Command: "echo pre_create"}},
		PostCreate: []config.Hook{{Type: "command", Command: "echo post_create"}},
		PreDelete:  []config.Hook{{Type: "command", Command: "echo pre_delete"}},
		PostDelete: []config.Hook{{Type: "command", Command: "echo post_delete"}},
	}}
	exec := NewExecutor(cfg, t.TempDir())

	for _, p := range []Phase{PhasePreCreate, PhasePostCreate, PhasePreDelete, PhasePostDelete} {
		var buf bytes.Buffer
		if err := exec.Run(p, Context{WorktreePath: t.TempDir(), MainPath: t.TempDir()}, &buf); err != nil {
			t.Errorf("phase %s: %v", p, err)
		}
		if !strings.Contains(buf.String(), string(p)) {
			t.Errorf("phase %s output missing marker: %q", p, buf.String())
		}
	}
}
