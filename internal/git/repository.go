// Package git wraps shell-outs to the system git binary.
package git

import (
	stdErrors "errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Repository is anchored at the main worktree path.
type Repository struct {
	mainWorktreePath string
}

// Open locates the main worktree starting from any path inside a git repo.
func Open(startDir string) (*Repository, error) {
	out, err := runGit(startDir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return nil, fmt.Errorf("not inside a git repository (looked in %s): %w", startDir, err)
	}
	commonDir := strings.TrimSpace(out)
	mainPath := commonDir
	if strings.HasSuffix(commonDir, string(filepath.Separator)+".git") || filepath.Base(commonDir) == ".git" {
		mainPath = filepath.Dir(commonDir)
	}
	abs, err := filepath.Abs(mainPath)
	if err != nil {
		return nil, fmt.Errorf("resolve main worktree path: %w", err)
	}
	return &Repository{mainWorktreePath: abs}, nil
}

// MainWorktreePath returns the absolute path to the main worktree.
func (r *Repository) MainWorktreePath() string {
	return r.mainWorktreePath
}

// RepoName returns the directory name of the main worktree.
func (r *Repository) RepoName() string {
	return filepath.Base(r.mainWorktreePath)
}

// UserName returns the configured git user.name (may be empty).
func (r *Repository) UserName() (string, error) {
	out, err := runGit(r.mainWorktreePath, "config", "--get", "user.name")
	if err != nil {
		var exitErr *exec.ExitError
		if stdErrors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// ListWorktrees parses `git worktree list --porcelain`.
func (r *Repository) ListWorktrees() ([]Worktree, error) {
	out, err := runGit(r.mainWorktreePath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}
	wts := parseWorktreeList(out)
	if len(wts) > 0 {
		wts[0].IsMain = true
	}
	return wts, nil
}

// AddWorktree creates a worktree at path on a new branch forked from base.
func (r *Repository) AddWorktree(path, branch, base string) error {
	args := []string{"worktree", "add", "-b", branch, path}
	if base != "" {
		args = append(args, base)
	}
	if _, err := runGit(r.mainWorktreePath, args...); err != nil {
		return fmt.Errorf("git worktree add: %w", err)
	}
	return nil
}

// RemoveWorktree runs `git worktree remove [--force] path`.
func (r *Repository) RemoveWorktree(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	if _, err := runGit(r.mainWorktreePath, args...); err != nil {
		return fmt.Errorf("git worktree remove: %w", err)
	}
	return nil
}

// BranchExists checks for a local branch.
func (r *Repository) BranchExists(name string) (bool, error) {
	if err := validateRef(name); err != nil {
		return false, err
	}
	_, err := runGit(r.mainWorktreePath, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if stdErrors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

// DeleteBranch removes a local branch. force toggles -D vs -d.
func (r *Repository) DeleteBranch(name string, force bool) error {
	if err := validateRef(name); err != nil {
		return err
	}
	flag := "-d"
	if force {
		flag = "-D"
	}
	if _, err := runGit(r.mainWorktreePath, "branch", flag, name); err != nil {
		return fmt.Errorf("git branch %s %s: %w", flag, name, err)
	}
	return nil
}

// HasUncommittedChanges reports whether `git status --porcelain` in worktreePath is non-empty.
func (r *Repository) HasUncommittedChanges(worktreePath string) (bool, error) {
	out, err := runGit(worktreePath, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}
	return strings.TrimSpace(out) != "", nil
}

// HasUnpushedCommits reports whether commits exist ahead of upstream.
// When no upstream is configured, falls back to commits between fallbackRef
// and HEAD; if fallbackRef is empty, returns true conservatively.
func (r *Repository) HasUnpushedCommits(worktreePath, fallbackRef string) (bool, error) {
	if _, err := runGit(worktreePath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}"); err == nil {
		out, err := runGit(worktreePath, "log", "@{upstream}..HEAD", "--oneline")
		if err != nil {
			return false, fmt.Errorf("git log @{upstream}..HEAD: %w", err)
		}
		return strings.TrimSpace(out) != "", nil
	}
	if fallbackRef == "" {
		return true, nil
	}
	out, err := runGit(worktreePath, "log", fallbackRef+"..HEAD", "--oneline")
	if err != nil {
		return true, nil
	}
	return strings.TrimSpace(out) != "", nil
}

func validateRef(name string) error {
	if name == "" {
		return stdErrors.New("ref name is empty")
	}
	if strings.ContainsAny(name, "\n\r") {
		return fmt.Errorf("ref %q contains newline", name)
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("ref %q contains '..'", name)
	}
	return nil
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if stdErrors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return string(out), fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return string(out), err
	}
	return string(out), nil
}
