package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/mattjmcnaughton/worktree-manager/internal/git"
	"github.com/mattjmcnaughton/worktree-manager/internal/task"
)

func newPwdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pwd <slug>",
		Short: "Print the absolute path of a managed worktree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			return runPwd(args[0], cwd, cmd.OutOrStdout())
		},
	}
	return cmd
}

func runPwd(slug, startDir string, out io.Writer) error {
	path, err := resolveWorktreePath(slug, startDir)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, path)
	return nil
}

// resolveWorktreePath maps a slug to its managed worktree path by looking the
// task up in the on-disk store anchored at startDir's enclosing repo.
func resolveWorktreePath(slug, startDir string) (string, error) {
	repo, err := git.Open(startDir)
	if err != nil {
		return "", err
	}
	store := task.NewStore(repo.MainWorktreePath())
	rec, err := store.Get(slug)
	if err != nil {
		return "", err
	}
	return rec.WorktreePath, nil
}
