//go:build integration

package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunCdPrintsAbsoluteWorktreePath(t *testing.T) {
	repoDir := setupRepoWithCreate(t, "")
	slug := "demo-task"
	want := worktreePathFor(repoDir, slug)

	var buf bytes.Buffer
	if err := runCd(slug, repoDir, &buf); err != nil {
		t.Fatalf("runCd: %v", err)
	}
	got := strings.TrimRight(buf.String(), "\n")
	if got != want {
		t.Errorf("runCd output = %q, want %q", got, want)
	}
	if strings.Count(buf.String(), "\n") > 1 {
		t.Errorf("expected single line output; got %q", buf.String())
	}
}

func TestRunCdUnknownSlugErrors(t *testing.T) {
	repoDir := initTestRepo(t)

	var buf bytes.Buffer
	err := runCd("no-such-slug", repoDir, &buf)
	if err == nil {
		t.Fatalf("expected error for unknown slug")
	}
	if buf.Len() != 0 {
		t.Errorf("expected stdout to stay clean on error; got %q", buf.String())
	}
}
