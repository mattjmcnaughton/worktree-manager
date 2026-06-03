package cli

import (
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/mattjmcnaughton/worktree-manager/internal/git"
	"github.com/mattjmcnaughton/worktree-manager/internal/task"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List managed worktrees",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			return runList(cwd, cmd.OutOrStdout())
		},
	}
	return cmd
}

func runList(startDir string, out io.Writer) error {
	repo, err := git.Open(startDir)
	if err != nil {
		return err
	}
	store := task.NewStore(repo.MainWorktreePath())
	tasks, err := store.List()
	if err != nil {
		return err
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Slug < tasks[j].Slug })

	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SLUG\tBRANCH\tWORKTREE")
	for _, t := range tasks {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", t.Slug, t.Branch, t.WorktreePath)
	}
	return tw.Flush()
}
