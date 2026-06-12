package cli

import (
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// shellLauncher runs an interactive shell in cwd; swapped out in tests.
var shellLauncher = func(shell, cwd string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.Command(shell)
	cmd.Dir = cwd
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func newShellCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shell <slug>",
		Short: "Open an interactive shell inside a managed worktree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			return runShell(args[0], cwd, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	return cmd
}

func runShell(slug, startDir string, stdin io.Reader, stdout, stderr io.Writer) error {
	wtPath, err := resolveWorktreePath(slug, startDir)
	if err != nil {
		return err
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	return shellLauncher(shell, wtPath, stdin, stdout, stderr)
}
