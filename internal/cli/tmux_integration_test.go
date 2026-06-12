//go:build integration

package cli

import (
	"bytes"
	"io"
	"testing"
)

func TestRunTmuxNewSessionAttachUsesSlugAndPath(t *testing.T) {
	repoDir := setupRepoWithCreate(t, "")
	slug := "demo-task"
	wantPath := worktreePathFor(repoDir, slug)

	var gotSlug, gotPath string
	orig := tmuxLauncher
	t.Cleanup(func() { tmuxLauncher = orig })
	tmuxLauncher = func(s, p string, _ io.Reader, _, _ io.Writer) error {
		gotSlug, gotPath = s, p
		return nil
	}

	var buf bytes.Buffer
	if err := runTmux(slug, repoDir, &buf, &buf, &buf); err != nil {
		t.Fatalf("runTmux: %v", err)
	}
	if gotSlug != slug {
		t.Errorf("session slug = %q, want %q", gotSlug, slug)
	}
	if gotPath != wantPath {
		t.Errorf("session cwd = %q, want %q", gotPath, wantPath)
	}
}

func TestRunTmuxUnknownSlugErrors(t *testing.T) {
	repoDir := initTestRepo(t)

	called := false
	orig := tmuxLauncher
	t.Cleanup(func() { tmuxLauncher = orig })
	tmuxLauncher = func(string, string, io.Reader, io.Writer, io.Writer) error {
		called = true
		return nil
	}

	var buf bytes.Buffer
	if err := runTmux("nope", repoDir, &buf, &buf, &buf); err == nil {
		t.Errorf("expected error for unknown slug")
	}
	if called {
		t.Errorf("launcher should not run when the slug fails to resolve")
	}
}
