package cli

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mattjmcnaughton/worktree-manager/internal/version"
)

func NewRoot() *cobra.Command {
	var logLevel string

	root := &cobra.Command{
		Use:           "worktree-manager",
		Short:         "A Go CLI for managing Git worktrees",
		Version:       version.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return setupLogging(logLevel)
		},
	}

	root.PersistentFlags().StringVar(
		&logLevel,
		"log-level",
		"info",
		"Log level (debug, info, warn, error)",
	)

	viper.SetEnvPrefix("WORKTREE_MANAGER")
	viper.AutomaticEnv()

	root.AddCommand(
		newCreateCmd(),
		newDeleteCmd(),
		newListCmd(),
		newCdCmd(),
	)

	return root
}

func setupLogging(level string) error {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		l = slog.LevelInfo
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: l})
	slog.SetDefault(slog.New(handler))
	return nil
}
