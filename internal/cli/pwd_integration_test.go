//go:build integration

package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunPwdPrintsAbsoluteWorktreePath(t *testing.T) {
	repoDir := setupRepoWithCreate(t, "")
	slug := "demo-task"
	want := worktreePathFor(repoDir, slug)

	var buf bytes.Buffer
	if err := runPwd(slug, repoDir, &buf); err != nil {
		t.Fatalf("runPwd: %v", err)
	}
	got := strings.TrimRight(buf.String(), "\n")
	if got != want {
		t.Errorf("runPwd output = %q, want %q", got, want)
	}
	if strings.Count(buf.String(), "\n") > 1 {
		t.Errorf("expected single line output; got %q", buf.String())
	}
}

func TestRunPwdUnknownSlugErrors(t *testing.T) {
	repoDir := initTestRepo(t)

	var buf bytes.Buffer
	err := runPwd("no-such-slug", repoDir, &buf)
	if err == nil {
		t.Fatalf("expected error for unknown slug")
	}
	if buf.Len() != 0 {
		t.Errorf("expected stdout to stay clean on error; got %q", buf.String())
	}
}
