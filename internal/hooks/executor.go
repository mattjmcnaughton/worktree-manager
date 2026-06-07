// Package hooks runs configured lifecycle hooks for create/delete.
package hooks

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mattjmcnaughton/worktree-manager/internal/config"
)

// Phase identifies which hook bucket to run.
type Phase string

const (
	PhasePreCreate  Phase = "pre_create"
	PhasePostCreate Phase = "post_create"
	PhasePreDelete  Phase = "pre_delete"
	PhasePostDelete Phase = "post_delete"
)

// Context carries the per-run identifiers consumed by hooks.
type Context struct {
	Slug          string
	Branch        string
	Base          string
	MainPath      string
	WorktreePath  string
	WorkspacePath string
}

// Executor runs hooks for a single repo+config.
type Executor struct {
	cfg      *config.Config
	repoRoot string
}

// NewExecutor binds an Executor to a config and its repo root.
func NewExecutor(cfg *config.Config, repoRoot string) *Executor {
	return &Executor{cfg: cfg, repoRoot: repoRoot}
}

// Run executes every hook for phase against ctx, streaming progress to w.
// Errors from optional:true hooks are logged and swallowed.
func (e *Executor) Run(phase Phase, ctx Context, w io.Writer) error {
	if e.cfg == nil {
		return nil
	}
	hooks := e.pick(phase)
	total := len(hooks)
	if total == 0 {
		return nil
	}
	for i, h := range hooks {
		label := h.Name
		if label == "" {
			label = fmt.Sprintf("%s/%d", h.Type, i+1)
		}
		fmt.Fprintf(w, "-> %s hook %d/%d (%s)\n", phase, i+1, total, label)
		if err := e.runOne(phase, h, ctx, w); err != nil {
			if h.Optional {
				fmt.Fprintf(w, "   warn: %v (optional, continuing)\n", err)
				continue
			}
			return fmt.Errorf("%s hook %d (%s): %w", phase, i+1, label, err)
		}
	}
	return nil
}

func (e *Executor) pick(p Phase) []config.Hook {
	switch p {
	case PhasePreCreate:
		return e.cfg.Hooks.PreCreate
	case PhasePostCreate:
		return e.cfg.Hooks.PostCreate
	case PhasePreDelete:
		return e.cfg.Hooks.PreDelete
	case PhasePostDelete:
		return e.cfg.Hooks.PostDelete
	}
	return nil
}

func (e *Executor) runOne(phase Phase, h config.Hook, ctx Context, w io.Writer) error {
	switch h.Type {
	case config.HookTypeCopy:
		return e.runCopy(h, ctx, w)
	case config.HookTypeSymlink:
		return e.runSymlink(h, ctx, w)
	case config.HookTypeCommand:
		return e.runCommand(phase, h, ctx, w)
	default:
		return fmt.Errorf("unknown hook type %q", h.Type)
	}
}

func (e *Executor) runCopy(h config.Hook, ctx Context, w io.Writer) error {
	src := absJoin(e.repoRoot, h.From)
	dst := absJoin(ctx.WorktreePath, h.To)
	if dst == "" {
		dst = absJoin(ctx.WorktreePath, h.From)
	}
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("source %s: %w", src, err)
	}
	if info.IsDir() {
		if err := copyDir(src, dst); err != nil {
			return fmt.Errorf("copy dir %s -> %s: %w", src, dst, err)
		}
		fmt.Fprintf(w, "   copy %s -> %s (dir)\n", src, dst)
		return nil
	}
	if err := copyFile(src, dst, info.Mode().Perm()); err != nil {
		return err
	}
	fmt.Fprintf(w, "   copy %s -> %s\n", src, dst)
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(dst), err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.WriteFile(dst, data, mode); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, srcInfo.Mode().Perm()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		sp := filepath.Join(src, entry.Name())
		dp := filepath.Join(dst, entry.Name())
		switch {
		case entry.IsDir():
			if err := copyDir(sp, dp); err != nil {
				return err
			}
		case entry.Type()&os.ModeSymlink != 0:
			target, err := os.Readlink(sp)
			if err != nil {
				return fmt.Errorf("readlink %s: %w", sp, err)
			}
			if err := os.Symlink(target, dp); err != nil {
				return fmt.Errorf("symlink %s -> %s: %w", dp, target, err)
			}
		default:
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if err := copyFile(sp, dp, info.Mode().Perm()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Executor) runSymlink(h config.Hook, ctx Context, w io.Writer) error {
	src := absJoin(e.repoRoot, h.From)
	dst := absJoin(ctx.WorktreePath, h.To)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(dst), err)
	}
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("destination %s already exists", dst)
	}
	if err := os.Symlink(src, dst); err != nil {
		return fmt.Errorf("symlink %s -> %s: %w", dst, src, err)
	}
	fmt.Fprintf(w, "   symlink %s -> %s\n", dst, src)
	return nil
}

func (e *Executor) runCommand(phase Phase, h config.Hook, ctx Context, w io.Writer) error {
	workDir := resolveWorkDir(h.WorkDir, ctx)
	cmd := exec.Command("sh", "-c", h.Command)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(),
		"WORKTREE_MANAGER_REPO_ROOT="+e.repoRoot,
		"WORKTREE_MANAGER_MAIN_WORKTREE="+ctx.MainPath,
		"WORKTREE_MANAGER_WORKTREE_PATH="+ctx.WorktreePath,
		"WORKTREE_MANAGER_WORKSPACE_PATH="+ctx.WorkspacePath,
		"WORKTREE_MANAGER_BRANCH="+ctx.Branch,
		"WORKTREE_MANAGER_SLUG="+ctx.Slug,
		"WORKTREE_MANAGER_BASE="+ctx.Base,
		"WORKTREE_MANAGER_PHASE="+string(phase),
	)
	for k, v := range h.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Stdout = w
	cmd.Stderr = w
	fmt.Fprintf(w, "   exec: %s\n", h.Command)
	return cmd.Run()
}

func resolveWorkDir(workDir string, ctx Context) string {
	switch workDir {
	case "", config.WorkDirWorktree:
		return ctx.WorktreePath
	case config.WorkDirMain:
		return ctx.MainPath
	}
	if filepath.IsAbs(workDir) {
		return workDir
	}
	return filepath.Join(ctx.WorktreePath, workDir)
}

func absJoin(base, rel string) string {
	if rel == "" {
		return ""
	}
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(base, strings.TrimPrefix(rel, "./"))
}
