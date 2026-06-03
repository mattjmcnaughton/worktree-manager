package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/mattjmcnaughton/worktree-manager/internal/git"
	"github.com/mattjmcnaughton/worktree-manager/internal/task"
)

func newCdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cd <slug>",
		Short: "Print the absolute path of a managed worktree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			return runCd(args[0], cwd, cmd.OutOrStdout())
		},
	}
	return cmd
}

func runCd(slug, startDir string, out io.Writer) error {
	repo, err := git.Open(startDir)
	if err != nil {
		return err
	}
	store := task.NewStore(repo.MainWorktreePath())
	rec, err := store.Get(slug)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, rec.WorktreePath)
	return nil
}
