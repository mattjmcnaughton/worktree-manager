//go:build integration

package cli

import (
	"bytes"
	"io"
	"testing"
)

func TestRunShellLaunchesShellInWorktree(t *testing.T) {
	repoDir := setupRepoWithCreate(t, "")
	slug := "demo-task"
	wantPath := worktreePathFor(repoDir, slug)

	var gotShell, gotCwd string
	orig := shellLauncher
	t.Cleanup(func() { shellLauncher = orig })
	shellLauncher = func(shell, cwd string, _ io.Reader, _, _ io.Writer) error {
		gotShell = shell
		gotCwd = cwd
		return nil
	}
	t.Setenv("SHELL", "/bin/zsh")

	var buf bytes.Buffer
	if err := runShell(slug, repoDir, &buf, &buf, &buf); err != nil {
		t.Fatalf("runShell: %v", err)
	}
	if gotShell != "/bin/zsh" {
		t.Errorf("shell = %q, want %q", gotShell, "/bin/zsh")
	}
	if gotCwd != wantPath {
		t.Errorf("cwd = %q, want %q", gotCwd, wantPath)
	}
}

func TestRunShellFallsBackToBinBashWhenShellEnvEmpty(t *testing.T) {
	repoDir := setupRepoWithCreate(t, "")
	slug := "demo-task"

	var gotShell string
	orig := shellLauncher
	t.Cleanup(func() { shellLauncher = orig })
	shellLauncher = func(shell, _ string, _ io.Reader, _, _ io.Writer) error {
		gotShell = shell
		return nil
	}
	t.Setenv("SHELL", "")

	var buf bytes.Buffer
	if err := runShell(slug, repoDir, &buf, &buf, &buf); err != nil {
		t.Fatalf("runShell: %v", err)
	}
	if gotShell != "/bin/bash" {
		t.Errorf("shell = %q, want %q (fallback)", gotShell, "/bin/bash")
	}
}

func TestRunShellUnknownSlugErrors(t *testing.T) {
	repoDir := initTestRepo(t)

	called := false
	orig := shellLauncher
	t.Cleanup(func() { shellLauncher = orig })
	shellLauncher = func(string, string, io.Reader, io.Writer, io.Writer) error {
		called = true
		return nil
	}

	var buf bytes.Buffer
	if err := runShell("nope", repoDir, &buf, &buf, &buf); err == nil {
		t.Errorf("expected error for unknown slug")
	}
	if called {
		t.Errorf("launcher should not run when the slug fails to resolve")
	}
}
