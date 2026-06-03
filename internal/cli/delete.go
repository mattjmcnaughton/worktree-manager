package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var (
		withBranch  bool
		force       bool
		forceBranch bool
	)

	cmd := &cobra.Command{
		Use:   "delete [slug-or-path]",
		Short: "Delete a worktree task",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("not implemented")
		},
	}

	cmd.Flags().BoolVar(&withBranch, "with-branch", false, "Also delete the branch")
	cmd.Flags().BoolVar(&force, "force", false, "Bypass safety checks for dirty/unpushed work")
	cmd.Flags().BoolVar(&forceBranch, "force-branch", false, "Force-delete the branch (-D)")

	return cmd
}
