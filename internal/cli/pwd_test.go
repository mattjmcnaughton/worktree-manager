package cli

import "testing"

func TestNewPwdCmdRegistersExpectedSurface(t *testing.T) {
	cmd := newPwdCmd()
	if cmd.Use != "pwd <slug>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "pwd <slug>")
	}
	if cmd.Args == nil {
		t.Errorf("expected Args validator to require exactly one argument")
	}
}

func TestNewShellCmdRegistersExpectedSurface(t *testing.T) {
	cmd := newShellCmd()
	if cmd.Use != "shell <slug>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "shell <slug>")
	}
	if cmd.Args == nil {
		t.Errorf("expected Args validator to require exactly one argument")
	}
}

func TestNewTmuxCmdRegistersExpectedSurface(t *testing.T) {
	cmd := newTmuxCmd()
	if cmd.Use != "tmux <slug>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "tmux <slug>")
	}
	if cmd.Args == nil {
		t.Errorf("expected Args validator to require exactly one argument")
	}
}
