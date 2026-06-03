package git

import "strings"

// Worktree represents a git worktree with basic metadata.
type Worktree struct {
	Path   string
	Branch string
	HEAD   string
	IsMain bool
}

func parseWorktreeList(output string) []Worktree {
	var worktrees []Worktree
	var current *Worktree

	flush := func() {
		if current != nil {
			worktrees = append(worktrees, *current)
			current = nil
		}
	}

	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimRight(raw, "\r")
		if line == "" {
			flush()
			continue
		}
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			current = &Worktree{Path: strings.TrimPrefix(line, "worktree ")}
		case current == nil:
			// ignore stray lines before first "worktree" header
		case strings.HasPrefix(line, "HEAD "):
			current.HEAD = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch refs/heads/"):
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		case line == "detached":
			// keep Branch empty
		}
	}
	flush()
	return worktrees
}
