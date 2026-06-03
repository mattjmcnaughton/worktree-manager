//go:build integration

package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunListShowsCreatedTasks(t *testing.T) {
	repoDir := setupRepoWithCreate(t, "")

	// Add a second task so we exercise multi-row output.
	var crBuf bytes.Buffer
	if err := runCreate(createOpts{Task: "another task"}, repoDir, &crBuf); err != nil {
		t.Fatalf("second runCreate: %v", err)
	}

	var buf bytes.Buffer
	if err := runList(repoDir, &buf); err != nil {
		t.Fatalf("runList: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"SLUG", "BRANCH", "WORKTREE",
		"demo-task", "tester/demo-task",
		"another-task", "tester/another-task",
		worktreePathFor(repoDir, "demo-task"),
		worktreePathFor(repoDir, "another-task"),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("list output missing %q; got:\n%s", want, out)
		}
	}
}

func TestRunListEmptyHasHeaderOnly(t *testing.T) {
	repoDir := initTestRepo(t)

	var buf bytes.Buffer
	if err := runList(repoDir, &buf); err != nil {
		t.Fatalf("runList: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "SLUG") {
		t.Errorf("expected header in output; got: %q", out)
	}
	// No task rows: lines after header should be empty.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) > 1 {
		t.Errorf("expected only header line; got %d lines: %q", len(lines), out)
	}
}
