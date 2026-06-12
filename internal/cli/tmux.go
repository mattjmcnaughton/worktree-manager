package cli

import (
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// tmuxLauncher runs `tmux new-session -A -s <slug> -c <wtPath>`; swapped out
// in tests so the suite never tries to attach to a real tmux server.
var tmuxLauncher = func(slug, wtPath string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.Command("tmux", "new-session", "-A", "-s", slug, "-c", wtPath)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func newTmuxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tmux <slug>",
		Short: "Attach to (or create) a tmux session for a managed worktree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			return runTmux(args[0], cwd, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	return cmd
}

func runTmux(slug, startDir string, stdin io.Reader, stdout, stderr io.Writer) error {
	wtPath, err := resolveWorktreePath(slug, startDir)
	if err != nil {
		return err
	}
	return tmuxLauncher(slug, wtPath, stdin, stdout, stderr)
}
