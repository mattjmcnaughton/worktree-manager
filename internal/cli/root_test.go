package cli

import (
	"sort"
	"testing"
)

func TestRootRegistersExpectedSubcommands(t *testing.T) {
	root := NewRoot()

	got := make(map[string]bool)
	for _, c := range root.Commands() {
		got[c.Name()] = true
	}

	want := []string{"create", "delete", "list", "cd"}
	for _, name := range want {
		if !got[name] {
			t.Errorf("expected subcommand %q to be registered; got: %v", name, sortedKeys(got))
		}
	}

	if got["example"] {
		t.Errorf("expected example command to be removed; still registered")
	}
}

func TestDeferredCommandsAreNotRegistered(t *testing.T) {
	root := NewRoot()

	got := make(map[string]bool)
	for _, c := range root.Commands() {
		got[c.Name()] = true
	}

	for _, deferred := range []string{"status", "exec"} {
		if got[deferred] {
			t.Errorf("command %q should not be exposed in this slice; remove until implemented", deferred)
		}
	}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
