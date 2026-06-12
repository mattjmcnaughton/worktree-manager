package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mattjmcnaughton/worktree-manager/internal/config"
	"github.com/mattjmcnaughton/worktree-manager/internal/git"
	"github.com/mattjmcnaughton/worktree-manager/internal/hooks"
	"github.com/mattjmcnaughton/worktree-manager/internal/resolver"
	"github.com/mattjmcnaughton/worktree-manager/internal/slug"
	"github.com/mattjmcnaughton/worktree-manager/internal/task"
)

type createOpts struct {
	Task      string
	Slug      string
	Base      string
	Exec      string
	Agentic   bool
	NoAgentic bool
}

func newCreateCmd() *cobra.Command {
	opts := createOpts{}

	cmd := &cobra.Command{
		Use:   "create [task]",
		Short: "Create a worktree task",
		Long: "Create a new worktree task from a free-text description, ticket ID, " +
			"GitHub ref, or explicit slug.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Task = args[0]
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			return runCreate(opts, cwd, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVar(&opts.Slug, "slug", "", "Explicit slug (skips normalization)")
	cmd.Flags().StringVar(&opts.Base, "base", "", "Base branch to fork the worktree from")
	cmd.Flags().StringVar(&opts.Exec, "exec", "", "Command to run in the worktree after creation")
	cmd.Flags().BoolVar(&opts.Agentic, "agentic", false, "Create .agentic/<slug>/ workspace")
	cmd.Flags().BoolVar(&opts.NoAgentic, "no-agentic", false, "Skip .agentic/<slug>/ workspace")

	return cmd
}

func runCreate(opts createOpts, startDir string, out io.Writer) error {
	repo, err := git.Open(startDir)
	if err != nil {
		return err
	}
	repoRoot := repo.MainWorktreePath()

	cfg, err := loadRepoConfig(repoRoot)
	if err != nil {
		return err
	}
	if cfg.Defaults.User == "" {
		if name := resolveSystemUser(); name != "" {
			if normalized, nerr := slug.Normalize(name); nerr == nil {
				cfg.Defaults.User = normalized
			}
		}
	}

	taskSlug, err := pickSlug(opts)
	if err != nil {
		return err
	}

	res, err := resolver.Resolve(cfg, resolver.Inputs{
		Slug:            taskSlug,
		RepoName:        repo.RepoName(),
		RepoRoot:        repoRoot,
		BaseOverride:    opts.Base,
		AgenticOverride: agenticOverride(opts),
	})
	if err != nil {
		return err
	}

	store := task.NewStore(repoRoot)

	executor := hooks.NewExecutor(cfg, repoRoot)
	hookCtx := hooks.Context{
		Slug:          res.Slug,
		Branch:        res.Branch,
		Base:          res.Base,
		MainPath:      repoRoot,
		WorktreePath:  res.WorktreePath,
		WorkspacePath: res.WorkspacePath,
	}

	if err := executor.Run(hooks.PhasePreCreate, hookCtx, out); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(res.WorktreePath), 0o755); err != nil {
		return fmt.Errorf("prepare worktree parent: %w", err)
	}

	if err := repo.AddWorktree(res.WorktreePath, res.Branch, res.Base); err != nil {
		return err
	}

	if res.WorkspacePath != "" {
		if err := os.MkdirAll(res.WorkspacePath, 0o755); err != nil {
			return fmt.Errorf("create workspace dir: %w", err)
		}
	}

	if err := executor.Run(hooks.PhasePostCreate, hookCtx, out); err != nil {
		return err
	}

	rec := &task.Task{
		Slug:          res.Slug,
		Branch:        res.Branch,
		Base:          res.Base,
		WorktreePath:  res.WorktreePath,
		WorkspacePath: res.WorkspacePath,
	}
	if err := store.Save(rec); err != nil {
		return err
	}

	printCreateSummary(out, res)
	return nil
}

func pickSlug(opts createOpts) (string, error) {
	if opts.Slug != "" {
		if err := slug.Validate(opts.Slug); err != nil {
			return "", err
		}
		return opts.Slug, nil
	}
	if opts.Task == "" {
		return "", errors.New("a task description or --slug is required")
	}
	return slug.Normalize(opts.Task)
}

func agenticOverride(opts createOpts) *bool {
	if opts.NoAgentic {
		b := false
		return &b
	}
	if opts.Agentic {
		b := true
		return &b
	}
	return nil
}

// resolveSystemUser returns the OS-level username. $USER wins so callers can
// override (containers without /etc/passwd, shells with a custom value); falls
// back to os/user.Current() when the env var is unset.
func resolveSystemUser() string {
	if env := os.Getenv("USER"); env != "" {
		return env
	}
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return ""
}

func loadRepoConfig(repoRoot string) (*config.Config, error) {
	repoPath := filepath.Join(repoRoot, ".worktree-manager.yml")
	globalPath := ""
	if dir, err := globalConfigDir(); err == nil {
		globalPath = filepath.Join(dir, "worktree-manager", "config.yml")
	}
	return config.Load(globalPath, repoPath, repoRoot)
}

// globalConfigDir resolves the directory holding the global config file.
// $XDG_CONFIG_HOME wins (per the XDG Base Directory spec, cross-platform);
// otherwise we fall back to ~/.config so users see the same path on Linux and
// macOS without needing to touch ~/Library/Application Support.
func globalConfigDir() (string, error) {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}

func printCreateSummary(out io.Writer, res resolver.Resolved) {
	fmt.Fprintln(out, "Created worktree task:")
	fmt.Fprintf(out, "  slug:      %s\n", res.Slug)
	fmt.Fprintf(out, "  branch:    %s\n", res.Branch)
	fmt.Fprintf(out, "  base:      %s\n", res.Base)
	fmt.Fprintf(out, "  worktree:  %s\n", res.WorktreePath)
	if res.WorkspacePath != "" {
		fmt.Fprintf(out, "  workspace: %s\n", res.WorkspacePath)
	}
}
