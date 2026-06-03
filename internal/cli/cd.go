package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

func newCdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cd <slug>",
		Short: "Print the absolute path of a managed worktree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("not implemented")
		},
	}
	return cmd
}
