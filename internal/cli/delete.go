package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/mattjmcnaughton/worktree-manager/internal/git"
	"github.com/mattjmcnaughton/worktree-manager/internal/hooks"
	"github.com/mattjmcnaughton/worktree-manager/internal/task"
)

type deleteOpts struct {
	Slug        string
	WithBranch  bool
	Force       bool
	ForceBranch bool
}

func newDeleteCmd() *cobra.Command {
	opts := deleteOpts{}

	cmd := &cobra.Command{
		Use:   "delete [slug-or-path]",
		Short: "Delete a worktree task",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Slug = args[0]
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			return runDelete(opts, cwd, cmd.OutOrStdout())
		},
	}

	cmd.Flags().BoolVar(&opts.WithBranch, "with-branch", false, "Also delete the branch")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Bypass safety checks for dirty/unpushed work")
	cmd.Flags().BoolVar(&opts.ForceBranch, "force-branch", false, "Force-delete the branch (-D)")

	return cmd
}

func runDelete(opts deleteOpts, startDir string, out io.Writer) error {
	if opts.Slug == "" {
		return errors.New("a slug is required")
	}

	repo, err := git.Open(startDir)
	if err != nil {
		return err
	}
	repoRoot := repo.MainWorktreePath()

	cfg, err := loadRepoConfig(repoRoot)
	if err != nil {
		return err
	}

	store := task.NewStore(repoRoot)
	rec, err := store.Get(opts.Slug)
	if err != nil {
		return err
	}

	if rec.WorktreePath == repoRoot {
		return fmt.Errorf("refusing to delete the main worktree at %s", repoRoot)
	}

	if !opts.Force {
		dirty, err := repo.HasUncommittedChanges(rec.WorktreePath)
		if err != nil {
			return err
		}
		if dirty {
			return fmt.Errorf("worktree %s has uncommitted changes; pass --force to override", rec.WorktreePath)
		}
		unpushed, err := repo.HasUnpushedCommits(rec.WorktreePath, rec.Base)
		if err != nil {
			return err
		}
		if unpushed {
			return fmt.Errorf("worktree %s has unpushed commits; pass --force to override", rec.WorktreePath)
		}
	}

	executor := hooks.NewExecutor(cfg, repoRoot)
	hookCtx := hooks.Context{
		Slug:          rec.Slug,
		Branch:        rec.Branch,
		Base:          rec.Base,
		MainPath:      repoRoot,
		WorktreePath:  rec.WorktreePath,
		WorkspacePath: rec.WorkspacePath,
	}

	if err := executor.Run(hooks.PhasePreDelete, hookCtx, out); err != nil {
		return err
	}

	if err := repo.RemoveWorktree(rec.WorktreePath, opts.Force); err != nil {
		return err
	}

	if opts.WithBranch {
		if err := repo.DeleteBranch(rec.Branch, opts.ForceBranch); err != nil {
			return err
		}
	}

	if err := executor.Run(hooks.PhasePostDelete, hookCtx, out); err != nil {
		return err
	}

	if err := store.Delete(rec.Slug); err != nil {
		return err
	}

	fmt.Fprintf(out, "Deleted worktree task %s\n", rec.Slug)
	return nil
}
