package main

import (
	"fmt"
	"os"

	"github.com/mattjmcnaughton/worktree-manager/internal/cli"
)

func main() {
	if err := cli.NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "worktree-manager:", err)
		os.Exit(1)
	}
}
