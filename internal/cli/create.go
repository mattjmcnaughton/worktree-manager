package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var (
		slug      string
		base      string
		exec      string
		agentic   bool
		noAgentic bool
	)

	cmd := &cobra.Command{
		Use:   "create [task]",
		Short: "Create a worktree task",
		Long: "Create a new worktree task from a free-text description, ticket ID, " +
			"GitHub ref, or explicit slug.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("not implemented")
		},
	}

	cmd.Flags().StringVar(&slug, "slug", "", "Explicit slug (skips normalization)")
	cmd.Flags().StringVar(&base, "base", "", "Base branch to fork the worktree from")
	cmd.Flags().StringVar(&exec, "exec", "", "Command to run after worktree creation")
	cmd.Flags().BoolVar(&agentic, "agentic", false, "Create .agentic/<slug>/ workspace")
	cmd.Flags().BoolVar(&noAgentic, "no-agentic", false, "Skip .agentic/<slug>/ workspace")

	return cmd
}
