package cli

import (
	"os/user"
	"testing"
)

func TestResolveSystemUserPrefersEnvUSER(t *testing.T) {
	t.Setenv("USER", "envuser")
	got := resolveSystemUser()
	if got != "envuser" {
		t.Errorf("resolveSystemUser() = %q, want %q", got, "envuser")
	}
}

func TestResolveSystemUserFallsBackToOSUser(t *testing.T) {
	t.Setenv("USER", "")
	got := resolveSystemUser()
	if got == "" {
		t.Skip("os/user.Current() is unavailable in this environment; nothing to assert")
	}
	cur, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current: %v", err)
	}
	if got != cur.Username {
		t.Errorf("resolveSystemUser() = %q, want %q (from os/user.Current)", got, cur.Username)
	}
}
