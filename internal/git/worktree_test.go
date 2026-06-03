package git

import (
	"testing"
)

func TestParseWorktreeListPopulatesFields(t *testing.T) {
	output := "worktree /home/u/repo\n" +
		"HEAD abc123\n" +
		"branch refs/heads/main\n" +
		"\n" +
		"worktree /home/u/repo/.worktrees/repo-feature\n" +
		"HEAD def456\n" +
		"branch refs/heads/u/feature\n" +
		"\n" +
		"worktree /home/u/repo/.worktrees/detached\n" +
		"HEAD ghi789\n" +
		"detached\n"

	got := parseWorktreeList(output)
	if len(got) != 3 {
		t.Fatalf("expected 3 worktrees, got %d: %+v", len(got), got)
	}

	if got[0].Path != "/home/u/repo" || got[0].Branch != "main" || got[0].HEAD != "abc123" {
		t.Errorf("entry 0 = %+v", got[0])
	}
	if got[1].Path != "/home/u/repo/.worktrees/repo-feature" || got[1].Branch != "u/feature" {
		t.Errorf("entry 1 = %+v", got[1])
	}
	if got[2].Path != "/home/u/repo/.worktrees/detached" || got[2].Branch != "" || got[2].HEAD != "ghi789" {
		t.Errorf("entry 2 = %+v", got[2])
	}
}

func TestParseWorktreeListEmpty(t *testing.T) {
	if got := parseWorktreeList(""); len(got) != 0 {
		t.Errorf("expected 0 worktrees, got %d", len(got))
	}
}
